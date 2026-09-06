# syntax=docker/dockerfile:1.7

# lunafox:dockerfile-mode=custom

ARG ENGINE_GO_VERSION=1.26
ARG TOOLS_GO_VERSION=1.26
ARG UBUNTU_BASE=ubuntu:noble-20260113
ARG ENGINE_IMAGE_VERSION
ARG ENGINE_IMAGE_SOURCE=https://github.com/yyhuni/lunafox
ARG WAYMORE_VERSION=v8.9
ARG KATANA_VERSION=v1.6.1
ARG URO_VERSION=v1.0.2
ARG HTTPX_VERSION=v1.10.0
ARG GOPROXY=https://proxy.golang.org,direct

FROM --platform=$BUILDPLATFORM public.ecr.aws/docker/library/golang:${TOOLS_GO_VERSION}-bookworm AS go-tools
ARG TARGETOS=linux
ARG TARGETARCH
ARG KATANA_VERSION
ARG HTTPX_VERSION
ARG GOPROXY
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH GOTOOLCHAIN=local GOPROXY=$GOPROXY
WORKDIR /out
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    set -eux; \
    GOPATH=/go GOBIN= go install "github.com/projectdiscovery/katana/cmd/katana@${KATANA_VERSION}"; \
    GOPATH=/go GOBIN= go install "github.com/projectdiscovery/httpx/cmd/httpx@${HTTPX_VERSION}"; \
    install_tool() { \
      tool="$1"; \
      for candidate in "/go/bin/${TARGETOS}_${TARGETARCH}/${tool}" "/go/bin/${tool}"; do \
        if [ -x "$candidate" ]; then \
          install -m 0755 "$candidate" "/out/${tool}"; \
          return 0; \
        fi; \
      done; \
      echo "missing built tool: ${tool}" >&2; \
      return 1; \
    }; \
    install_tool katana; \
    install_tool httpx

FROM --platform=$TARGETPLATFORM ${UBUNTU_BASE} AS python-tools
COPY url_collection/requirements.lock /tmp/requirements.lock
# Uro's supported entrypoint is pip's generated console script; its module
# does not invoke main() when run with `python -m uro.uro`.
RUN apt-get update && apt-get install -y --no-install-recommends python3 python3-pip \
    && rm -rf /var/lib/apt/lists/* \
    && python3 -m pip install --break-system-packages --no-cache-dir --target /opt/lunafox-python --requirement /tmp/requirements.lock \
    && PYTHONPATH=/opt/lunafox-python python3 -m waymore.waymore --help >/dev/null \
    && PYTHONPATH=/opt/lunafox-python /opt/lunafox-python/bin/uro --help >/dev/null \
    && install -d -m 0755 /out \
    && printf '%s\n' '#!/bin/sh' 'exec env PYTHONPATH=/opt/lunafox-tools/python python3 -m waymore.waymore "$@"' > /out/waymore \
    && chmod 0755 /out/waymore

FROM --platform=$BUILDPLATFORM public.ecr.aws/docker/library/golang:${ENGINE_GO_VERSION}-bookworm AS engine-builder
ARG TARGETOS=linux
ARG TARGETARCH
ARG GOPROXY
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH GOTOOLCHAIN=local GOPROXY=$GOPROXY
WORKDIR /src/url_collection
# Copy the Engine's independent module before resolving its local engine-go
# replacement from the named context.
COPY --from=contracts . /contracts
COPY --from=engine-go . /engine-go
COPY url_collection/go.mod url_collection/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    go mod download
COPY url_collection ./
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    go build -trimpath -ldflags='-s -w' -o /out/url-collection-engine ./cmd/url-collection-engine

FROM --platform=$TARGETPLATFORM ${UBUNTU_BASE} AS runtime-base
ARG ENGINE_IMAGE_VERSION
ARG ENGINE_IMAGE_SOURCE
ARG WAYMORE_VERSION
ARG KATANA_VERSION
ARG URO_VERSION
ARG HTTPX_VERSION
RUN test -n "$ENGINE_IMAGE_VERSION" && test -n "$ENGINE_IMAGE_SOURCE"
LABEL org.opencontainers.image.source="${ENGINE_IMAGE_SOURCE}" \
      org.opencontainers.image.version="${ENGINE_IMAGE_VERSION}"
ENV PATH=/opt/lunafox-tools/bin:$PATH \
    PYTHONPATH=/opt/lunafox-tools/python
WORKDIR /workspace
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates python3 \
    && rm -rf /var/lib/apt/lists/* \
    && install -d -m 0755 /opt/lunafox-engine/bin /opt/lunafox-tools/bin \
      /run/lunafox/context /run/lunafox/credential /run/lunafox/inputs \
      /run/lunafox/resources/config /run/lunafox/resources/platform /run/lunafox/socket /workspace
COPY --from=engine-builder /out/url-collection-engine /opt/lunafox-engine/bin/url-collection-engine
COPY --from=go-tools /out/katana /opt/lunafox-tools/bin/katana
COPY --from=go-tools /out/httpx /opt/lunafox-tools/bin/httpx
COPY --from=python-tools /opt/lunafox-python /opt/lunafox-tools/python
COPY --from=python-tools /out/waymore /opt/lunafox-tools/bin/waymore
COPY --from=python-tools /opt/lunafox-python/bin/uro /opt/lunafox-tools/bin/uro
RUN set -eux; \
    printf '%s\n' "waymore=${WAYMORE_VERSION}" "katana=${KATANA_VERSION}" "uro=${URO_VERSION}" "httpx=${HTTPX_VERSION}" > /opt/lunafox-tools/VERSIONS; \
    chmod 0755 /opt/lunafox-engine/bin/url-collection-engine

FROM runtime-base AS verify
RUN --mount=type=bind,source=url_collection/tests/container,target=/run/lunafox-conformance \
    /bin/sh /run/lunafox-conformance/container-conformance.sh /opt/lunafox-engine/bin/url-collection-engine \
    && : > /conformance-passed

FROM runtime-base AS runtime
RUN --mount=type=bind,from=verify,source=/conformance-passed,target=/run/.conformance-passed \
    test -f /run/.conformance-passed

# Agent attaches fixed task-scoped ABI mounts without environment overrides.
# Canonical task mounts: /run/lunafox/context/execution.pb,
# /run/lunafox/socket/engine.sock, /run/lunafox/credential/token,
# /run/lunafox/inputs/subdomains.txt, /run/lunafox/inputs/host-ports.jsonl,
# /run/lunafox/inputs/website-urls.txt, and /run/lunafox/inputs/endpoint-urls.txt.
USER 0:0
STOPSIGNAL SIGTERM
ENTRYPOINT ["/opt/lunafox-engine/bin/url-collection-engine"]
CMD []
