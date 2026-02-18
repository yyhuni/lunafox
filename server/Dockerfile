# syntax=docker/dockerfile:1

# ============================================
# Go Server - build
# ============================================
FROM golang:1.26 AS builder

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build (static where possible)
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server

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
