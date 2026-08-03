#!/bin/bash
set -e

# ============================================================
# new-api 生产环境构建脚本
# 构建前端（两个主题）和后端，输出到 release/ 目录
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
echo "=== 构建文档站点（Next.js standalone）==="
cd "$SCRIPT_DIR/new-api-docs"
bun install --frozen-lockfile
bun run build

# 验证 standalone 输出
if [ ! -f "$SCRIPT_DIR/new-api-docs/.next/standalone/server.js" ]; then
  echo "❌ 文档站点 standalone 输出缺失"
  exit 1
fi
echo "✅ 文档站点 standalone: $(du -sh "$SCRIPT_DIR/new-api-docs/.next/standalone" | cut -f1)"

# -------------------- 构建完成 --------------------
echo ""
echo "=== 构建完成 ==="
echo "  release/server                ($(du -h "$SCRIPT_DIR/release/server" | cut -f1))"
echo "  文档站点 .next/standalone     ($(du -sh "$SCRIPT_DIR/new-api-docs/.next/standalone" | cut -f1))"
echo "  前端已通过 //go:embed 嵌入到二进制中"
