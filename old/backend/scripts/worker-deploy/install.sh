#!/bin/bash
# ============================================
# XingRin 远程节点安装脚本
# 用途：安装 Docker 环境 + 预拉取镜像
# 支持：Ubuntu / Debian / Kali
# 
# 架构说明：
# 1. 安装 Docker 环境
# 2. 预拉取 worker 镜像（避免任务执行时网络延迟）
# 3. agent 通过 start-agent.sh 启动（心跳上报）
# 4. 扫描任务由主服务器通过 SSH 执行 docker run
# 
# 镜像版本管理：
# - IMAGE_TAG 由主服务器传入，确保版本一致性
# - 预拉取后，任务执行时使用 --pull=missing 直接用本地镜像
# ============================================

set -e

MARKER_DIR="/opt/xingrin"
DOCKER_MARKER="${MARKER_DIR}/.docker_installed"

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

# 渐变色定义
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
BOLD='\033[1m'
DIM='\033[2m'

log_info() { echo -e "${CYAN}  ▸${NC} $1"; }
log_success() { echo -e "${GREEN}  ✔${NC} $1"; }
log_warn() { echo -e "${YELLOW}  ⚠${NC} $1"; }
log_error() { echo -e "${RED}  ✖${NC} $1"; }

# 炫酷 Banner
show_banner() {
    echo -e ""
    echo -e "${CYAN}${BOLD}    ██╗  ██╗██╗███╗   ██╗ ██████╗ ██████╗ ██╗███╗   ██╗${NC}"
    echo -e "${CYAN}    ╚██╗██╔╝██║████╗  ██║██╔════╝ ██╔══██╗██║████╗  ██║${NC}"
    echo -e "${BLUE}${BOLD}     ╚███╔╝ ██║██╔██╗ ██║██║  ███╗██████╔╝██║██╔██╗ ██║${NC}"
    echo -e "${BLUE}     ██╔██╗ ██║██║╚██╗██║██║   ██║██╔══██╗██║██║╚██╗██║${NC}"
    echo -e "${MAGENTA}${BOLD}    ██╔╝ ██╗██║██║ ╚████║╚██████╔╝██║  ██║██║██║ ╚████║${NC}"
    echo -e "${MAGENTA}    ╚═╝  ╚═╝╚═╝╚═╝  ╚═══╝ ╚═════╝ ╚═╝  ╚═╝╚═╝╚═╝  ╚═══╝${NC}"
    echo -e ""
    echo -e "${DIM}    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}      🚀 分布式安全扫描平台 │ Worker 节点部署${NC}"
    echo -e "${DIM}    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e ""
}

# 完成 Banner
show_complete() {
    echo -e ""
    echo -e "${GREEN}${BOLD}    ╔═══════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}${BOLD}    ║                                                   ║${NC}"
    echo -e "${GREEN}${BOLD}    ║   ██████╗  ██████╗ ███╗   ██╗███████╗██╗          ║${NC}"
    echo -e "${GREEN}${BOLD}    ║   ██╔══██╗██╔═══██╗████╗  ██║██╔════╝██║          ║${NC}"
    echo -e "${GREEN}${BOLD}    ║   ██║  ██║██║   ██║██╔██╗ ██║█████╗  ██║          ║${NC}"
    echo -e "${GREEN}${BOLD}    ║   ██║  ██║██║   ██║██║╚██╗██║██╔══╝  ╚═╝          ║${NC}"
    echo -e "${GREEN}${BOLD}    ║   ██████╔╝╚██████╔╝██║ ╚████║███████╗██╗          ║${NC}"
    echo -e "${GREEN}${BOLD}    ║   ╚═════╝  ╚═════╝ ╚═╝  ╚═══╝╚══════╝╚═╝          ║${NC}"
    echo -e "${GREEN}${BOLD}    ║                                                   ║${NC}"
    echo -e "${GREEN}${BOLD}    ║       ✨ XingRin Worker 节点部署完成！            ║${NC}"
    echo -e "${GREEN}${BOLD}    ║                                                   ║${NC}"
    echo -e "${GREEN}${BOLD}    ╚═══════════════════════════════════════════════════╝${NC}"
    echo -e ""
}

