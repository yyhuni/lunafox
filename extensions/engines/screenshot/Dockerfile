# syntax=docker/dockerfile:1.7

# lunafox:dockerfile-mode=custom

ARG ENGINE_GO_VERSION=1.26
ARG TOOLS_GO_VERSION=1.26
ARG UBUNTU_BASE=debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818
ARG ENGINE_IMAGE_VERSION
ARG ENGINE_IMAGE_SOURCE=https://github.com/yyhuni/lunafox
ARG HTTPX_VERSION=v1.10.0
ARG CHROMIUM_VERSION=v131.0.6778
ARG CHROMIUM_BUILD=85
ARG CHROMIUM_PACKAGE_VERSION=131.0.6778.85-1~deb12u1
ARG CHROMIUM_AMD64_SHA256=2568e4727db01a3f2090ba3a27f473a89ff5014c3025ea2d0cfc72fa6466ffb0
ARG CHROMIUM_ARM64_SHA256=8f5fc4830b7030b706ccc868e712451330e5eaa4076264f95da3c6ec80007112
ARG CHROMIUM_COMMON_AMD64_SHA256=7b1b673f0331b2d9774617504b864a0243d91a3a108f67c61903ae8ff8ba40ed
ARG CHROMIUM_COMMON_ARM64_SHA256=2c0822f1664de1940ff4c0bd6a0f6225fdeac422753ac57943a68711d783cca6
ARG LIBWEBP_VERSION=v1.6.0
ARG LIBWEBP_SHA256=93a852c2b3efafee3723efd4636de855b46f9fe1efddd607e1f42f60fc8f2136
ARG GOPROXY=https://proxy.golang.org,direct

FROM --platform=$BUILDPLATFORM public.ecr.aws/docker/library/golang:1.26-bookworm AS httpx-builder
ARG TARGETOS=linux
ARG TARGETARCH
ARG HTTPX_VERSION
ARG GOPROXY
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH GOTOOLCHAIN=local GOPROXY=$GOPROXY
WORKDIR /out
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    set -eux; \
    GOPATH=/go GOBIN= go install "github.com/projectdiscovery/httpx/cmd/httpx@${HTTPX_VERSION}"; \
    for candidate in "/go/bin/${TARGETOS}_${TARGETARCH}/httpx" "/go/bin/httpx"; do \
      if [ -x "$candidate" ]; then install -m 0755 "$candidate" /out/httpx; break; fi; \
    done; \
    test -x /out/httpx

