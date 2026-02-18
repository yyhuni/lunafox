# syntax=docker/dockerfile:1.7

# Worker application image:
# - heavy scanning tools are prebuilt in worker/Dockerfile.tools
# - this Dockerfile focuses on compiling worker binary and assembling runtime

ARG WORKER_TOOLS_IMAGE=docker.io/yyhuni/lunafox-worker-tools:tools-latest
FROM ${WORKER_TOOLS_IMAGE} AS tools-source

# ============================================
# Stage 1: Build Go Worker
# ============================================
FROM golang:1.26 AS worker-builder

ARG GO111MODULE=on
ARG GOPROXY=https://goproxy.cn,direct

ENV GO111MODULE=$GO111MODULE
ENV GOPROXY=$GOPROXY

WORKDIR /src

# Copy deps first for caching
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    go mod download

# Copy source and build
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    CGO_ENABLED=0 go build -o /worker ./cmd/worker

# ============================================
# Stage 2: Runtime image
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

# Copy prebuilt scanning tools from tools image
COPY --from=tools-source /opt/lunafox-tools/bin /opt/lunafox-tools/bin

# Copy Go Worker
COPY --from=worker-builder /worker /usr/local/bin/worker

# Run as root by default:
# - Avoid host mount permission issues
# - Some scanners (e.g., masscan) need elevated privileges
# For least privilege, switch to a non-root user and add required capabilities.
CMD ["worker"]
