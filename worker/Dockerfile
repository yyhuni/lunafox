# syntax=docker/dockerfile:1.7

# Worker application image:
# - build scanner tools in this Dockerfile (no external tools image dependency)
# - compile worker binary and assemble runtime image

# ============================================
# Stage 1: Build scanner tools
# ============================================
FROM golang:1.26 AS tools-builder # TODO: Version 1.26 May have issues. Version 1.24 is recommended for testing

ARG GO111MODULE=on
ARG GOPROXY=https://proxy.golang.org,direct
ARG GOFLAGS=-p=1
ARG GOMAXPROCS=2
ARG MASSDNS_VERSION=v1.1.0

ENV GO111MODULE=$GO111MODULE
ENV GOPROXY=$GOPROXY
ENV GOFLAGS=$GOFLAGS
ENV CGO_ENABLED=1
ENV GOMAXPROCS=$GOMAXPROCS

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
  if git clone --branch "${MASSDNS_VERSION}" --depth=1 https://github.com/blechschmidt/massdns.git /tmp/massdns; then \
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
  now_ms() { \
  date +%s%3N 2>/dev/null || echo "$(( $(date +%s) * 1000 ))"; \
  }; \
  probe_module="golang.org/x/text@v0.14.0"; \
  rank_file="$(mktemp)"; \
  probe_proxy() { \
  proxy="$1"; \
  sumdb="$2"; \
  label="$3"; \
  start_ms="$(now_ms)"; \
  if command -v timeout >/dev/null 2>&1; then \
  if timeout 20s env GO111MODULE=on GOPROXY="$proxy" GOSUMDB="$sumdb" GIT_TERMINAL_PROMPT=0 GOFLAGS=-p=1 GOMAXPROCS=1 go list -m -mod=mod "$probe_module" >/dev/null 2>&1; then \
  status=0; \
  else \
  status=1; \
  fi; \
  else \
  if env GO111MODULE=on GOPROXY="$proxy" GOSUMDB="$sumdb" GIT_TERMINAL_PROMPT=0 GOFLAGS=-p=1 GOMAXPROCS=1 go list -m -mod=mod "$probe_module" >/dev/null 2>&1; then \
  status=0; \
  else \
  status=1; \
  fi; \
  fi; \
  end_ms="$(now_ms)"; \
  latency="$((end_ms - start_ms))"; \
  if [ "$status" -eq 0 ]; then \
  echo "go proxy probe ok: $label (${latency}ms) proxy=$proxy sumdb=$sumdb" >&2; \
  else \
  echo "go proxy probe failed: $label (${latency}ms) proxy=$proxy sumdb=$sumdb" >&2; \
  fi; \
  printf '%s|%s|%s|%s|%s\n' "$status" "$latency" "$label" "$proxy" "$sumdb" >>"$rank_file"; \
  }; \
  probe_proxy "https://proxy.golang.org,direct" "sum.golang.org" "official"; \
  probe_proxy "https://goproxy.cn,direct" "sum.golang.google.cn" "goproxy-cn"; \
  probe_proxy "direct" "sum.golang.org" "direct"; \
  probe_proxy "https://goproxy.cn,direct" "off" "goproxy-cn-nosumdb"; \
  sort -t '|' -k1,1n -k2,2n "$rank_file" >"${rank_file}.sorted"; \
  echo "go proxy ranked order:" >&2; \
  while IFS='|' read -r status latency label proxy sumdb; do \
  echo "  status=$status latency=${latency}ms label=$label proxy=$proxy sumdb=$sumdb" >&2; \
  done <"${rank_file}.sorted"; \
  install_tool() { \
  pkg="$1"; \
  while IFS='|' read -r _status _latency label proxy sumdb; do \
  tries=0; \
  while [ "$tries" -lt 2 ]; do \
  tries=$((tries + 1)); \
  if GOSUMDB="$sumdb" GOPROXY="$proxy" GIT_TERMINAL_PROMPT=0 go install -v "$pkg"; then \
  echo "install success for $pkg via $label (proxy=$proxy sumdb=$sumdb)" >&2; \
  return 0; \
  fi; \
  echo "install failed for $pkg via $label (attempt $tries), retrying..." >&2; \
  sleep $((tries * 2)); \
  done; \
  done <"${rank_file}.sorted"; \
  echo "install failed for $pkg after trying all proxy candidates" >&2; \
  exit 1; \
  }; \
  install_tool github.com/projectdiscovery/httpx/cmd/httpx@v1.8.1; \
  install_tool github.com/projectdiscovery/naabu/v2/cmd/naabu@v2.4.0; \
  install_tool github.com/projectdiscovery/subfinder/v2/cmd/subfinder@v2.12.0; \
  install_tool github.com/projectdiscovery/nuclei/v3/cmd/nuclei@v3.7.0; \
  install_tool github.com/projectdiscovery/katana/cmd/katana@v1.4.0; \
  install_tool github.com/ffuf/ffuf/v2@v2.1.0; \
  install_tool github.com/d3mondev/puredns/v2@v2.1.1; \
  install_tool github.com/yyhuni/xingfinger@v0.0.0-20251228093343-bd42c7be5a70; \
  install_tool github.com/hahwul/dalfox/v2@v2.12.0; \
  rm -f "$rank_file" "${rank_file}.sorted"

