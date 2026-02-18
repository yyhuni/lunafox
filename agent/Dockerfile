# syntax=docker/dockerfile:1

# ============================================
# Go Agent - build
# ============================================
FROM golang:1.26 AS builder

ARG GO111MODULE=on
ARG GOPROXY=https://goproxy.cn,direct

ENV GO111MODULE=$GO111MODULE
ENV GOPROXY=$GOPROXY

WORKDIR /src

# Cache dependencies
COPY agent/go.mod agent/go.sum ./
RUN go mod download

# Copy source
COPY agent ./agent

WORKDIR /src/agent

# Build (static where possible)
RUN CGO_ENABLED=0 go build -o /out/agent ./cmd/agent

# ============================================
# Go Agent - runtime
# ============================================
FROM debian:bookworm-20260112-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /out/agent /usr/local/bin/agent

CMD ["agent"]
