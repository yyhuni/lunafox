# syntax=docker/dockerfile:1.7

# lunafox:dockerfile-mode=custom

# The Runtime Image builds and starts only the generated Engine API v2 author
# facade; no legacy runtime.v1 server entrypoint is included.
ARG ENGINE_GO_VERSION=1.26
ARG UBUNTU_BASE=ubuntu:noble-20260113
ARG ENGINE_IMAGE_VERSION
ARG ENGINE_IMAGE_SOURCE=https://github.com/yyhuni/lunafox
ARG OBSERVER_WARD_VERSION=v2026.6.28-lunafox.1
ARG OBSERVER_WARD_COMMIT=65801cf6d4b3dd4bb07ea7a1cf6e849713c42ea6
ARG GOPROXY=https://proxy.golang.org,direct

# Build Observer Ward for the target architecture from the immutable LunaFox
# fork tag. The peeled commit check prevents an annotated tag from silently
# moving.
FROM --platform=$BUILDPLATFORM rust:1.94-bookworm AS observer-ward-builder
ARG TARGETARCH
ARG OBSERVER_WARD_VERSION
ARG OBSERVER_WARD_COMMIT

RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates \
      git \
      gcc-aarch64-linux-gnu \
      libc6-dev-arm64-cross \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src
RUN set -eux; \
    case "$TARGETARCH" in \
      amd64) \
        target="x86_64-unknown-linux-gnu"; \
        ;; \
      arm64) \
        target="aarch64-unknown-linux-gnu"; \
        export CARGO_TARGET_AARCH64_UNKNOWN_LINUX_GNU_LINKER=aarch64-linux-gnu-gcc; \
        export CC_aarch64_unknown_linux_gnu=aarch64-linux-gnu-gcc; \
        ;; \
      *) echo "unsupported Observer Ward target architecture: $TARGETARCH" >&2; exit 1 ;; \
    esac; \
    git clone --branch "$OBSERVER_WARD_VERSION" --depth=1 \
      https://github.com/yyhuni/observer_ward_for_luna.git observer_ward; \
    test "$(git -C observer_ward rev-parse "refs/tags/${OBSERVER_WARD_VERSION}^{commit}")" = "$OBSERVER_WARD_COMMIT"; \
    cd observer_ward; \
    rustup target add "$target"; \
    cargo build --locked --release --target "$target" --package observer_ward; \
    install -m 0755 "target/$target/release/observer_ward" /out-observer-ward

FROM --platform=$BUILDPLATFORM public.ecr.aws/docker/library/golang:${ENGINE_GO_VERSION}-bookworm AS engine-builder
ARG TARGETOS=linux
ARG TARGETARCH
ARG GOPROXY

ENV CGO_ENABLED=0 \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH \
    GOTOOLCHAIN=local \
    GOPROXY=$GOPROXY

WORKDIR /src/fingerprint_detection
# Build context is extensions/engines; copy the Engine's independent module
# before resolving its local engine-go replacement from the named context.
COPY --from=contracts . /contracts
COPY --from=engine-go . /engine-go
COPY fingerprint_detection/go.mod fingerprint_detection/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    go mod download
COPY fingerprint_detection ./
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    go build -trimpath -ldflags='-s -w' \
      -o /out/fingerprint-detection-runtime-engine \
      ./cmd/fingerprint-detection-engine

FROM --platform=$TARGETPLATFORM ${UBUNTU_BASE} AS runtime-base
ARG ENGINE_IMAGE_VERSION
ARG ENGINE_IMAGE_SOURCE
ARG OBSERVER_WARD_VERSION
ARG OBSERVER_WARD_COMMIT

RUN test -n "$ENGINE_IMAGE_VERSION" && test -n "$ENGINE_IMAGE_SOURCE"

LABEL org.opencontainers.image.source="${ENGINE_IMAGE_SOURCE}" \
      org.opencontainers.image.version="${ENGINE_IMAGE_VERSION}"

ENV PATH=/opt/lunafox-tools/bin:$PATH

WORKDIR /workspace
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
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

COPY --from=engine-builder /out/fingerprint-detection-runtime-engine \
  /opt/lunafox-engine/bin/fingerprint-detection-runtime-engine
COPY --from=observer-ward-builder /out-observer-ward \
  /opt/lunafox-tools/bin/observer_ward.real

# The wrapper gives the pinned source release a stable inventory version while
# preserving every normal Observer Ward argument for Engine-owned execution.
RUN set -eux; \
    printf '%s\n' '#!/bin/sh' \
      'if [ "${1:-}" = "--version" ] || [ "${1:-}" = "-version" ]; then' \
      "  echo \"observer_ward ${OBSERVER_WARD_VERSION}\"" \
      '  exit 0' \
      'fi' \
      'exec /opt/lunafox-tools/bin/observer_ward.real "$@"' \
      > /opt/lunafox-tools/bin/observer-ward; \
    chmod 0755 /opt/lunafox-tools/bin/observer-ward /opt/lunafox-tools/bin/observer_ward.real; \
    printf '%s\n' \
      "observer_ward=${OBSERVER_WARD_VERSION}" \
      "observer_ward_commit=${OBSERVER_WARD_COMMIT}" \
      > /opt/lunafox-tools/VERSIONS; \
    chmod 0755 /opt/lunafox-engine/bin/fingerprint-detection-runtime-engine

FROM runtime-base AS verify
RUN --mount=type=bind,source=fingerprint_detection/tests/container,target=/run/lunafox-conformance \
    /bin/sh /run/lunafox-conformance/container-conformance.sh \
      /opt/lunafox-engine/bin/fingerprint-detection-runtime-engine \
    && : > /conformance-passed

# The final runtime stage consumes only the marker, so source-owned tests are
# never retained in the shipped image.
FROM runtime-base AS runtime
RUN --mount=type=bind,from=verify,source=/conformance-passed,target=/run/.conformance-passed \
    test -f /run/.conformance-passed

# The Engine reads the fixed binary ABI mounts directly. The image deliberately
# defines no bootstrap environment override or package-overridable command.
# Canonical mount targets remain Agent-owned and absent until task creation:
# /run/lunafox/context/execution.pb, /run/lunafox/credential/token,
# /run/lunafox/socket/engine.sock, /run/lunafox/inputs/subdomains.txt,
# /run/lunafox/inputs/host-ports.jsonl, /run/lunafox/inputs/website-urls.txt,
# /run/lunafox/inputs/endpoint-urls.txt,
# /run/lunafox/resources/platform/fingerprintLibraryFingerPrintHub/fingerprinthub_web.json,
# and the writable /workspace.
USER 0:0
STOPSIGNAL SIGTERM
ENTRYPOINT ["/opt/lunafox-engine/bin/fingerprint-detection-runtime-engine"]
CMD []
