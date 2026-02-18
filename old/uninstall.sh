#!/bin/bash
set -e

# ==============================================================================
# 颜色定义
# ==============================================================================
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
RESET='\033[0m'

# ==============================================================================
# 日志函数
# ==============================================================================
info() {
    echo -e "${BLUE}[INFO]${RESET} $1"
}

success() {
    echo -e "${GREEN}[OK]${RESET} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${RESET} $1"
}

error() {
    echo -e "${RED}[ERROR]${RESET} $1"
}

step() {
    echo -e "\n${BOLD}${CYAN}>>> $1${RESET}"
}

header() {
    echo -e "${BOLD}${BLUE}============================================================${RESET}"
    echo -e "${BOLD}${BLUE}   $1${RESET}"
    echo -e "${BOLD}${BLUE}============================================================${RESET}"
}

# ==============================================================================
# 权限检查
# ==============================================================================
if [ "$EUID" -ne 0 ]; then
    error "请使用 sudo 运行此脚本"
    echo -e "   正确用法: ${BOLD}sudo ./uninstall.sh${RESET}"
    exit 1
fi

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
DOCKER_DIR="$ROOT_DIR/docker"

header "XingRin 一键卸载脚本 (Ubuntu)"
info "项目路径: ${BOLD}$ROOT_DIR${RESET}"

if [ ! -d "$DOCKER_DIR" ]; then
    error "未找到 docker 目录，请确认项目结构。"
    exit 1
fi

# ==============================================================================
# 1. 停止并删除全部容器/网络
# ==============================================================================
step "[1/5] 是否停止并删除全部容器/网络？(Y/n)"
read -r ans_stop
ans_stop=${ans_stop:-Y}

if [[ $ans_stop =~ ^[Yy]$ ]]; then
    info "正在停止并删除容器/网络..."
    cd "$DOCKER_DIR"
    if command -v docker compose >/dev/null 2>&1; then
        COMPOSE_CMD="docker compose"
    else
        COMPOSE_CMD="docker-compose"
    fi
    
    # 先强制停止并删除可能占用网络的容器（xingrin-agent 等）
    docker rm -f xingrin-agent xingrin-watchdog 2>/dev/null || true
    
    # 清理所有可能的 XingRin 相关容器
    docker ps -a | grep -E "(xingrin|docker-)" | awk '{print $1}' | xargs -r docker rm -f 2>/dev/null || true
    
    # 停止两种模式的容器（不带 -v，volume 在第 5 步单独处理）
    [ -f "docker-compose.yml" ] && ${COMPOSE_CMD} -f docker-compose.yml down 2>/dev/null || true
    [ -f "docker-compose.dev.yml" ] && ${COMPOSE_CMD} -f docker-compose.dev.yml down 2>/dev/null || true
    
    # 手动删除网络（以防 compose 未能删除）
    docker network rm xingrin_network docker_default 2>/dev/null || true
    
    success "容器和网络已停止/删除（如存在）。"
else
    warn "已跳过停止/删除容器/网络。"
fi

# ==============================================================================
# 2. 删除 /opt/xingrin 数据目录
# ==============================================================================

OPT_XINGRIN_DIR="/opt/xingrin"

step "[2/5] 是否删除数据目录 ($OPT_XINGRIN_DIR)？(Y/n)"
echo -e "   ${YELLOW}包含：扫描结果、日志、指纹库、字典、Nuclei 模板等${RESET}"
read -r ans_data
ans_data=${ans_data:-Y}

if [[ $ans_data =~ ^[Yy]$ ]]; then
    info "正在删除数据目录..."
    rm -rf "$OPT_XINGRIN_DIR"
    success "已删除 $OPT_XINGRIN_DIR"
else
    warn "已保留数据目录。"
fi

# ==============================================================================
# 3. 删除 docker/.env 配置文件
# ==============================================================================
ENV_FILE="$DOCKER_DIR/.env"

step "[3/5] 是否删除配置文件 ($ENV_FILE)？(Y/n)"
echo -e "   ${YELLOW}注意：删除后下次安装将生成新的随机密码。${RESET}"
read -r ans_env
ans_env=${ans_env:-Y}

if [[ $ans_env =~ ^[Yy]$ ]]; then
    info "正在删除配置文件..."
    rm -f "$ENV_FILE"
    success "已删除 $ENV_FILE。"
else
    warn "已保留 $ENV_FILE。"
fi

