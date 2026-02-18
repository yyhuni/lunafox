# syntax=docker/dockerfile:1.7

# ============================================
# Go Server - build
# ============================================
FROM golang:1.26 AS builder

ARG GO111MODULE=on
ARG GOPROXY=https://proxy.golang.org,direct

ENV GO111MODULE=$GO111MODULE
ENV GOPROXY=$GOPROXY

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    go mod download

# Copy source
COPY . .

# Build (static where possible)
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    CGO_ENABLED=0 go build -o /out/server ./cmd/server

# ============================================
# Go Server - runtime
# ============================================
FROM debian:bookworm-20260112-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /out/server /usr/local/bin/server

EXPOSE 8080

CMD ["server"]
