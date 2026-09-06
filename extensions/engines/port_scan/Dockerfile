# syntax=docker/dockerfile:1.7

# lunafox:dockerfile-mode=standard
# lunafox:generated:begin
ARG ENGINE_GO_VERSION=1.26
ARG UBUNTU_BASE=ubuntu:noble-20260113
ARG ENGINE_IMAGE_VERSION
ARG ENGINE_IMAGE_SOURCE=https://github.com/yyhuni/lunafox
ARG GOPROXY=https://proxy.golang.org,direct

# lunafox:engine-owned:build:begin
ARG NAABU_VERSION=v2.4.0
FROM --platform=$BUILDPLATFORM alpine:3.23.2 AS naabu-downloader-base

FROM naabu-downloader-base AS naabu-downloader
ARG TARGETARCH
ARG NAABU_VERSION

WORKDIR /out
RUN set -eux; \
    case "$TARGETARCH" in \
      amd64) \
        archive="naabu_2.4.0_linux_amd64.zip"; \
        checksum="13972348cfa00d3fb18f8ecd4b62a9c3b57bc75ef037cea87cd4bed3463dca02" \
        ;; \
      arm64) \
        archive="naabu_2.4.0_linux_arm64.zip"; \
        checksum="a6d67c1e2a0ced4347512dd3265458b50b207908b17c9c76d38a755f5536c58b" \
        ;; \
      *) echo "unsupported Naabu target architecture: $TARGETARCH" >&2; exit 1 ;; \
    esac; \
    url="https://github.com/projectdiscovery/naabu/releases/download/${NAABU_VERSION}/${archive}"; \
    attempt=1; \
    until wget -T 60 -O "/tmp/$archive" "$url"; do \
      test "$attempt" -lt 5; \
      attempt=$((attempt + 1)); \
      rm -f "/tmp/$archive"; \
      sleep 2; \
    done; \
    printf '%s  %s\n' "$checksum" "/tmp/$archive" | sha256sum -c -; \
    unzip -q "/tmp/$archive" naabu -d /out; \
    test -f /out/naabu; \
    chmod 0755 /out/naabu; \
    test -x /out/naabu; \
    rm -f "/tmp/$archive"
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

WORKDIR /src/port_scan
COPY --from=contracts . /contracts
COPY --from=engine-go . /engine-go
COPY port_scan/go.mod port_scan/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    go mod download
COPY port_scan ./
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    go build -trimpath -ldflags='-s -w' \
      -o /out/port-scan-runtime-engine \
      ./cmd/port-scan-engine

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
ARG NAABU_VERSION
RUN apt-get update && apt-get install -y --no-install-recommends libpcap0.8t64 \
    && rm -rf /var/lib/apt/lists/*
COPY --from=naabu-downloader /out/naabu /opt/lunafox-tools/bin/naabu
RUN set -eux; \
    printf '%s\n' "naabu=${NAABU_VERSION}" > /opt/lunafox-tools/VERSIONS; \
    chmod 0755 /opt/lunafox-tools/bin/naabu
# lunafox:engine-owned:runtime-base:end

COPY --from=engine-builder /out/port-scan-runtime-engine \
  /opt/lunafox-engine/bin/port-scan-runtime-engine
RUN chmod 0755 /opt/lunafox-engine/bin/port-scan-runtime-engine

FROM runtime-base AS verify
# lunafox:engine-owned:verify:begin
# Add Engine-owned verify-stage additions only when required.
# lunafox:engine-owned:verify:end
RUN --mount=type=bind,source=port_scan/tests/container,target=/run/lunafox-conformance \
    /bin/sh /run/lunafox-conformance/container-conformance.sh \
      /opt/lunafox-engine/bin/port-scan-runtime-engine \
    && : > /conformance-passed

FROM runtime-base AS runtime
RUN --mount=type=bind,from=verify,source=/conformance-passed,target=/run/.conformance-passed \
    test -f /run/.conformance-passed

# lunafox:engine-owned:runtime-final:begin
# No final-stage additions are required by Port Scan.
# lunafox:engine-owned:runtime-final:end
# Canonical task targets remain Agent-owned mounts: /run/lunafox/context/execution.pb,
# /run/lunafox/credential/token, /run/lunafox/socket/engine.sock,
# /run/lunafox/inputs/subdomains.txt, /run/lunafox/inputs/host-ports.jsonl,
# /run/lunafox/inputs/website-urls.txt, and /run/lunafox/inputs/endpoint-urls.txt.
USER 0:0
STOPSIGNAL SIGTERM
ENTRYPOINT ["/opt/lunafox-engine/bin/port-scan-runtime-engine"]
CMD []
# lunafox:generated:end
