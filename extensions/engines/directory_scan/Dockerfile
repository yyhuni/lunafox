# syntax=docker/dockerfile:1.7

# lunafox:dockerfile-mode=standard
# lunafox:generated:begin
ARG ENGINE_GO_VERSION=1.26
ARG UBUNTU_BASE=ubuntu:noble-20260113
ARG ENGINE_IMAGE_VERSION
ARG ENGINE_IMAGE_SOURCE=https://github.com/yyhuni/lunafox
ARG GOPROXY=https://proxy.golang.org,direct

# lunafox:engine-owned:build:begin
ARG FFUF_VERSION=v2.2.1
FROM --platform=$BUILDPLATFORM alpine:3.23.2 AS ffuf-downloader
ARG TARGETARCH
ARG FFUF_VERSION

WORKDIR /out
RUN set -eux; \
    case "$TARGETARCH" in \
      amd64) \
        archive="ffuf_2.2.1_linux_amd64.tar.gz"; \
        checksum="86307885810d3c36ba4a3e9ba5178c2d9027bba0dd7f4ea39e39e7c972b62396" \
        ;; \
      arm64) \
        archive="ffuf_2.2.1_linux_arm64.tar.gz"; \
        checksum="89ad4f50345e6a9a48ecc8d241811d582cedf96279276b48d666b12b31260484" \
        ;; \
      *) echo "unsupported FFUF target architecture: $TARGETARCH" >&2; exit 1 ;; \
    esac; \
    url="https://github.com/ffuf/ffuf/releases/download/${FFUF_VERSION}/${archive}"; \
    attempt=1; \
    until wget -T 60 -O "/tmp/$archive" "$url"; do \
      test "$attempt" -lt 5; \
      attempt=$((attempt + 1)); \
      rm -f "/tmp/$archive"; \
      sleep 2; \
    done; \
    printf '%s  %s\n' "$checksum" "/tmp/$archive" | sha256sum -c -; \
    tar -xzf "/tmp/$archive" -C /out ffuf; \
    test -f /out/ffuf; \
    chmod 0755 /out/ffuf; \
    test -x /out/ffuf; \
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

WORKDIR /src/directory_scan
COPY --from=contracts . /contracts
COPY --from=engine-go . /engine-go
COPY directory_scan/go.mod directory_scan/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    go mod download
COPY directory_scan ./
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    go build -trimpath -ldflags='-s -w' \
      -o /out/directory-scan-engine \
      ./cmd/directory-scan-engine

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
ARG FFUF_VERSION
RUN install -d -m 0755 /run/lunafox/ffuf-config/ffuf
COPY --from=ffuf-downloader /out/ffuf /opt/lunafox-tools/bin/ffuf
RUN set -eux; \
    printf '%s\n' "ffuf=${FFUF_VERSION}" > /opt/lunafox-tools/VERSIONS; \
    chmod 0755 /opt/lunafox-tools/bin/ffuf
# lunafox:engine-owned:runtime-base:end

COPY --from=engine-builder /out/directory-scan-engine \
  /opt/lunafox-engine/bin/directory-scan-engine
RUN chmod 0755 /opt/lunafox-engine/bin/directory-scan-engine

FROM runtime-base AS verify
# lunafox:engine-owned:verify:begin
ENV XDG_CONFIG_HOME=/run/lunafox/ffuf-config
# lunafox:engine-owned:verify:end
RUN --mount=type=bind,source=directory_scan/tests/container,target=/run/lunafox-conformance \
    /bin/sh /run/lunafox-conformance/container-conformance.sh \
      /opt/lunafox-engine/bin/directory-scan-engine \
    && : > /conformance-passed

FROM runtime-base AS runtime
RUN --mount=type=bind,from=verify,source=/conformance-passed,target=/run/.conformance-passed \
    test -f /run/.conformance-passed

# lunafox:engine-owned:runtime-final:begin
ENV XDG_CONFIG_HOME=/run/lunafox/ffuf-config
# lunafox:engine-owned:runtime-final:end
# Canonical task targets remain Agent-owned mounts: /run/lunafox/context/execution.pb,
# /run/lunafox/credential/token, /run/lunafox/socket/engine.sock,
# /run/lunafox/inputs/subdomains.txt, /run/lunafox/inputs/host-ports.jsonl,
# /run/lunafox/inputs/website-urls.txt, and /run/lunafox/inputs/endpoint-urls.txt.
USER 0:0
STOPSIGNAL SIGTERM
ENTRYPOINT ["/opt/lunafox-engine/bin/directory-scan-engine"]
CMD []
# lunafox:generated:end
