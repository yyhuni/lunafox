# syntax=docker/dockerfile:1.7

# This wrapper accepts one immutable Agent bundle selected by its complete
# source input fingerprint. Metadata beside the executables is verified
# before the build and intentionally never becomes an OCI image input.
FROM debian:bookworm-20260112-slim

ARG TARGETARCH
ARG AGENT_ARTIFACT_ID

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY agent/bin/${AGENT_ARTIFACT_ID}/lunafox-agent-linux-amd64 /agent-assets/lunafox-agent-linux-amd64
COPY agent/bin/${AGENT_ARTIFACT_ID}/lunafox-agent-linux-arm64 /agent-assets/lunafox-agent-linux-arm64
COPY agent/bin/${AGENT_ARTIFACT_ID}/lunafox-engine-mount-preflight-linux-amd64 /agent-assets/lunafox-engine-mount-preflight-linux-amd64
COPY agent/bin/${AGENT_ARTIFACT_ID}/lunafox-engine-mount-preflight-linux-arm64 /agent-assets/lunafox-engine-mount-preflight-linux-arm64

RUN set -eux; \
    test -x /agent-assets/lunafox-agent-linux-amd64; \
    test -x /agent-assets/lunafox-agent-linux-arm64; \
    test -x /agent-assets/lunafox-engine-mount-preflight-linux-amd64; \
    test -x /agent-assets/lunafox-engine-mount-preflight-linux-arm64; \
    case "$TARGETARCH" in \
      amd64) suffix=amd64 ;; \
      arm64) suffix=arm64 ;; \
      *) echo "unsupported TARGETARCH: $TARGETARCH" >&2; exit 1 ;; \
    esac; \
    test -x "/agent-assets/lunafox-agent-linux-${suffix}"; \
    test -x "/agent-assets/lunafox-engine-mount-preflight-linux-${suffix}"; \
    install -m 0755 "/agent-assets/lunafox-agent-linux-${suffix}" /usr/local/bin/agent; \
    install -m 0755 "/agent-assets/lunafox-engine-mount-preflight-linux-${suffix}" /usr/local/bin/lunafox-engine-mount-preflight; \
    rm -rf /agent-assets

WORKDIR /app
CMD ["agent"]
