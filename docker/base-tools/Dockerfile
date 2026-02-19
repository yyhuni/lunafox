# syntax=docker/dockerfile:1.7

FROM ubuntu:22.04

ARG DEBIAN_FRONTEND=noninteractive
ARG SHFMT_VERSION=v3.10.0

RUN apt-get update && apt-get install -y --no-install-recommends \
    bash \
    ca-certificates \
    curl \
    git \
    jq \
    make \
    python3 \
    ripgrep \
    shellcheck \
    xz-utils \
    && rm -rf /var/lib/apt/lists/*

RUN set -eux; \
    arch="$(dpkg --print-architecture)"; \
    case "$arch" in \
      amd64) shfmt_arch="amd64" ;; \
      arm64) shfmt_arch="arm64" ;; \
      *) echo "unsupported architecture: $arch" >&2; exit 1 ;; \
    esac; \
    curl -fsSL -o /tmp/shfmt "https://github.com/mvdan/sh/releases/download/${SHFMT_VERSION}/shfmt_${SHFMT_VERSION}_linux_${shfmt_arch}"; \
    install -m 0755 /tmp/shfmt /usr/local/bin/shfmt; \
    rm -f /tmp/shfmt; \
    shfmt -version

CMD ["bash"]
