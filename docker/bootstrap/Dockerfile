# syntax=docker/dockerfile:1.7

# This Dockerfile is part of the public Runtime source closure. The resulting
# image is a one-shot bootstrap runtime that prepares the Compose-managed Agent identity.
FROM --platform=$BUILDPLATFORM golang:1.26 AS builder

ARG TARGETOS=linux
ARG TARGETARCH
ARG GO111MODULE=on
ARG GOPROXY=https://proxy.golang.org,direct
ENV GO111MODULE=$GO111MODULE \
    GOPROXY=$GOPROXY \
    CGO_ENABLED=0 \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH

WORKDIR /src
COPY contracts/go.mod contracts/go.sum ./contracts/
COPY engine-go/go.mod engine-go/go.sum ./engine-go/
COPY server/go.mod server/go.sum ./server/
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    cd server && go mod download

COPY contracts ./contracts
COPY engine-go ./engine-go
COPY server ./server
COPY extensions/workflows ./extensions/workflows
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    cd server && \
    go build -trimpath -buildvcs=false -o /out/server ./cmd/server && \
    go build -trimpath -buildvcs=false -o /out/lunafox-upgrader ./cmd/lunafox-upgrader

FROM alpine:3.23.5

RUN apk add --no-cache \
    bash \
    ca-certificates \
    docker-cli=29.5.2-r0 \
    docker-cli-compose=2.40.3-r6 \
    openssl

WORKDIR /app
COPY --from=builder /out/server /usr/local/bin/server
COPY --from=builder /out/lunafox-upgrader /usr/local/bin/lunafox-upgrader
COPY --from=builder /src/extensions/workflows /usr/local/share/lunafox/workflows
COPY docker/bootstrap/bootstrap.sh /usr/local/bin/lunafox-bootstrap
COPY docker/bootstrap/cert-init.sh /usr/local/bin/lunafox-cert-init
COPY docker/bootstrap/config-init.sh /usr/local/bin/lunafox-config-init
RUN chmod 0755 /usr/local/bin/lunafox-bootstrap /usr/local/bin/lunafox-cert-init /usr/local/bin/lunafox-config-init /usr/local/bin/lunafox-upgrader

# Compose selects explicit entrypoints for the long-running upgrader and the
# one-shot helpers. The default remains the business bootstrap command.
ENTRYPOINT ["/usr/local/bin/lunafox-bootstrap"]
