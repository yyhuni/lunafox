# Worker Dockerfile - Go version
# Multi-stage build: compile scanning tools + build Go Worker

# ============================================
# Stage 1: Build Go scanning tools
# ============================================
FROM golang:1.26 AS tools-builder

ARG GO111MODULE=on
ARG GOPROXY=https://goproxy.cn,direct

ENV GO111MODULE=$GO111MODULE
ENV GOPROXY=$GOPROXY
ENV CGO_ENABLED=1

# Install build dependencies (build stage, unpinned to get latest security updates)
# hadolint ignore=DL3008
RUN apt-get update && apt-get install -y --no-install-recommends \
    libpcap-dev \
    git \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Build massdns (required by puredns)
RUN git clone https://github.com/blechschmidt/massdns.git /tmp/massdns && \
    cd /tmp/massdns && \
    make && \
    cp bin/massdns /usr/local/bin/massdns

# Install ProjectDiscovery and other Go tools
RUN go install -v github.com/projectdiscovery/httpx/cmd/httpx@latest && \
    go install -v github.com/projectdiscovery/naabu/v2/cmd/naabu@latest && \
    go install -v github.com/projectdiscovery/subfinder/v2/cmd/subfinder@latest && \
    go install -v github.com/projectdiscovery/nuclei/v3/cmd/nuclei@latest && \
    go install -v github.com/projectdiscovery/katana/cmd/katana@latest && \
    go install -v github.com/tomnomnom/assetfinder@latest && \
    go install -v github.com/ffuf/ffuf/v2@latest && \
    go install -v github.com/d3mondev/puredns/v2@latest && \
    go install -v github.com/yyhuni/xingfinger@latest && \
    go install -v github.com/hahwul/dalfox/v2@latest

# ============================================
# Stage 2: Build Go Worker
# ============================================
FROM golang:1.26 AS worker-builder

ARG GO111MODULE=on
ARG GOPROXY=https://goproxy.cn,direct

ENV GO111MODULE=$GO111MODULE
ENV GOPROXY=$GOPROXY

WORKDIR /src

# Copy deps first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 go build -o /worker ./cmd/worker

# ============================================
# Stage 3: Runtime image
# ============================================
FROM ubuntu:noble-20260113

ENV DEBIAN_FRONTEND=noninteractive

WORKDIR /app

# Install runtime deps + Python environment
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    libpcap-dev \
    nmap \
    masscan \
    curl \
    git \
    python3 \
    python3-pip \
    python3-venv \
    pipx \
    && rm -rf /var/lib/apt/lists/*

# Create python symlink
RUN ln -s /usr/bin/python3 /usr/bin/python

# Directory conventions:
# - /opt/lunafox        data only (overridden by host mount)
# - /opt/lunafox-tools  tools only (from image, not overridden)
RUN mkdir -p \
    /opt/lunafox/wordlists \
    /opt/lunafox/workspace \
    /opt/lunafox-tools/bin \
    /opt/lunafox-tools/share \
    /opt/lunafox-tools/pipx

# Copy resolvers.txt (fixed DNS server list)
COPY resources/resolvers.txt /opt/lunafox/wordlists/resolvers.txt

# pipx: install Python tools into project-specific directory
ENV PIPX_HOME=/opt/lunafox-tools/pipx
ENV PIPX_BIN_DIR=/opt/lunafox-tools/bin
ENV PATH=/opt/lunafox-tools/bin:$PATH

# Install Python scanning tools
RUN pipx install uro && \
    pipx install waymore && \
    pipx install dnsgen

# Copy Go scanning tools (into /opt/lunafox-tools/bin)
COPY --from=tools-builder /go/bin/* /opt/lunafox-tools/bin/
COPY --from=tools-builder /usr/local/bin/massdns /opt/lunafox-tools/bin/massdns

# Copy Go Worker
COPY --from=worker-builder /worker /usr/local/bin/worker

# Run as root by default:
# - Avoid host mount permission issues
# - Some scanners (e.g., masscan) need elevated privileges
# For least privilege, switch to a non-root user and add required capabilities.
CMD ["worker"]
