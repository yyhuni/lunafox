# syntax=docker/dockerfile:1.7

# lunafox:dockerfile-mode=custom

# The Runtime Image builds and starts only the generated Engine API v2 author
# facade; no legacy runtime.v1 server entrypoint is included.

ARG ENGINE_GO_VERSION=1.26
ARG TOOLS_GO_VERSION=1.24
ARG UBUNTU_BASE=ubuntu:noble-20260113
ARG ENGINE_IMAGE_VERSION
ARG ENGINE_IMAGE_SOURCE=https://github.com/yyhuni/lunafox

# Keep these versions in the image definition. Release tooling may override
# them only when the corresponding engine/tool contract is updated together.
ARG SUBFINDER_VERSION=v2.12.0
ARG PUREDNS_VERSION=v2.1.2-0.20260223162428-46bd4c963ec2
ARG MASSDNS_VERSION=v1.1.0
ARG GOPROXY=https://proxy.golang.org,direct

FROM --platform=$BUILDPLATFORM public.ecr.aws/docker/library/golang:${TOOLS_GO_VERSION}-bookworm AS go-tools
ARG TARGETOS=linux
ARG TARGETARCH
ARG SUBFINDER_VERSION
ARG PUREDNS_VERSION
ARG GOPROXY

ENV CGO_ENABLED=0 \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH \
    GOTOOLCHAIN=local \
    GOPROXY=$GOPROXY

WORKDIR /out
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    set -eux; \
    install_dir=/out/bin; \
    gopath=/go; \
    mkdir -p "$install_dir"; \
    GOPATH="$gopath" GOBIN= go install "github.com/projectdiscovery/subfinder/v2/cmd/subfinder@${SUBFINDER_VERSION}"; \
    GOPATH="$gopath" GOBIN= go install "github.com/d3mondev/puredns/v2@${PUREDNS_VERSION}"; \
    subfinder_path="$gopath/bin/subfinder"; \
    [ -f "$subfinder_path" ] || subfinder_path="$gopath/bin/${TARGETOS}_${TARGETARCH}/subfinder"; \
    puredns_path="$gopath/bin/puredns"; \
    [ -f "$puredns_path" ] || puredns_path="$gopath/bin/${TARGETOS}_${TARGETARCH}/puredns"; \
    install -m 0755 "$subfinder_path" /out/subfinder; \
    install -m 0755 "$puredns_path" /out/puredns

# MassDNS contains native pieces, so this stage runs on the target platform.
# Buildx supplies emulation when the target differs from the host.
FROM --platform=$TARGETPLATFORM ${UBUNTU_BASE} AS native-tools
ARG MASSDNS_VERSION
ARG DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
      build-essential \
      ca-certificates \
      git \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /tmp
RUN set -eux; \
    git clone --branch "${MASSDNS_VERSION}" --depth=1 \
      https://github.com/blechschmidt/massdns.git massdns; \
    make -C massdns; \
    install -m 0755 massdns/bin/massdns /out-massdns

FROM --platform=$BUILDPLATFORM public.ecr.aws/docker/library/golang:${ENGINE_GO_VERSION}-bookworm AS engine-builder
ARG TARGETOS=linux
ARG TARGETARCH
ARG GOPROXY

ENV CGO_ENABLED=0 \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH \
    GOTOOLCHAIN=local \
    GOPROXY=$GOPROXY

WORKDIR /src/subdomain_discovery
# The canonical build context is extensions/engines. Copy the Engine's
# independent module before resolving its local engine-go replacement from the
# named context without widening the repository root context.
COPY --from=contracts . /contracts
COPY --from=engine-go . /engine-go
COPY subdomain_discovery/go.mod subdomain_discovery/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    go mod download
COPY subdomain_discovery ./
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    go build -trimpath -ldflags='-s -w' \
      -o /out/subdomain-discovery-runtime-engine \
      ./cmd/subdomain-discovery-engine