# 等待 apt 锁释放
wait_for_apt_lock() {
    local max_wait=60
    local waited=0
    while sudo fuser /var/lib/apt/lists/lock >/dev/null 2>&1 || \
          sudo fuser /var/lib/dpkg/lock >/dev/null 2>&1 || \
          sudo fuser /var/lib/dpkg/lock-frontend >/dev/null 2>&1; do
        if [ $waited -eq 0 ]; then
            log_info "等待 apt 锁释放..."
        fi
        sleep 2
        waited=$((waited + 2))
        if [ $waited -ge $max_wait ]; then
            log_warn "等待 apt 锁超时，继续尝试..."
            break
        fi
    done
}

# 检测操作系统
detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$ID
    else
        log_error "无法检测操作系统"
        exit 1
    fi
    
    if [[ "$OS" != "ubuntu" && "$OS" != "debian" && "$OS" != "kali" ]]; then
        log_error "仅支持 Ubuntu/Debian/Kali 系统"
        exit 1
    fi
}

# 安装 Docker
install_docker() {
    if command -v docker &> /dev/null; then
        log_info "Docker 已安装: $(docker --version)"
        return 0
    fi
    
    log_info "安装 Docker..."
    
    wait_for_apt_lock
    
    # 安装依赖
    sudo apt-get update -qq
    sudo apt-get install -y -qq ca-certificates curl gnupg lsb-release >/dev/null 2>&1
    
    # 添加 Docker GPG key
    sudo mkdir -p /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/${OS}/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg 2>/dev/null
    
    # 添加 Docker 源
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/${OS} $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
    
    # 安装 Docker
    sudo apt-get update -qq
    sudo apt-get install -y -qq docker-ce docker-ce-cli containerd.io docker-compose-plugin >/dev/null 2>&1
    
    # 启动 Docker
    sudo systemctl enable docker >/dev/null 2>&1 || true
    sudo systemctl start docker >/dev/null 2>&1 || true
    
    # 添加当前用户到 docker 组
    sudo usermod -aG docker $USER 2>/dev/null || true
    
    log_success "Docker 安装完成"
}

# 创建数据目录
create_dirs() {
    log_info "创建数据目录..."
    sudo mkdir -p "${MARKER_DIR}/results"
    sudo mkdir -p "${MARKER_DIR}/logs"
    sudo mkdir -p "${MARKER_DIR}/fingerprints"
    sudo mkdir -p "${MARKER_DIR}/wordlists"
    sudo chmod -R 777 "${MARKER_DIR}"
    log_success "数据目录已创建"
}

# 清理旧容器
cleanup_old_containers() {
    log_info "清理旧容器..."
    
    # 停止并删除旧的 agent 容器
    docker stop xingrin-agent 2>/dev/null || true
    docker rm xingrin-agent 2>/dev/null || true
    
    # 兼容旧名称
    docker stop xingrin-watchdog 2>/dev/null || true
    docker rm xingrin-watchdog 2>/dev/null || true
    
    log_success "旧容器已清理"
}

# 拉取 Worker 镜像（预先拉取，避免任务执行时网络延迟）
# 
# 镜像拉取策略：
# 1. 安装时预先拉取 worker 镜像到本地
# 2. 后续任务执行时使用 --pull=missing，直接用本地镜像
# 3. 版本由主服务器的 IMAGE_TAG 决定，确保版本一致性
pull_image() {
    log_info "拉取 Worker 镜像..."
    # 镜像版本由部署时传入（deploy_service.py 注入 IMAGE_TAG 环境变量）
    if [ -z "$IMAGE_TAG" ]; then
        log_error "IMAGE_TAG 未设置，请确保部署时传入版本号"
        exit 1
    fi
    local docker_user="${DOCKER_USER:-yyhuni}"
    # 拉取指定版本的 worker 镜像（用于执行扫描任务）
    sudo docker pull "${docker_user}/xingrin-worker:${IMAGE_TAG}"
    log_success "镜像拉取完成: ${docker_user}/xingrin-worker:${IMAGE_TAG}"
}

# 主流程
main() {
    show_banner
    
    detect_os
    install_docker
    cleanup_old_containers
    create_dirs
    pull_image
    
    touch "$DOCKER_MARKER"
    
    show_complete
}

main "$@"