FROM --platform=$TARGETPLATFORM ubuntu:noble AS cwebp-builder
ARG LIBWEBP_VERSION
ARG LIBWEBP_SHA256
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl build-essential cmake libpng-dev \
    && rm -rf /var/lib/apt/lists/* \
    && curl -fsSL "https://github.com/webmproject/libwebp/archive/refs/tags/${LIBWEBP_VERSION}.tar.gz" -o /tmp/libwebp.tar.gz \
    && printf '%s  %s\n' "$LIBWEBP_SHA256" /tmp/libwebp.tar.gz | sha256sum -c - \
    && mkdir -p /src \
    && tar -xzf /tmp/libwebp.tar.gz -C /src \
    && cmake -S "/src/libwebp-${LIBWEBP_VERSION#v}" -B /src/build -DCMAKE_BUILD_TYPE=Release -DBUILD_SHARED_LIBS=OFF -DWEBP_BUILD_CWEBP=ON -DWEBP_BUILD_DWEBP=OFF -DWEBP_BUILD_EXTRAS=OFF -DWEBP_BUILD_GIF2WEBP=OFF -DWEBP_BUILD_IMG2WEBP=OFF -DWEBP_BUILD_VWEBP=OFF -DWEBP_BUILD_WEBPINFO=OFF -DWEBP_BUILD_WEBPMUX=OFF \
    && cmake --build /src/build --target cwebp --parallel \
    && install -d -m 0755 /out/bin \
    && for candidate in /src/build/sharpyuv/cwebp /src/build/cwebp /src/build/examples/cwebp; do \
         if [ -x "$candidate" ]; then install -m 0755 "$candidate" /out/bin/cwebp; break; fi; \
       done \
    && test -x /out/bin/cwebp \
    && /out/bin/cwebp -version | grep -F '1.6.0'

FROM --platform=$BUILDPLATFORM public.ecr.aws/docker/library/golang:${ENGINE_GO_VERSION}-bookworm AS engine-builder
ARG TARGETOS=linux
ARG TARGETARCH
ARG GOPROXY
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH GOTOOLCHAIN=local GOPROXY=$GOPROXY
WORKDIR /src/screenshot
# Copy the Engine's independent module before resolving its local engine-go
# replacement from the named context.
COPY --from=contracts . /contracts
COPY --from=engine-go . /engine-go
COPY screenshot/go.mod screenshot/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    go mod download
COPY screenshot ./
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    go build -trimpath -ldflags='-s -w' -o /out/screenshot-runtime-engine ./cmd/screenshot-engine

FROM --platform=$TARGETPLATFORM ${UBUNTU_BASE} AS runtime-base
ARG ENGINE_IMAGE_VERSION
ARG ENGINE_IMAGE_SOURCE
ARG HTTPX_VERSION
ARG CHROMIUM_VERSION
ARG CHROMIUM_BUILD
ARG CHROMIUM_PACKAGE_VERSION
ARG CHROMIUM_AMD64_SHA256
ARG CHROMIUM_ARM64_SHA256
ARG CHROMIUM_COMMON_AMD64_SHA256
ARG CHROMIUM_COMMON_ARM64_SHA256
ARG LIBWEBP_VERSION
ARG TARGETARCH
RUN test -n "$ENGINE_IMAGE_VERSION" && test -n "$ENGINE_IMAGE_SOURCE"
LABEL org.opencontainers.image.source="${ENGINE_IMAGE_SOURCE}" \
      org.opencontainers.image.version="${ENGINE_IMAGE_VERSION}"
ENV PATH=/opt/lunafox-tools/bin:$PATH
WORKDIR /workspace
# Chromium is installed from a dated Debian security snapshot so both target
# architectures use the same browser build. The package bytes are checked
# against the snapshot's published SHA256 values before apt resolves runtime
# dependencies.
# HTTPX JSON output initializes its optional DIT classifier. A local empty
# model makes that optional path fail closed instead of downloading a model
# into the task workspace or reaching the network at runtime.
RUN set -eux; \
    printf '%s\n' 'deb [check-valid-until=no] http://snapshot.debian.org/archive/debian-security/20241123T102846Z bookworm-security main' > /etc/apt/sources.list.d/lunafox-chromium.list; \
    apt-get update; \
    apt-get install -y --no-install-recommends ca-certificates python3 fonts-liberation; \
    apt-get download "chromium=${CHROMIUM_PACKAGE_VERSION}" "chromium-common=${CHROMIUM_PACKAGE_VERSION}"; \
    case "$TARGETARCH" in \
      amd64) chromium_sha="$CHROMIUM_AMD64_SHA256"; common_sha="$CHROMIUM_COMMON_AMD64_SHA256" ;; \
      arm64) chromium_sha="$CHROMIUM_ARM64_SHA256"; common_sha="$CHROMIUM_COMMON_ARM64_SHA256" ;; \
      *) echo "unsupported Chromium architecture: $TARGETARCH" >&2; exit 1 ;; \
    esac; \
    chromium_deb="$(find / -maxdepth 2 -name "chromium_${CHROMIUM_PACKAGE_VERSION}_*.deb" -print -quit)"; \
    common_deb="$(find / -maxdepth 2 -name "chromium-common_${CHROMIUM_PACKAGE_VERSION}_*.deb" -print -quit)"; \
    test -n "$chromium_deb" && test -n "$common_deb"; \
    printf '%s  %s\n' "$chromium_sha" "$chromium_deb" | sha256sum -c -; \
    printf '%s  %s\n' "$common_sha" "$common_deb" | sha256sum -c -; \
    apt-get install -y --no-install-recommends "$chromium_deb" "$common_deb"; \
    rm -f "$chromium_deb" "$common_deb" /etc/apt/sources.list.d/lunafox-chromium.list; \
    rm -rf /var/lib/apt/lists/*; \
    chromium --version | grep -F "${CHROMIUM_VERSION#v}.${CHROMIUM_BUILD}"; \
    install -d -m 0755 /opt/lunafox-engine/bin /opt/lunafox-tools/bin \
      /run/lunafox/context /run/lunafox/credential /run/lunafox/inputs \
      /run/lunafox/resources/config /run/lunafox/resources/platform \
      /run/lunafox/socket /workspace /root/.dit; \
    printf '{}\n' > /root/.dit/model.json
COPY --from=engine-builder /out/screenshot-runtime-engine /opt/lunafox-engine/bin/screenshot-runtime-engine
COPY --from=httpx-builder /out/httpx /opt/lunafox-tools/bin/httpx
COPY --from=cwebp-builder /out/bin/cwebp /opt/lunafox-tools/bin/cwebp
RUN set -eux; \
    printf '%s\n' \
      'export CHROMIUM_FLAGS="$CHROMIUM_FLAGS --no-zygote"' \
      'export CHROMIUM_FLAGS="$CHROMIUM_FLAGS --disable-gpu"' \
      'export CHROMIUM_FLAGS="$CHROMIUM_FLAGS --disable-gpu-rasterization"' \
      'export CHROMIUM_FLAGS="$CHROMIUM_FLAGS --disable-features=UseOzonePlatform"' \
      'export CHROMIUM_FLAGS="$CHROMIUM_FLAGS --disable-dev-shm-usage"' \
      > /etc/chromium.d/99-lunafox-headless; \
    printf '%s\n' "httpx=${HTTPX_VERSION}" "chromium=${CHROMIUM_VERSION}" "cwebp=${LIBWEBP_VERSION}" > /opt/lunafox-tools/VERSIONS; \
    ln -s /usr/bin/chromium /opt/lunafox-tools/bin/chromium; \
    chmod 0755 /opt/lunafox-engine/bin/screenshot-runtime-engine

FROM runtime-base AS verify
RUN --network=none --mount=type=bind,source=screenshot/tests/container,target=/run/lunafox-conformance \
    /bin/sh /run/lunafox-conformance/container-conformance.sh /opt/lunafox-engine/bin/screenshot-runtime-engine \
    && : > /conformance-passed

FROM runtime-base AS runtime
RUN --mount=type=bind,from=verify,source=/conformance-passed,target=/run/.conformance-passed \
    test -f /run/.conformance-passed

# Agent attaches fixed task-scoped ABI mounts without environment overrides.
# Canonical input paths for the shared image
# contract are /run/lunafox/inputs/subdomains.txt,
# /run/lunafox/inputs/host-ports.jsonl, /run/lunafox/inputs/website-urls.txt,
# and /run/lunafox/inputs/endpoint-urls.txt.
# Chromium intentionally runs as root without browser sandboxing; this is the
# accepted image isolation boundary for the first Screenshot release.
# Canonical targets: /run/lunafox/context/execution.pb,
# /run/lunafox/socket/engine.sock, /run/lunafox/credential/token,
# /run/lunafox/inputs/website-urls.txt, and /workspace.
USER 0:0
STOPSIGNAL SIGTERM
ENTRYPOINT ["/opt/lunafox-engine/bin/screenshot-runtime-engine"]
CMD []
