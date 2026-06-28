#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

# 1. 构建
echo "=== 第一步: 构建 ==="
bash build.sh

# 2. Git 提交
echo ""
echo "=== 第二步: Git 提交 ==="
git add release/ Dockerfile.runtime docker-compose.yml build.sh push.sh
git commit -m "build: 更新 release 构建产物及部署文件" || echo "无变更需提交"

# 3. Git 推送
echo ""
echo "=== 第三步: Git 推送 ==="
git push

echo ""
echo "=== 全部完成 ==="
