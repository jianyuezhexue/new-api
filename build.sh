#!/bin/bash
set -e

# ============================================================
# new-api 本地构建脚本（仅在开发机上运行，不在服务器上构建）
# 构建前端（两个主题）和后端，输出到 release/ 目录
# 产物上传到服务器后由 deploy.sh 部署
# ============================================================

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

# -------------------- 前端构建 --------------------
echo "=== 安装 workspace 依赖 ==="
cd web
bun install --frozen-lockfile
cd ..

echo "=== 构建前端（default 主题）==="
cd web/default
bun run build
cd ../..

echo "=== 构建前端（classic 主题）==="
cd web/classic
bun run build
cd ../..

# rsbuild 清空了 dist 目录，重建 .gitkeep 以消除 //go:embed 警告
touch web/default/dist/.gitkeep web/classic/dist/.gitkeep

# -------------------- 后端构建 --------------------
echo "=== 构建后端（linux/amd64）==="
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -trimpath \
    -o release/server .

# UPX 压缩（可选）
if command -v upx &> /dev/null; then
    echo "=== UPX 压缩中 ==="
    upx --best --lzma release/server
else
    echo "=== 提示: 安装 upx 可进一步压缩二进制 ==="
fi

# -------------------- 文档站点构建 --------------------
echo "=== 构建文档站点（Next.js standalone，仅 API 参考）==="
cd "$SCRIPT_DIR/new-api-docs"

# 只保留 API 章节，删掉其他内容源 → 编译产物体积从 900MB 降到 ~50MB
echo "  清理非 API 内容（guide installation apps skills support business legal）..."
for lang in zh en ja; do
  for dir in guide installation apps skills support business legal; do
    rm -rf "content/docs/$lang/$dir" 2>/dev/null || true
  done
done

bun install --frozen-lockfile
bun run build

if [ ! -f "$SCRIPT_DIR/new-api-docs/.next/standalone/server.js" ]; then
  echo "❌ 文档站点 standalone 输出缺失"
  exit 1
fi

echo "=== 打包文档产物（10k+ 散文件 → 单个 tar.xz）==="
cd "$SCRIPT_DIR/new-api-docs"

# 剔除运行时不需要的构建依赖，缩小编译产物体积
echo "  清理构建时依赖（typescript @img 等，运行时不需要）..."
rm -rf .next/standalone/node_modules/typescript \
       .next/standalone/node_modules/@img \
       .next/standalone/node_modules/sharp \
       .next/standalone/node_modules/@types

# 按 Dockerfile ADD 解压后的目标布局组织文件
rm -rf _bundle && mkdir -p _bundle
cp -a .next/standalone/. _bundle/          # server.js + node_modules → /app/

# 只复制被构建产物引用的静态资源（剔除未引用的 assets，缩小编译产物体积）
echo "  过滤未引用的 assets..."
mkdir -p _bundle/public
# 复制非 assets 的 public 文件（如 favicon.ico）
find public -maxdepth 1 -type f -exec cp -a {} _bundle/public/ \;
# 扫描 .next/ 构建产物，提取所有 /assets/... 引用，只复制被引用的文件/目录
mkdir -p _bundle/public/assets
REFERENCED_PATHS=$(grep -rohE '["'"'"']/assets/[^"'"'"'?#]+' .next/ 2>/dev/null | \
  sed 's/^["'"'"']//;s/\\*$//' | \
  sort -u)
COPIED=0
for path in $REFERENCED_PATHS; do
  rel="${path#/assets/}"
  src="public/assets/$rel"
  if [ -f "$src" ]; then
    mkdir -p "$(dirname "_bundle/public/assets/$rel")"
    cp -a "$src" "_bundle/public/assets/$rel" && COPIED=$((COPIED + 1))
  elif [ -d "$src" ]; then
    mkdir -p "$(dirname "_bundle/public/assets/$rel")"
    cp -a "$src" "_bundle/public/assets/$rel" && COPIED=$((COPIED + 1))
  fi
done
echo "  ✓ 已复制 $COPIED 个被引用的 assets"
# 兜底：如果扫描不到任何引用（极端情况），回退复制全部
if [ $COPIED -eq 0 ]; then
  echo "  ⚠ 未检测到 assets 引用，回退复制全部 public/assets/"
  cp -a public/assets/. _bundle/public/assets/
fi

mkdir -p _bundle/.next && cp -a .next/static _bundle/.next/  # 编译产物 → /app/.next/static/
cp -a openapi _bundle/openapi              # OpenAPI 规范 → /app/openapi/

# xz 压缩（Docker ADD 支持 tar.xz 自动解压，压缩率高能过 GitHub 100MB 限制）
XZ_OPT=-9e tar cJf docs-bundle.tar.xz -C _bundle .
rm -rf _bundle
echo "✅ docs-bundle.tar.xz: $(ls -lh "$SCRIPT_DIR/new-api-docs/docs-bundle.tar.xz" | awk '{print $5}')"

# -------------------- 构建完成 --------------------
echo ""
echo "=== 构建完成 ==="
echo "  release/server              ($(du -h "$SCRIPT_DIR/release/server" | cut -f1))"
echo "  docs-bundle.tar.xz          ($(ls -lh "$SCRIPT_DIR/new-api-docs/docs-bundle.tar.xz" | awk '{print $5}'))"
