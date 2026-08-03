#!/bin/bash
# ============================================
# build-docker.sh
# 本地构建 + 打包 Docker 镜像，交付给生产环境直接运行
#
# 用法:
#   bash scripts/build-docker.sh          # 构建镜像
#   bash scripts/build-docker.sh --export # 构建 + 导出 tar 包
# ============================================
set -e

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
IMAGE="new-api-docs:latest"
EXPORT_FILE="$ROOT/new-api-docs.tar"

echo "============================================"
echo "  new-api-docs Docker 构建"
echo "============================================"
echo ""

# ── 1. 本地构建（CPU 密集，在生产环境之外完成） ──
echo "==> [1/3] 安装依赖 & 本地构建..."
cd "$ROOT"
bun install --silent 2>/dev/null || bun install

echo "==> 执行 next build (standalone 模式)..."
bun run build

# 验证 standalone 输出
if [ ! -f "$ROOT/.next/standalone/server.js" ]; then
  echo "❌ 错误: standalone 输出缺失，检查 next.config.mjs 中 output 配置"
  exit 1
fi
echo "✅ 构建完成 ($(du -sh "$ROOT/.next/standalone" | cut -f1))"

# ── 2. Docker 打包（轻量，只复制预构建产物） ──
echo ""
echo "==> [2/3] 构建 Docker 镜像..."
cd "$ROOT"
docker build -t "$IMAGE" .

IMAGE_SIZE=$(docker images "$IMAGE" --format "{{.Size}}" | head -1)
echo "✅ 镜像构建完成: $IMAGE ($IMAGE_SIZE)"

# ── 3. 可选导出 ──
if [ "${1:-}" = "--export" ]; then
  echo ""
  echo "==> [3/3] 导出 tar 包..."
  docker save "$IMAGE" -o "$EXPORT_FILE"
  TAR_SIZE=$(ls -lh "$EXPORT_FILE" | awk '{print $5}')
  echo "✅ 已导出: $EXPORT_FILE ($TAR_SIZE)"
  echo ""
  echo "──────────────────────────────────────"
  echo "  交付物: $EXPORT_FILE"
  echo ""
  echo "  生产环境部署:"
  echo "    docker load -i new-api-docs.tar"
  echo "    docker run -d -p 3000:3000 new-api-docs:latest"
  echo "──────────────────────────────────────"
else
  echo ""
  echo "──────────────────────────────────────"
  echo "  镜像: $IMAGE"
  echo ""
  echo "  推送镜像到仓库 (可选):"
  echo "    docker tag $IMAGE registry.example.com/new-api-docs:latest"
  echo "    docker push registry.example.com/new-api-docs:latest"
  echo ""
  echo "  本地运行验证:"
  echo "    docker run -d -p 3000:3000 $IMAGE"
  echo "──────────────────────────────────────"
fi
