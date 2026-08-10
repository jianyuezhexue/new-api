#!/bin/bash
set -e

ROOT="$(cd "$(dirname "$0")" && pwd)"

# 1. 杀掉已有进程
echo "==> 清理已有进程..."
lsof -ti:3000 | xargs kill -9 2>/dev/null || true
lsof -ti:5173 | xargs kill -9 2>/dev/null || true
lsof -ti:3001 | xargs kill -9 2>/dev/null || true
sleep 1

# 2. 启动后端 (端口 3000)
echo "==> 启动后端..."
cd "$ROOT"
export SQL_DSN='root:buildingblocks@tcp(8.155.47.77:3306)/new-api'
export REDIS_CONN_STRING='redis://:buildingblocks@8.155.47.77:6379'
go run main.go &
BACKEND_PID=$!

# 3. 启动前端 (端口 5173)
echo "==> 启动前端..."
cd "$ROOT/web/default"
bun install --silent 2>/dev/null || true
bun run dev -- --port 5173 --host 0.0.0.0 &
FRONTEND_PID=$!

# 4. 启动文档站点 (端口 3001)
echo "==> 启动文档站点..."
cd "$ROOT/new-api-docs"
bun install --silent 2>/dev/null || true
bun run dev -- --port 3001 &
DOCS_PID=$!

# 等待服务就绪
sleep 4

echo ""
echo "=================================="
echo "  后端 API:    http://localhost:3000"
echo "  前端页面:    http://localhost:5173"
echo "  文档站点:    http://localhost:3001"
echo "=================================="
echo ""
echo "按 Ctrl+C 停止所有服务"

# 捕获退出信号，清理子进程
cleanup() {
  echo ""
  echo "==> 停止服务..."
  kill $BACKEND_PID $FRONTEND_PID $DOCS_PID 2>/dev/null
  wait $BACKEND_PID $FRONTEND_PID $DOCS_PID 2>/dev/null
  echo "==> 已停止"
  exit 0
}
trap cleanup INT TERM

# 等待任意子进程退出
wait -n $BACKEND_PID $FRONTEND_PID $DOCS_PID 2>/dev/null
cleanup
