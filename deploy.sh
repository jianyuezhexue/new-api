#!/usr/bin/env bash
# ================================================================
# new-api 零停机蓝绿部署脚本
#
# 工作原理：
#   blue / green 两个容器实例，同一时刻只有一个在接收流量。
#   部署时：构建新镜像 → 启动备用容器 → 等待就绪 → 切换流量 → 停止旧容器
#
# 用法：
#   bash deploy.sh
#
# 前置条件：
#   - 本地已执行 build.sh，产物已上传到服务器：
#     release/server（后端二进制）
#     new-api-docs/docs-bundle.tar.xz（文档站点）
#   - docker compose
#   - nginx-proxy-manager 已配置好 proxy host（指向 new-api-blue 或 new-api-green）
#
# 注意：永远不在服务器上构建，构建在本地完成，服务器只负责部署。
# ================================================================

set -euo pipefail

# ── 颜色 ──
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

log()    { echo -e "${GREEN}[$(date '+%H:%M:%S')]${NC} $*"; }
warn()   { echo -e "${YELLOW}[$(date '+%H:%M:%S')] WARN${NC}  $*"; }
err()    { echo -e "${RED}[$(date '+%H:%M:%S')] ERROR${NC} $*"; }
header() { echo -e "\n${CYAN}━━━ $* ━━━${NC}"; }
step()   { echo -e "${BLUE}  ➜${NC} $*"; }

# ── 配置 ──
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

BLUE_SVC="new-api-blue"
GREEN_SVC="new-api-green"
NPM_CONTAINER="nginx-proxy-manager"
PROXY_CONF="/data/nginx/proxy_host/1.conf"
NPM_DB="/data/database.sqlite"
PROXY_HOST_ID="1"                       # npm proxy_host 表的 id
HEALTH_RETRIES=30                       # 30 × 10s = 最多等 5 分钟
HEALTH_INTERVAL=10                      # 健康检查间隔（秒）
VERIFY_RETRIES=3                        # 切换后验证次数

# ── 参数解析 ──
for arg in "$@"; do
  case "$arg" in
    --help|-h)
      echo "用法: bash deploy.sh [--help|-h]"
      echo ""
      echo "蓝绿零停机部署。构建在本地完成，服务器只部署。"
      echo "前置：release/server + new-api-docs/docs-bundle.tar.xz 已就位"
      exit 0
      ;;
  esac
done

# ── 工具函数 ──

container_running() {
  local name="$1"
  docker ps --format '{{.Names}}' 2>/dev/null | grep -qx "$name"
}

container_healthy() {
  local name="$1"
  # 如果容器有 healthcheck，Docker 会设置 health status
  local status
  status=$(docker inspect --format='{{.State.Health.Status}}' "$name" 2>/dev/null || echo "unknown")
  [ "$status" = "healthy" ]
}

get_nginx_upstream() {
  # 返回 nginx 当前指向的容器名（如 new-api-blue, new-api-green, new-api）
  docker exec "$NPM_CONTAINER" sh -c \
    "grep 'set \$server' $PROXY_CONF 2>/dev/null" \
    2>/dev/null | \
    sed -E 's/.*set \$server\s+"?([^"; ]+)"?;/\1/' || echo "unknown"
}

wait_for_healthy() {
  local container="$1"
  local retries="$2"
  step "等待 $container 健康检查通过..."

  for i in $(seq 1 "$retries"); do
    local status
    status=$(docker inspect --format='{{.State.Health.Status}}' "$container" 2>/dev/null || echo "starting")

    # 同时手动检查端口是否可达
    local http_code
    http_code=$(docker exec "$container" wget -q -O - http://localhost:3000/api/status 2>/dev/null | \
      python3 -c "import sys,json; d=json.load(sys.stdin); print('ok' if d.get('success') else 'fail')" 2>/dev/null || echo "fail")

    if [ "$status" = "healthy" ] && [ "$http_code" = "ok" ]; then
      log "  ✓ $container 已就绪 (health: $status, api: ok)"
      return 0
    fi

    printf "  [%2d/%2d] health=%s api=%s\n" "$i" "$retries" "$status" "$http_code"
    sleep "$HEALTH_INTERVAL"
  done

  err "$container 健康检查超时（${retries}x${HEALTH_INTERVAL}s）"
  return 1
}

