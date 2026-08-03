# ============================================
# new-api-docs 多阶段 Dockerfile
# 构建阶段打包 .next/standalone，运行阶段仅包含最小产物
# Docker BuildKit 自动缓存：源码不变 → 秒级构建
# ============================================

# ── 构建阶段 ──
FROM oven/bun:1 AS builder
WORKDIR /app

# 依赖层（变更频率最低，优先缓存）
COPY package.json bun.lock ./
RUN bun install --frozen-lockfile

# 源码层
COPY source.config.ts tsconfig.json next.config.mjs postcss.config.mjs ./
COPY src ./src
COPY content ./content
COPY public ./public
COPY openapi ./openapi
COPY scripts ./scripts

# 构建（output: 'standalone' → .next/standalone/）
RUN bun run build

# ── 运行阶段 ──
FROM node:22-alpine
WORKDIR /app

ENV NODE_ENV=production
ENV HOSTNAME=0.0.0.0
ENV PORT=3000

# 从构建阶段复制 standalone 输出（包含 traced node_modules + server.js）
COPY --from=builder /app/.next/standalone ./

# standalone 模式需手动复制静态资源
COPY --from=builder /app/public ./public
COPY --from=builder /app/.next/static ./.next/static

# OpenAPI 规范文件（fumadocs-openapi 运行时读取）
COPY --from=builder /app/openapi ./openapi

EXPOSE 3000

CMD ["node", "server.js"]
