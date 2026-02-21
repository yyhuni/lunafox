# syntax=docker/dockerfile:1.7

# Worker application image:
# - build scanner tools in this Dockerfile (no external tools image dependency)
# - compile worker binary and assemble runtime image

# ============================================
# Stage 1: Build scanner tools
# ============================================
FROM golang:1.26 AS tools-builder

ARG GO111MODULE=on
ARG GOPROXY=https://proxy.golang.org,https://goproxy.cn,direct
ARG GOFLAGS=-p=4
ARG GOMAXPROCS=4
ARG GONOSUMDB=github.com/yyhuni/xingfinger

ENV GO111MODULE=$GO111MODULE
ENV GOPROXY=$GOPROXY
ENV GOFLAGS=$GOFLAGS
ENV CGO_ENABLED=1
ENV GOMAXPROCS=$GOMAXPROCS
ENV GONOSUMDB=$GONOSUMDB

# Install build dependencies for CGO tools
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    apt-get update && apt-get install -y --no-install-recommends \
    libpcap-dev \
    git \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Build massdns (required by puredns)
RUN set -eux; \
    tries=0; \
    while [ "$tries" -lt 5 ]; do \
      tries=$((tries + 1)); \
      rm -rf /tmp/massdns; \
      if git clone --depth=1 https://github.com/blechschmidt/massdns.git /tmp/massdns; then \
        break; \
      fi; \
      echo "massdns clone failed (attempt $tries), retrying..." >&2; \
      sleep $((tries * 2)); \
    done; \
    [ -d /tmp/massdns/.git ] || { echo "failed to clone massdns after 5 attempts" >&2; exit 1; }; \
    cd /tmp/massdns && \
    make && \
    cp bin/massdns /usr/local/bin/massdns

# Build Go tools with retries
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    set -eux; \
    install_tool() { \
      pkg="$1"; \
      tries=0; \
      while [ "$tries" -lt 3 ]; do \
        tries=$((tries + 1)); \
        if go install -v "$pkg"; then \
          return 0; \
        fi; \
        echo "install failed for $pkg (attempt $tries), retrying..." >&2; \
        sleep $((tries * 2)); \
      done; \
      echo "install failed for $pkg after 3 attempts" >&2; \
      exit 1; \
    }; \
    install_tool_no_sumdb() { \
      pkg="$1"; \
      proxy_candidates="https://proxy.golang.org,direct https://goproxy.cn,direct direct"; \
      for proxy in $proxy_candidates; do \
        tries=0; \
        while [ "$tries" -lt 4 ]; do \
          tries=$((tries + 1)); \
          tmp_modcache="$(mktemp -d)"; \
          tmp_gocache="$(mktemp -d)"; \
          errlog="$(mktemp)"; \
          if GOSUMDB=off GOPROXY="$proxy" GOFLAGS=-p=2 GOMAXPROCS=2 GOMODCACHE="$tmp_modcache" GOCACHE="$tmp_gocache" GIT_TERMINAL_PROMPT=0 GIT_HTTP_VERSION=HTTP/1.1 go install -v "$pkg" 2>"$errlog"; then \
            rm -rf "$tmp_modcache" "$tmp_gocache"; \
            rm -f "$errlog"; \
            return 0; \
          fi; \
          echo "install (GOSUMDB=off GOPROXY=$proxy) failed for $pkg (attempt $tries), retrying..." >&2; \
          tail -n 40 "$errlog" >&2 || true; \
          rm -rf "$tmp_modcache" "$tmp_gocache"; \
          rm -f "$errlog"; \
          sleep $((tries * 2)); \
        done; \
      done; \
      echo "install (GOSUMDB=off) failed for $pkg after retries across proxy candidates" >&2; \
      exit 1; \
    }; \
    install_tool github.com/projectdiscovery/httpx/cmd/httpx@v1.8.1; \
    install_tool github.com/projectdiscovery/naabu/v2/cmd/naabu@v2.4.0; \
    install_tool github.com/projectdiscovery/subfinder/v2/cmd/subfinder@v2.12.0; \
    install_tool github.com/projectdiscovery/nuclei/v3/cmd/nuclei@v3.7.0; \
    install_tool github.com/projectdiscovery/katana/cmd/katana@v1.4.0; \
    install_tool github.com/tomnomnom/assetfinder@v0.1.1; \
    install_tool github.com/ffuf/ffuf/v2@v2.1.0; \
    install_tool github.com/d3mondev/puredns/v2@v2.1.1; \
    install_tool_no_sumdb github.com/yyhuni/xingfinger@v0.0.0-20251228093343-bd42c7be5a70; \
    install_tool github.com/hahwul/dalfox/v2@v2.12.0

# ============================================
# Stage 2: Build Go Worker
# ============================================
FROM golang:1.26 AS worker-builder

ARG GO111MODULE=on
ARG GOPROXY=https://proxy.golang.org,https://goproxy.cn,direct

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

# Copy prebuilt scanning tools from tools builder
COPY --from=tools-builder /go/bin/ /opt/lunafox-tools/bin
COPY --from=tools-builder /usr/local/bin/massdns /opt/lunafox-tools/bin/massdns

# Copy Go Worker
COPY --from=worker-builder /worker /usr/local/bin/worker

# Run as root by default:
# - Avoid host mount permission issues
# - Some scanners (e.g., masscan) need elevated privileges
# For least privilege, switch to a non-root user and add required capabilities.
CMD ["worker"]
