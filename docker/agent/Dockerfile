# syntax=docker/dockerfile:1.7

# This wrapper accepts only a prepared binary context.
FROM debian:bookworm-20260112-slim

ARG TARGETARCH

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY . /agent-assets/

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
