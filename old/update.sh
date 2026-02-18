#!/bin/bash
# ============================================
# XingRin 系统更新脚本
# 用途：更新代码 + 同步版本 + 重建镜像 + 重启服务
# ============================================
#
# 更新流程：
# 1. 停止服务
# 2. git pull 拉取最新代码
# 3. 合并 .env 新配置项 + 同步 VERSION
# 4. 构建/拉取镜像（开发模式构建，生产模式拉取）
# 5. 启动服务（server 启动时自动执行数据库迁移）
#
# 用法:
#   sudo ./update.sh                 生产模式更新（拉取 Docker Hub 镜像）
#   sudo ./update.sh --dev           开发模式更新（本地构建镜像）
#   sudo ./update.sh --no-frontend   更新后只启动后端
#   sudo ./update.sh --dev --no-frontend     开发环境更新后只启动后端

cd "$(dirname "$0")"

# 权限检查
if [ "$EUID" -ne 0 ]; then
    printf "\033[0;31m✗ 请使用 sudo 运行此脚本\033[0m\n"
    printf "  正确用法: \033[1msudo ./update.sh\033[0m\n"
    exit 1
fi

# 跨平台 sed -i（兼容 macOS 和 Linux）
sed_inplace() {
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' "$@"
    else
        sed -i "$@"
    fi
}

# 解析参数判断模式
DEV_MODE=false
for arg in "$@"; do
    case $arg in
        --dev) DEV_MODE=true ;;
    esac
done

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
DIM='\033[2m'
BOLD='\033[1m'
NC='\033[0m'

# 日志函数
log_step() { printf "${CYAN}▶${NC} %s\n" "$1"; }
log_ok() { printf "  ${GREEN}✓${NC} %s\n" "$1"; }
log_info() { printf "  ${DIM}→${NC} %s\n" "$1"; }
log_warn() { printf "  ${YELLOW}!${NC} %s\n" "$1"; }
log_error() { printf "${RED}✗${NC} %s\n" "$1"; }

# 合并 .env 新配置项（保留用户已有值）
merge_env_config() {
    local example_file="docker/.env.example"
    local env_file="docker/.env"
    
    if [ ! -f "$example_file" ] || [ ! -f "$env_file" ]; then
        return
    fi
    
    local new_keys=0
    
    while IFS= read -r line || [ -n "$line" ]; do
        [[ -z "$line" || "$line" =~ ^# ]] && continue
        local key="${line%%=*}"
        [[ -z "$key" || "$key" == "$line" ]] && continue
        
        if ! grep -q "^${key}=" "$env_file"; then
            printf '%s\n' "$line" >> "$env_file"
            log_info "新增配置: $key"
            ((new_keys++))
        fi
    done < "$example_file"
    
    if [ $new_keys -gt 0 ]; then
        log_ok "已添加 $new_keys 个新配置项"
    else
        log_ok "配置已是最新"
    fi
}

# 显示标题
printf "\n"
printf "${BOLD}${BLUE}┌────────────────────────────────────────┐${NC}\n"
if [ "$DEV_MODE" = true ]; then
    printf "${BOLD}${BLUE}│${NC}       ${BOLD}XingRin 系统更新${NC}                 ${BOLD}${BLUE}│${NC}\n"
    printf "${BOLD}${BLUE}│${NC}       ${DIM}开发模式 · 本地构建${NC}               ${BOLD}${BLUE}│${NC}\n"
else
    printf "${BOLD}${BLUE}│${NC}       ${BOLD}XingRin 系统更新${NC}                 ${BOLD}${BLUE}│${NC}\n"
    printf "${BOLD}${BLUE}│${NC}       ${DIM}生产模式 · Docker Hub${NC}             ${BOLD}${BLUE}│${NC}\n"
fi
printf "${BOLD}${BLUE}└────────────────────────────────────────┘${NC}\n"
printf "\n"

# 警告提示
printf "${YELLOW}┌─ 注意事项 ─────────────────────────────┐${NC}\n"
printf "${YELLOW}│${NC} • 此功能为测试性功能，可能导致升级失败   ${YELLOW}│${NC}\n"
printf "${YELLOW}│${NC} • 升级会覆盖所有默认引擎配置             ${YELLOW}│${NC}\n"
printf "${YELLOW}│${NC} • 自定义配置请先备份或创建新引擎         ${YELLOW}│${NC}\n"
printf "${YELLOW}│${NC} • 推荐：卸载后重新安装以获得最佳体验     ${YELLOW}│${NC}\n"
printf "${YELLOW}└────────────────────────────────────────┘${NC}\n"
printf "\n"

printf "${YELLOW}是否继续更新？${NC} [y/N] "
read -r ans_continue
ans_continue=${ans_continue:-N}

if [[ ! $ans_continue =~ ^[Yy]$ ]]; then
    printf "\n${DIM}已取消更新${NC}\n"
    exit 0
fi
printf "\n"

# Step 1: 停止服务
log_step "停止服务..."
./stop.sh 2>&1 | sed 's/^/  /'
log_ok "服务已停止"

# Step 2: 拉取代码
printf "\n"
log_step "拉取最新代码..."
if git pull --rebase 2>&1 | sed 's/^/  /'; then
    log_ok "代码已更新"
else
    log_error "git pull 失败，请手动解决冲突后重试"
    exit 1
fi

# Step 3: 检查配置更新 + 版本同步
printf "\n"
log_step "同步配置..."
merge_env_config

# 版本同步：从 VERSION 文件更新 IMAGE_TAG
if [ -f "VERSION" ]; then
    NEW_VERSION=$(cat VERSION | tr -d '[:space:]')
    if [ -n "$NEW_VERSION" ]; then
        if grep -q "^IMAGE_TAG=" "docker/.env"; then
            sed_inplace "s/^IMAGE_TAG=.*/IMAGE_TAG=$NEW_VERSION/" "docker/.env"
        else
            printf '%s\n' "IMAGE_TAG=$NEW_VERSION" >> "docker/.env"
        fi
        log_ok "版本同步: $NEW_VERSION"
    fi
fi

# Step 4: 构建/拉取镜像
printf "\n"
log_step "更新镜像..."

if [ "$DEV_MODE" = true ]; then
    # 开发模式：本地构建所有镜像（包括 Worker）
    log_info "构建 Worker 镜像..."
    
    # 读取 IMAGE_TAG
    IMAGE_TAG=$(grep "^IMAGE_TAG=" "docker/.env" | cut -d'=' -f2)
    if [ -z "$IMAGE_TAG" ]; then
        IMAGE_TAG="dev"
    fi
    
    # 构建 Worker 镜像（Worker 是临时容器，不在 compose 中，需要单独构建）
    docker build -t docker-worker -f docker/worker/Dockerfile . 2>&1 | sed 's/^/  /'
    docker tag docker-worker docker-worker:${IMAGE_TAG} 2>&1 | sed 's/^/  /'
    log_ok "Worker 镜像: docker-worker:${IMAGE_TAG}"
    
    log_info "其他服务镜像将在启动时构建"
else
    log_info "镜像将在启动时从 Docker Hub 拉取"
fi

# Step 5: 启动服务
printf "\n"
log_step "启动服务..."
./start.sh "$@"

# 完成提示
printf "\n"
printf "${GREEN}┌────────────────────────────────────────┐${NC}\n"
printf "${GREEN}│${NC}  ${BOLD}${GREEN}✓${NC} ${BOLD}更新完成${NC}                            ${GREEN}│${NC}\n"
printf "${GREEN}└────────────────────────────────────────┘${NC}\n"
printf "\n"
