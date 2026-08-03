# ============================================
# new-api-docs 运行时镜像
# build.sh → docs-bundle.tar.xz → ADD 自动解压
# 和 Dockerfile.runtime 同模式：只 ADD，不 BUILD
# ============================================

FROM node:20-alpine
WORKDIR /app

ENV NODE_ENV=production
ENV HOSTNAME=0.0.0.0
ENV PORT=3000

# ADD 自动解压 tar.xz（Docker 原生支持 xz 压缩）
ADD docs-bundle.tar.xz /app/

EXPOSE 3000

CMD ["node", "server.js"]
