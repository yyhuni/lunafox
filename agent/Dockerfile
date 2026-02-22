# ============================================
# Go Agent - build
# ============================================
FROM golang:1.26 AS builder

ARG GO111MODULE=on
ARG GOPROXY=https://proxy.golang.org,direct

ENV GO111MODULE=$GO111MODULE
ENV GOPROXY=$GOPROXY

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    set -eux; \
    try_mod_download() { \
      proxy="$1"; \
      sumdb="$2"; \
      label="$3"; \
      echo "agent go mod download via $label (proxy=$proxy sumdb=$sumdb)" >&2; \
      if GOSUMDB="$sumdb" GOPROXY="$proxy" GIT_TERMINAL_PROMPT=0 go mod download; then \
        return 0; \
      fi; \
      return 1; \
    }; \
    try_mod_download "https://proxy.golang.org,direct" "sum.golang.org" "official" || \
      try_mod_download "https://goproxy.cn,direct" "sum.golang.google.cn" "goproxy-cn" || \
      try_mod_download "direct" "sum.golang.org" "direct" || \
      try_mod_download "https://goproxy.cn,direct" "off" "goproxy-cn-nosumdb"

# Copy source
COPY . .

# Build (static where possible)
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    set -eux; \
    try_go_build() { \
      proxy="$1"; \
      sumdb="$2"; \
      label="$3"; \
      echo "agent go build via $label (proxy=$proxy sumdb=$sumdb)" >&2; \
      if GOSUMDB="$sumdb" GOPROXY="$proxy" GIT_TERMINAL_PROMPT=0 CGO_ENABLED=0 go build -o /out/agent ./cmd/agent; then \
        return 0; \
      fi; \
      return 1; \
    }; \
    try_go_build "https://proxy.golang.org,direct" "sum.golang.org" "official" || \
      try_go_build "https://goproxy.cn,direct" "sum.golang.google.cn" "goproxy-cn" || \
      try_go_build "direct" "sum.golang.org" "direct" || \
      try_go_build "https://goproxy.cn,direct" "off" "goproxy-cn-nosumdb"

# ============================================
# Go Agent - runtime
# ============================================
FROM debian:bookworm-20260112-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /out/agent /usr/local/bin/agent

CMD ["agent"]
