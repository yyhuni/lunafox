# syntax=docker/dockerfile:1.7

# lunafox:dockerfile-mode=standard
# lunafox:generated:begin
ARG ENGINE_GO_VERSION=1.26
ARG UBUNTU_BASE=ubuntu:noble-20260113
ARG ENGINE_IMAGE_VERSION
ARG ENGINE_IMAGE_SOURCE=https://github.com/yyhuni/lunafox
ARG GOPROXY=https://proxy.golang.org,direct

# lunafox:engine-owned:build:begin
ARG TOOLS_GO_VERSION=1.24
ARG HTTPX_VERSION=v1.8.1
FROM --platform=$BUILDPLATFORM public.ecr.aws/docker/library/golang:${TOOLS_GO_VERSION}-bookworm AS go-tools
ARG TARGETOS=linux
ARG TARGETARCH
ARG HTTPX_VERSION
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
    GOPATH="$gopath" GOBIN= go install "github.com/projectdiscovery/httpx/cmd/httpx@${HTTPX_VERSION}"; \
    httpx_path="$gopath/bin/httpx"; \
    [ -f "$httpx_path" ] || httpx_path="$gopath/bin/${TARGETOS}_${TARGETARCH}/httpx"; \
    install -m 0755 "$httpx_path" /out/httpx
# lunafox:engine-owned:build:end

FROM --platform=$BUILDPLATFORM public.ecr.aws/docker/library/golang:${ENGINE_GO_VERSION}-bookworm AS engine-builder
ARG TARGETOS=linux
ARG TARGETARCH
ARG GOPROXY

ENV CGO_ENABLED=0 \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH \
    GOTOOLCHAIN=local \
    GOPROXY=$GOPROXY

WORKDIR /src/website_discovery
COPY --from=contracts . /contracts
COPY --from=engine-go . /engine-go
COPY website_discovery/go.mod website_discovery/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    go mod download
COPY website_discovery ./
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    go build -trimpath -ldflags='-s -w' \
      -o /out/website-discovery-runtime-engine \
      ./cmd/website-discovery-engine

FROM --platform=$TARGETPLATFORM ${UBUNTU_BASE} AS runtime-base
ARG ENGINE_IMAGE_VERSION
ARG ENGINE_IMAGE_SOURCE

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

# lunafox:engine-owned:runtime-base:begin
ARG HTTPX_VERSION
COPY --from=go-tools /out/httpx /opt/lunafox-tools/bin/httpx
RUN set -eux; \
    printf '%s\n' "httpx=${HTTPX_VERSION}" > /opt/lunafox-tools/VERSIONS; \
    chmod 0755 /opt/lunafox-tools/bin/httpx
# lunafox:engine-owned:runtime-base:end

COPY --from=engine-builder /out/website-discovery-runtime-engine \
  /opt/lunafox-engine/bin/website-discovery-runtime-engine
RUN chmod 0755 /opt/lunafox-engine/bin/website-discovery-runtime-engine

FROM runtime-base AS verify
# lunafox:engine-owned:verify:begin
# Add Engine-owned verify-stage additions only when required.
# lunafox:engine-owned:verify:end
RUN --mount=type=bind,source=website_discovery/tests/container,target=/run/lunafox-conformance \
    /bin/sh /run/lunafox-conformance/container-conformance.sh \
      /opt/lunafox-engine/bin/website-discovery-runtime-engine \
    && : > /conformance-passed

FROM runtime-base AS runtime
RUN --mount=type=bind,from=verify,source=/conformance-passed,target=/run/.conformance-passed \
    test -f /run/.conformance-passed

# lunafox:engine-owned:runtime-final:begin
# No final-stage additions are required by Website Discovery.
# lunafox:engine-owned:runtime-final:end
# Canonical task targets remain Agent-owned mounts: /run/lunafox/context/execution.pb,
# /run/lunafox/credential/token, /run/lunafox/socket/engine.sock,
# /run/lunafox/inputs/subdomains.txt, /run/lunafox/inputs/host-ports.jsonl,
# /run/lunafox/inputs/website-urls.txt, and /run/lunafox/inputs/endpoint-urls.txt.
USER 0:0
STOPSIGNAL SIGTERM
ENTRYPOINT ["/opt/lunafox-engine/bin/website-discovery-runtime-engine"]
CMD []
# lunafox:generated:end
