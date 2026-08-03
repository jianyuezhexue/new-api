# ============================================
# Production Dockerfile for new-api-docs
# Expects pre-built .next/ from local machine
# Build: bash scripts/build-docker.sh
# ============================================

FROM node:22-alpine

WORKDIR /app

ENV NODE_ENV=production
ENV HOSTNAME=0.0.0.0
ENV PORT=3000

# Copy standalone server output (includes traced node_modules + server.js)
COPY .next/standalone ./

# Copy static assets (standalone mode doesn't auto-include these)
COPY public ./public
COPY .next/static ./.next/static

# Copy OpenAPI specs (read at runtime by fumadocs-openapi)
COPY openapi ./openapi

EXPOSE 3000

CMD ["node", "server.js"]
