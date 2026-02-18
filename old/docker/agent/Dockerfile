# ============================================
# XingRin Agent - 轻量心跳上报镜像
# 用途：心跳上报 + 负载监控 + 版本检查
# 基础镜像：Alpine Linux (~5MB)
# 最终大小：~10MB
# ============================================

FROM alpine:3.19

# 构建参数：版本号
ARG IMAGE_TAG=unknown

# 安装必要工具
RUN apk add --no-cache \
    bash \
    curl \
    procps

# 复制 agent 脚本
COPY backend/scripts/worker-deploy/agent.sh /app/agent.sh
RUN chmod +x /app/agent.sh

# 将版本号写入环境变量（运行时可用）
ENV IMAGE_TAG=${IMAGE_TAG}

# 工作目录
WORKDIR /app

# 默认命令
CMD ["bash", "/app/agent.sh"]