# ============================================
# Stage 2: Build Nmap (pin latest upstream release)
# ============================================
FROM ubuntu:noble-20260113 AS nmap-builder

ARG DEBIAN_FRONTEND=noninteractive
ARG NMAP_VERSION=7.98

RUN apt-get update && apt-get install -y --no-install-recommends \
  build-essential \
  ca-certificates \
  curl \
  libpcap-dev \
  libssl-dev \
  && rm -rf /var/lib/apt/lists/*

RUN set -eux; \
  curl -fsSL -o /tmp/nmap.tar.bz2 "https://nmap.org/dist/nmap-${NMAP_VERSION}.tar.bz2"; \
  tar -xjf /tmp/nmap.tar.bz2 -C /tmp; \
  cd /tmp/nmap-${NMAP_VERSION}; \
  ./configure --prefix=/usr/local; \
  make -j"$(nproc)"; \
  make install DESTDIR=/nmap-root; \
  rm -rf /tmp/nmap*;

# ============================================
# Stage 3: Build Go Worker
# ============================================
FROM golang:1.26 AS worker-builder

ARG GO111MODULE=on
ARG GOPROXY=https://proxy.golang.org,direct

ENV GO111MODULE=$GO111MODULE
ENV GOPROXY=$GOPROXY

WORKDIR /src

# Copy deps first for caching
COPY --from=contracts . /contracts
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
  --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
  set -eux; \
  try_mod_download() { \
  proxy="$1"; \
  sumdb="$2"; \
  label="$3"; \
  echo "go mod download via $label (proxy=$proxy sumdb=$sumdb)" >&2; \
  if GOSUMDB="$sumdb" GOPROXY="$proxy" GIT_TERMINAL_PROMPT=0 go mod download; then \
  return 0; \
  fi; \
  return 1; \
  }; \
  try_mod_download "https://proxy.golang.org,direct" "sum.golang.org" "official" || \
  try_mod_download "https://goproxy.cn,direct" "sum.golang.google.cn" "goproxy-cn" || \
  try_mod_download "direct" "sum.golang.org" "direct" || \
  try_mod_download "https://goproxy.cn,direct" "off" "goproxy-cn-nosumdb"

# Copy source and build
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
  --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
  set -eux; \
  try_go_build() { \
  proxy="$1"; \
  sumdb="$2"; \
  label="$3"; \
  echo "go build via $label (proxy=$proxy sumdb=$sumdb)" >&2; \
  if GOSUMDB="$sumdb" GOPROXY="$proxy" GIT_TERMINAL_PROMPT=0 CGO_ENABLED=0 go build -o /worker ./cmd/worker; then \
  return 0; \
  fi; \
  return 1; \
  }; \
  try_go_build "https://proxy.golang.org,direct" "sum.golang.org" "official" || \
  try_go_build "https://goproxy.cn,direct" "sum.golang.google.cn" "goproxy-cn" || \
  try_go_build "direct" "sum.golang.org" "direct" || \
  try_go_build "https://goproxy.cn,direct" "off" "goproxy-cn-nosumdb"

# ============================================
# Stage 4: Runtime image
# ============================================
FROM ubuntu:noble-20260113

ENV DEBIAN_FRONTEND=noninteractive
ARG URO_VERSION=1.0.2
ARG WAYMORE_VERSION=8.5
ARG DNSGEN_VERSION=1.0.4
ARG MASSCAN_APT_VERSION=2:1.3.2+ds1-1

WORKDIR /app

# Install runtime deps + Python environment
RUN apt-get update && apt-get install -y --no-install-recommends \
  ca-certificates \
  libpcap-dev \
  "masscan=${MASSCAN_APT_VERSION}" \
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
RUN pipx install "uro==${URO_VERSION}" && \
  pipx install "waymore==${WAYMORE_VERSION}" && \
  pipx install "dnsgen==${DNSGEN_VERSION}"

# Copy prebuilt scanning tools from tools builder
COPY --from=tools-builder /go/bin/ /opt/lunafox-tools/bin
COPY --from=tools-builder /usr/local/bin/massdns /opt/lunafox-tools/bin/massdns
COPY --from=nmap-builder /nmap-root/usr/local/ /usr/local/

# Copy Go Worker
COPY --from=worker-builder /worker /usr/local/bin/worker

# Run as root by default:
# - Avoid host mount permission issues
# - Some scanners (e.g., masscan) need elevated privileges
# For least privilege, switch to a non-root user and add required capabilities.
CMD ["worker"]
