# LEGACY NOTICE
#
# This Dockerfile builds the *legacy* Python executor image from the old stack.
# For the Go Worker image (new architecture), use: worker/Dockerfile
#
# 第一阶段：使用 Go 官方镜像编译工具
FROM golang:1.24 AS go-builder

ENV GOPROXY=https://goproxy.cn,direct
# Naabu 需要 CGO 和 libpcap
ENV CGO_ENABLED=1

# 安装编译依赖（libpcap-dev 用于 naabu，git/build-essential 用于编译 massdns）
RUN apt-get update && apt-get install -y \
    libpcap-dev \
    git \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# 安装 massdns（puredns 依赖）
RUN git clone https://github.com/blechschmidt/massdns.git /tmp/massdns && \
    cd /tmp/massdns && \
    make && \
    cp bin/massdns /usr/local/bin/massdns

# 安装 ProjectDiscovery 等 Go 工具（需要 CGO 的工具如 naabu）
RUN go install -v github.com/projectdiscovery/httpx/cmd/httpx@latest && \
    go install -v github.com/projectdiscovery/naabu/v2/cmd/naabu@latest && \
    go install -v github.com/projectdiscovery/subfinder/v2/cmd/subfinder@latest && \
    go install -v github.com/projectdiscovery/nuclei/v3/cmd/nuclei@latest && \
    go install -v github.com/projectdiscovery/katana/cmd/katana@latest && \
    go install -v github.com/tomnomnom/assetfinder@latest && \
    go install -v github.com/ffuf/ffuf/v2@latest && \
    go install -v github.com/d3mondev/puredns/v2@latest && \
    go install -v github.com/yyhuni/xingfinger@latest

# 安装漏洞扫描器
RUN go install github.com/hahwul/dalfox/v2@latest

# 第二阶段：运行时镜像
FROM ubuntu:24.04

# 避免交互式提示
ENV DEBIAN_FRONTEND=noninteractive

# 设置工作目录
WORKDIR /app

# 1. 安装基础工具和 Python
# 注意：ARM64 使用 ports.ubuntu.com，可能存在镜像同步延迟，需要重试机制
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    python3 \
    python3-pip \
    python3-venv \
    pipx \
    git \
    curl \
    wget \
    unzip \
    jq \
    tmux \
    nmap \
    masscan \
    libpcap-dev \
    ca-certificates \
    fonts-liberation \
    libnss3 \
    libxss1 \
    libasound2t64 \
    || (rm -rf /var/lib/apt/lists/* && apt-get update && apt-get install -y --no-install-recommends \
    python3 python3-pip python3-venv pipx git curl wget unzip jq tmux nmap masscan libpcap-dev \
    ca-certificates fonts-liberation libnss3 libxss1 libasound2t64) \
    && rm -rf /var/lib/apt/lists/*

# 安装 Chromium（通过 Playwright 安装，支持 ARM64 和 AMD64）
# Ubuntu 24.04 的 chromium-browser 是 snap 过渡包，Docker 中不可用
RUN pip install playwright --break-system-packages && \
    playwright install chromium && \
    apt-get update && \
    playwright install-deps chromium && \
    rm -rf /var/lib/apt/lists/*

# 设置 Chrome 路径供 httpx 等工具使用（Playwright 安装位置）
ENV CHROME_PATH=/root/.cache/ms-playwright/chromium-*/chrome-linux/chrome
# 创建软链接确保 httpx 的 -system-chrome 能找到浏览器
RUN CHROME_BIN=$(find /root/.cache/ms-playwright -name chrome -type f 2>/dev/null | head -1) && \
    ln -sf "$CHROME_BIN" /usr/bin/chromium-browser && \
    ln -sf "$CHROME_BIN" /usr/bin/chromium && \
    ln -sf "$CHROME_BIN" /usr/bin/chrome && \
    ln -sf "$CHROME_BIN" /usr/bin/google-chrome-stable

# 建立 python 软链接
RUN ln -s /usr/bin/python3 /usr/bin/python

# 2. 使用 pipx 安装 Python 扫描工具
ENV PATH="/root/.local/bin:$PATH"
RUN pipx install uro && \
    pipx install waymore && \
    pipx install dnsgen

# 3. 安装 Sublist3r（Python 脚本工具，放在 /usr/local/share 标准目录）
RUN git clone https://github.com/aboul3la/Sublist3r.git /usr/local/share/Sublist3r && \
    pip3 install --no-cache-dir -r /usr/local/share/Sublist3r/requirements.txt --break-system-packages

# 4. 从 go-builder 阶段复制 Go 环境和编译好的工具
# 创建项目专用工具目录（符合 FHS 标准，/opt 用于独立软件包）
# 避免与系统工具或 Python 包冲突，避免被 /opt/xingrin 挂载覆盖
RUN mkdir -p /opt/xingrin-tools/bin

ENV GOPATH=/root/go
ENV GOPROXY=https://goproxy.cn,direct

COPY --from=go-builder /usr/local/go /usr/local/go

# 从 go-builder 复制扫描工具到专用目录（避免与系统工具或 Python 包冲突）
COPY --from=go-builder /go/bin/* /opt/xingrin-tools/bin/
COPY --from=go-builder /usr/local/bin/massdns /opt/xingrin-tools/bin/massdns

# 将专用工具目录添加到 PATH（优先级高于 /usr/local/bin，避免冲突）
ENV PATH=/opt/xingrin-tools/bin:/usr/local/go/bin:/usr/local/bin:$PATH:$GOPATH/bin

# 5. 安装 uv（ Python 包管理器）并安装 Python 依赖
COPY backend/requirements.txt .
RUN pip install uv --break-system-packages && \
    uv pip install --system -r requirements.txt --break-system-packages && \
    rm -f /usr/local/lib/python3.*/dist-packages/argparse.py && \
    rm -rf /usr/local/lib/python3.*/dist-packages/__pycache__/argparse* && \
    rm -rf /root/.cache/uv && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# 6. 设置 Prefect 配置目录（避免 home 目录不存在的警告）
ENV PREFECT_HOME=/app/.prefect
RUN mkdir -p /app/.prefect

# 7. 复制后端代码
COPY backend /app/backend
ENV PYTHONPATH=/app/backend

# 工作目录设置为 backend，方便运行 python -m 命令
WORKDIR /app/backend

# 默认命令（实际由 TaskDistributor 指定具体脚本）
CMD ["python", "--version"]