switch_nginx() {
  local target="$1"   # new-api-blue 或 new-api-green
  local old_target
  old_target=$(get_nginx_upstream)

  if [ "$old_target" = "$target" ]; then
    step "nginx 已指向 $target，无需切换"
    return 0
  fi

  step "切换 nginx upstream: $old_target → $target"

  # 1) 更新 nginx 配置文件
  docker exec "$NPM_CONTAINER" sh -c \
    "sed -i 's|set \\\$server .*|set \\\$server         \"$target\";|' $PROXY_CONF" 2>/dev/null

  # 2) 更新 npm SQLite 数据库（保持 UI 一致性）
  local tmp_db="/tmp/npm_deploy_$$.sqlite"
  docker cp "$NPM_CONTAINER:$NPM_DB" "$tmp_db" 2>/dev/null && {
    sqlite3 "$tmp_db" "UPDATE proxy_host SET forward_host='$target' WHERE id=$PROXY_HOST_ID AND is_deleted=0;" 2>/dev/null && \
    docker cp "$tmp_db" "$NPM_CONTAINER:$NPM_DB" 2>/dev/null && \
    step "  npm 数据库已同步" || \
    warn "  npm 数据库更新失败（不影响流量切换，下次通过 UI 保存时会覆盖）"
    rm -f "$tmp_db"
  } || warn "  无法读取 npm 数据库（不影响流量切换）"

  # 3) 热重载 nginx（毫秒级，不中断活跃连接）
  step "  重载 nginx..."
  if docker exec "$NPM_CONTAINER" nginx -s reload 2>/dev/null; then
    log "  ✓ nginx 已重载"
  else
    err "  nginx reload 失败！正在回滚..."
    # 回滚配置文件
    docker exec "$NPM_CONTAINER" sh -c \
      "sed -i 's|set \\\$server .*|set \\\$server         \"$old_target\";|' $PROXY_CONF" 2>/dev/null
    docker exec "$NPM_CONTAINER" nginx -s reload 2>/dev/null || true
    return 1
  fi
}

verify_traffic() {
  local expected="$1"
  step "验证流量是否切到 $expected ..."

  for i in $(seq 1 $VERIFY_RETRIES); do
    local http_code

    # 蓝绿容器不暴露 host 端口，通过 nginx 外部域名验证
    http_code=$(curl -s -o /dev/null -w "%{http_code}" "https://tokens.buildingblock.top/api/status" 2>/dev/null)
    http_code="${http_code:-000}"

    if [ "$http_code" = "200" ]; then
      log "  ✓ 验证通过 (HTTP $http_code)"
      return 0
    fi
    printf "  [%d/%d] HTTP %s，3 秒后重试...\n" "$i" "$VERIFY_RETRIES" "$http_code"
    sleep 3
  done

  err "流量验证失败"
  return 1
}

# ── 主流程 ──

echo ""
echo -e "${CYAN}╔══════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║     new-api 蓝绿部署 (Zero-Downtime)          ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════════════╝${NC}"
echo ""

# ─── Step 0: 拉取最新代码 ───
header "Step 0: 拉取最新代码"
git pull
log "✓ 代码已更新"

# ─── Step 1: 检查产物 ───
header "Step 1: 检查产物"

if [ ! -f release/server ]; then
  err "release/server 不存在，请先在本地执行 build.sh 并将产物上传到服务器"
  exit 1
fi
log "✓ release/server 就位"

