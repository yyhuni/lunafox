# syntax=docker/dockerfile:1.7

# This Dockerfile is part of the public Runtime source closure. The resulting
# image is a one-shot bootstrap runtime that starts the private Agent artifact.
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
    cd server && go build -trimpath -buildvcs=false -o /out/server ./cmd/server

FROM docker:28.5.2-cli

RUN apk add --no-cache \
	bash \
    ca-certificates \
    curl \
	jq

WORKDIR /app
COPY --from=builder /out/server /usr/local/bin/server
COPY --from=builder /src/extensions/workflows /usr/local/share/lunafox/workflows
COPY docker/bootstrap/bootstrap.sh /usr/local/bin/lunafox-bootstrap
RUN chmod 0755 /usr/local/bin/lunafox-bootstrap

# The image is intentionally one-shot; Compose controls when this command is
# created and waits for its zero exit status before starting the public edge.
ENTRYPOINT ["/usr/local/bin/lunafox-bootstrap"]