FROM --platform=$TARGETPLATFORM ${UBUNTU_BASE} AS runtime-base
ARG ENGINE_IMAGE_VERSION
ARG ENGINE_IMAGE_SOURCE
ARG SUBFINDER_VERSION
ARG PUREDNS_VERSION
ARG MASSDNS_VERSION

RUN test -n "$ENGINE_IMAGE_VERSION" && test -n "$ENGINE_IMAGE_SOURCE"

LABEL org.opencontainers.image.source="${ENGINE_IMAGE_SOURCE}" \
      org.opencontainers.image.version="${ENGINE_IMAGE_VERSION}"

ENV PATH=/opt/lunafox-tools/bin:$PATH

WORKDIR /workspace
RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && install -d -m 0755 \
      /opt/lunafox-engine/bin \
      /opt/lunafox-tools/bin \
      /run/lunafox/context \
      /run/lunafox/credential \
      /run/lunafox/inputs \
      /run/lunafox/resources/config \
      /run/lunafox/resources/platform \
      /run/lunafox/socket \
      /workspace

COPY --from=engine-builder /out/subdomain-discovery-runtime-engine \
  /opt/lunafox-engine/bin/subdomain-discovery-runtime-engine
COPY --from=go-tools /out/subfinder /opt/lunafox-tools/bin/subfinder
COPY --from=go-tools /out/puredns /opt/lunafox-tools/bin/puredns
COPY --from=native-tools /out-massdns /opt/lunafox-tools/bin/massdns.real

# MassDNS has no stable upstream --version flag. Keep the real executable
# private and expose a transparent wrapper with a deterministic version marker.
RUN set -eux; \
    printf '%s\n' '#!/bin/sh' \
      'if [ "${1:-}" = "--version" ] || [ "${1:-}" = "-version" ]; then' \
      "  echo \"massdns ${MASSDNS_VERSION}\"" \
      '  exit 0' \
      'fi' \
      'exec /opt/lunafox-tools/bin/massdns.real "$@"' \
      > /opt/lunafox-tools/bin/massdns; \
    chmod 0755 /opt/lunafox-tools/bin/massdns; \
    printf '%s\n' \
      "subfinder=${SUBFINDER_VERSION}" \
      "puredns=${PUREDNS_VERSION}" \
      "massdns=${MASSDNS_VERSION}" \
      > /opt/lunafox-tools/VERSIONS; \
    chmod 0755 /opt/lunafox-engine/bin/subdomain-discovery-runtime-engine

FROM runtime-base AS verify
RUN --mount=type=bind,source=subdomain_discovery/tests/container,target=/run/lunafox-conformance \
    /bin/sh /run/lunafox-conformance/container-conformance.sh \
      /opt/lunafox-engine/bin/subdomain-discovery-runtime-engine \
    && : > /conformance-passed

# The final stage consumes only this marker, forcing source-owned verification
# during the default build without retaining test files in the Runtime Image.
FROM runtime-base AS runtime
RUN --mount=type=bind,from=verify,source=/conformance-passed,target=/run/.conformance-passed \
    test -f /run/.conformance-passed

# The binary is an Engine API v2 client and exits when its single handler run
# completes; it never exposes the legacy Worker/runtime.v1 server protocol.
# The Engine reads the fixed binary ABI mounts directly. The image deliberately
# defines no bootstrap environment override or package-overridable command.
# Canonical targets: /run/lunafox/context/execution.pb,
# /run/lunafox/socket/engine.sock, /run/lunafox/credential/token,
# /run/lunafox/inputs/subdomains.txt, /run/lunafox/inputs/host-ports.jsonl,
# /run/lunafox/inputs/website-urls.txt, and /run/lunafox/inputs/endpoint-urls.txt.
USER 0:0
STOPSIGNAL SIGTERM
ENTRYPOINT ["/opt/lunafox-engine/bin/subdomain-discovery-runtime-engine"]
CMD []
