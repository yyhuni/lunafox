# syntax=docker/dockerfile:1.7

ARG BASE_TOOLS_IMAGE=ghcr.io/yyhuni/lunafox-base-tools:stable
FROM ${BASE_TOOLS_IMAGE}

ARG GO_VERSION=1.26.0
ARG NODE_VERSION=20.20.0
ARG PNPM_VERSION=10.30.0

ENV GOPATH=/root/go
ENV GOCACHE=/root/.cache/go-build
ENV GOMODCACHE=/root/go/pkg/mod
ENV PATH=/usr/local/go/bin:/usr/local/bin:${PATH}

RUN set -eux; \
    arch="$(dpkg --print-architecture)"; \
    case "$arch" in \
      amd64) go_arch="amd64" ;; \
      arm64) go_arch="arm64" ;; \
      *) echo "unsupported architecture: $arch" >&2; exit 1 ;; \
    esac; \
    curl -fsSL -o /tmp/go.tar.gz "https://go.dev/dl/go${GO_VERSION}.linux-${go_arch}.tar.gz"; \
    rm -rf /usr/local/go; \
    tar -C /usr/local -xzf /tmp/go.tar.gz; \
    rm -f /tmp/go.tar.gz; \
    go version

RUN set -eux; \
    arch="$(dpkg --print-architecture)"; \
    case "$arch" in \
      amd64) node_arch="x64" ;; \
      arm64) node_arch="arm64" ;; \
      *) echo "unsupported architecture: $arch" >&2; exit 1 ;; \
    esac; \
    mkdir -p /usr/local/lib/nodejs; \
    curl -fsSL -o /tmp/node.tar.xz "https://nodejs.org/dist/v${NODE_VERSION}/node-v${NODE_VERSION}-linux-${node_arch}.tar.xz"; \
    tar -C /usr/local/lib/nodejs -xf /tmp/node.tar.xz; \
    rm -f /tmp/node.tar.xz; \
    ln -sf "/usr/local/lib/nodejs/node-v${NODE_VERSION}-linux-${node_arch}/bin/node" /usr/local/bin/node; \
    ln -sf "/usr/local/lib/nodejs/node-v${NODE_VERSION}-linux-${node_arch}/bin/npm" /usr/local/bin/npm; \
    ln -sf "/usr/local/lib/nodejs/node-v${NODE_VERSION}-linux-${node_arch}/bin/npx" /usr/local/bin/npx; \
    ln -sf "/usr/local/lib/nodejs/node-v${NODE_VERSION}-linux-${node_arch}/bin/corepack" /usr/local/bin/corepack; \
    node -v; \
    npm -v; \
    npm install -g "pnpm@${PNPM_VERSION}"; \
    pnpm_bin="/usr/local/lib/nodejs/node-v${NODE_VERSION}-linux-${node_arch}/bin/pnpm"; \
    [ -x "${pnpm_bin}" ]; \
    ln -sf "${pnpm_bin}" /usr/local/bin/pnpm; \
    which pnpm; \
    pnpm -v

CMD ["bash"]
