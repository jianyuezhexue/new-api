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
if bun run build; then
    echo "classic 主题构建成功"
else
    echo "警告: classic 主题构建失败，使用占位页面"
    mkdir -p dist
    echo '<!doctype html><html><head><title>New API</title></head><body><h1>New API</h1><p>classic 主题构建不可用 — 请使用 default 主题。</p></body></html>' > dist/index.html
fi
cd ../..

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

echo ""
echo "=== 构建完成 ==="
ls -lh release/server
echo ""
echo "release/server  ($(du -h release/server | cut -f1))"
echo "前端已通过 //go:embed 嵌入到二进制中"