# ==============================================================================
# 4. 删除本地 Postgres 容器及数据卷（如果使用本地 DB）
# ==============================================================================
step "[4/5] 若使用本地 PostgreSQL 容器：是否删除数据库容器和 volume？(Y/n)"
read -r ans_db
ans_db=${ans_db:-Y}

if [[ $ans_db =~ ^[Yy]$ ]]; then
    info "尝试删除与 XingRin 相关的 Postgres 容器和数据卷..."
    # 删除可能的容器名（不同 compose 版本命名不同）
    docker rm -f docker-postgres-1 xingrin-postgres postgres 2>/dev/null || true
    
    # 删除可能的 volume 名（取决于项目名和 compose 配置）
    # 先列出要删除的 volume
    for vol in postgres_data docker_postgres_data xingrin_postgres_data; do
        if docker volume inspect "$vol" >/dev/null 2>&1; then
            if docker volume rm "$vol" 2>/dev/null; then
                success "已删除 volume: $vol"
            else
                warn "无法删除 volume: $vol（可能正在被使用，请先停止所有容器）"
            fi
        fi
    done
    success "本地 Postgres 数据卷清理完成。"
else
    warn "已保留本地 Postgres 容器和 volume。"
fi

step "[5/5] 是否删除与 XingRin 相关的 Docker 镜像？(Y/n)"
read -r ans_images
ans_images=${ans_images:-Y}

if [[ $ans_images =~ ^[Yy]$ ]]; then
    info "正在删除 Docker 镜像..."
    
    # 从 .env 读取版本号，如果不存在则使用 latest
    if [ -f "$DOCKER_DIR/.env" ]; then
        DOCKER_USER=$(grep "^DOCKER_USER=" "$DOCKER_DIR/.env" | cut -d= -f2 || echo "yyhuni")
        IMAGE_TAG=$(grep "^IMAGE_TAG=" "$DOCKER_DIR/.env" | cut -d= -f2 || echo "latest")
    else
        DOCKER_USER="yyhuni"
        IMAGE_TAG="latest"
    fi
    
    # 删除指定版本的镜像
    docker rmi "${DOCKER_USER}/xingrin-server:${IMAGE_TAG}" 2>/dev/null || true
    docker rmi "${DOCKER_USER}/xingrin-frontend:${IMAGE_TAG}" 2>/dev/null || true
    docker rmi "${DOCKER_USER}/xingrin-nginx:${IMAGE_TAG}" 2>/dev/null || true
    docker rmi "${DOCKER_USER}/xingrin-agent:${IMAGE_TAG}" 2>/dev/null || true
    docker rmi "${DOCKER_USER}/xingrin-worker:${IMAGE_TAG}" 2>/dev/null || true
    
    # 同时删除 latest 标签（如果存在）
    if [ "$IMAGE_TAG" != "latest" ]; then
        docker rmi "${DOCKER_USER}/xingrin-server:latest" 2>/dev/null || true
        docker rmi "${DOCKER_USER}/xingrin-frontend:latest" 2>/dev/null || true
        docker rmi "${DOCKER_USER}/xingrin-nginx:latest" 2>/dev/null || true
        docker rmi "${DOCKER_USER}/xingrin-agent:latest" 2>/dev/null || true
        docker rmi "${DOCKER_USER}/xingrin-worker:latest" 2>/dev/null || true
    fi
    
    docker rmi redis:7-alpine 2>/dev/null || true
    
    # 删除本地构建的开发镜像
    docker rmi docker-server docker-frontend docker-nginx docker-agent docker-worker 2>/dev/null || true
    docker rmi "docker-worker:${IMAGE_TAG}-dev" 2>/dev/null || true
    
    success "Docker 镜像已删除（如存在）。"
else
    warn "已保留 Docker 镜像。"
fi

# 清理构建缓存（可选，会导致下次构建变慢）
echo ""
echo -n -e "${BOLD}${CYAN}[?] 是否清理 Docker 构建缓存？(y/N) ${RESET}"
echo -e "${YELLOW}（清理后下次构建会很慢，一般不需要）${RESET}"
read -r ans_cache
ans_cache=${ans_cache:-N}

if [[ $ans_cache =~ ^[Yy]$ ]]; then
    info "清理 Docker 构建缓存..."
    docker builder prune -af 2>/dev/null || true
    success "构建缓存已清理。"
else
    warn "已保留构建缓存（推荐）。"
fi

success "卸载流程已完成。"