if [ ! -f new-api-docs/docs-bundle.tar.xz ]; then
  warn "new-api-docs/docs-bundle.tar.xz 不存在，文档站点将不会更新"
fi

# ─── Step 2: 检测状态 ───
header "Step 2: 检测当前状态"

BLUE_RUNNING=$(container_running "$BLUE_SVC" && echo "yes" || echo "no")
GREEN_RUNNING=$(container_running "$GREEN_SVC" && echo "yes" || echo "no")
OLD_RUNNING=$(container_running "new-api" && echo "yes" || echo "no")
NGINX_UPSTREAM=$(get_nginx_upstream)

echo ""
echo "  ┌──────────────────────────────────────────┐"
printf  "  │ %-20s : %-15s │\n" "new-api-blue" "$([ "$BLUE_RUNNING" = "yes" ] && echo "运行中" || echo "已停止")"
printf  "  │ %-20s : %-15s │\n" "new-api-green" "$([ "$GREEN_RUNNING" = "yes" ] && echo "运行中" || echo "已停止")"
printf  "  │ %-20s : %-15s │\n" "new-api (旧)" "$([ "$OLD_RUNNING" = "yes" ] && echo "运行中" || echo "已停止")"
printf  "  │ %-20s : %-15s │\n" "nginx →" "$NGINX_UPSTREAM"
echo "  └──────────────────────────────────────────┘"
echo ""

# ─── 确定 active / standby 角色 ───
ACTIVE=""
STANDBY=""

if [ "$OLD_RUNNING" = "yes" ]; then
  # ── 首次迁移：旧容器在跑，迁移到蓝绿架构 ──
  warn "检测到旧容器 'new-api' 仍在运行，将迁移到蓝绿架构"
  ACTIVE="new-api"
  STANDBY="$GREEN_SVC"           # 先用 green 接替
  SKIP_STOP_OLD=false            # 我们手动停旧容器，不用 compose

elif [ "$BLUE_RUNNING" = "yes" ] && [ "$GREEN_RUNNING" = "yes" ]; then
  # ── 异常：两个都在跑 ──
  warn "两个容器都在运行（可能是上次部署残留）"
  BLUE_HEALTHY=$(container_healthy "$BLUE_SVC" && echo "yes" || echo "no")
  GREEN_HEALTHY=$(container_healthy "$GREEN_SVC" && echo "yes" || echo "no")

  if [ "$BLUE_HEALTHY" = "yes" ]; then
    ACTIVE="$BLUE_SVC"; STANDBY="$GREEN_SVC"
  elif [ "$GREEN_HEALTHY" = "yes" ]; then
    ACTIVE="$GREEN_SVC"; STANDBY="$BLUE_SVC"
  else
    ACTIVE="$BLUE_SVC"; STANDBY="$GREEN_SVC"   # 都 unhealthy，保守选择
  fi
  log "active=$ACTIVE, standby=$STANDBY (将重建 standby)"

elif [ "$BLUE_RUNNING" = "yes" ]; then
  ACTIVE="$BLUE_SVC"; STANDBY="$GREEN_SVC"

elif [ "$GREEN_RUNNING" = "yes" ]; then
  ACTIVE="$GREEN_SVC"; STANDBY="$BLUE_SVC"

else
  # ── 冷启动：两个都没有 ──
  warn "未检测到运行中的容器，进行冷启动"
  ACTIVE="none"
  STANDBY="$BLUE_SVC"            # 先起 blue
fi

log "当前 active:  ${YELLOW}$ACTIVE${NC}"
log "待启动 standby: ${GREEN}$STANDBY${NC}"

# ─── Step 3: 启动 standby ───
header "Step 3: 启动 standby 容器 ($STANDBY)"

# 先确保 standby 不存在（清理上次残留）
if container_running "$STANDBY"; then
  warn "$STANDBY 已在运行，先停止它（可能版本过旧）"
  docker stop "$STANDBY" 2>/dev/null || true
  docker rm "$STANDBY" 2>/dev/null || true
fi

docker compose up -d "$STANDBY"
log "$STANDBY 容器已启动"

# ─── Step 4: 等待 standby 就绪 ───
header "Step 4: 等待 standby 健康检查通过"
if ! wait_for_healthy "$STANDBY" $HEALTH_RETRIES; then
  err "standby 未能在规定时间内就绪，部署中止"
  warn "当前 active ($ACTIVE) 不受影响，继续提供服务"
  # 清理未就绪的 standby
  docker stop "$STANDBY" 2>/dev/null || true
  docker rm "$STANDBY" 2>/dev/null || true
  exit 1
fi

# ─── Step 5: 切换流量 ───
header "Step 5: 切换流量到 $STANDBY"

# 回滚标记：如果在切换过程中出错，尝试恢复到旧 active
ROLLBACK_NEEDED=false

if [ "$ACTIVE" = "none" ]; then
  # 冷启动：直接配 nginx
  step "冷启动：配置 nginx 指向 $STANDBY"
  if ! switch_nginx "$STANDBY"; then
    err "nginx 配置失败，冷启动中止"
    docker stop "$STANDBY" 2>/dev/null || true
    exit 1
  fi
else
  # 正常切换
  if ! switch_nginx "$STANDBY"; then
    err "nginx 切换失败，当前流量仍在 $ACTIVE 上"
    exit 1
  fi
fi

# ─── Step 6: 验证 ───
header "Step 6: 验证新服务"
sleep 2   # 等 nginx 完全生效
if ! verify_traffic "$STANDBY"; then
  err "流量验证失败！"

  # 如果是从 active 切过来的，尝试回滚
  if [ "$ACTIVE" != "none" ]; then
    warn "正在回滚到 $ACTIVE ..."
    if switch_nginx "$ACTIVE"; then
      log "已回滚到 $ACTIVE"
    else
      err "回滚失败！请手动检查 nginx 配置"
    fi
  fi
  exit 1
fi

# ─── Step 7: 停止旧容器 ───
header "Step 7: 停止旧容器"

if [ "$ACTIVE" = "none" ]; then
  log "冷启动完成，无需停止旧容器"
elif [ "$ACTIVE" = "new-api" ]; then
  step "停止旧容器 new-api ..."
  docker stop new-api 2>/dev/null || true
  docker rm new-api 2>/dev/null || true
  log "✓ 旧容器 new-api 已停止"
else
  step "停止旧容器 $ACTIVE ..."
  docker stop "$ACTIVE" 2>/dev/null || true
  docker rm "$ACTIVE" 2>/dev/null || true
  log "✓ $ACTIVE 已停止"
fi

# ─── 完成 ───

# 计算下次部署的 standby（当前未用的那个蓝绿容器）
if [ "$STANDBY" = "$GREEN_SVC" ]; then
  NEXT_STANDBY="$BLUE_SVC"
elif [ "$STANDBY" = "$BLUE_SVC" ]; then
  NEXT_STANDBY="$GREEN_SVC"
else
  NEXT_STANDBY="$BLUE_SVC"   # 冷启动 fallback
fi

header "部署完成"
echo ""
echo "  ┌──────────────────────────────────────────┐"
printf  "  │ %-25s : %-8s │\n" "当前在线" "$STANDBY"
printf  "  │ %-25s : %-8s │\n" "nginx 指向" "$(get_nginx_upstream)"
printf  "  │ %-25s : %-8s │\n" "备用容器" "$NEXT_STANDBY"
echo "  └──────────────────────────────────────────┘"
echo ""
log "零停机部署完成 ✓"
echo ""
echo "  下次部署时，将启动 $NEXT_STANDBY 作为 standby，"
echo "  就绪后再切回 $NEXT_STANDBY，实现交替轮换。"
echo ""

exit 0
