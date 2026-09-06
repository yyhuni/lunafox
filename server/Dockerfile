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

# Cache module dependencies. The Server module uses the committed local
# contracts and engine-go replacements directly, so release builds must not
# depend on an untracked developer go.work file.
COPY contracts/go.mod contracts/go.sum ./contracts/
COPY engine-go/go.mod engine-go/go.sum ./engine-go/
COPY server/go.mod server/go.sum ./server/
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    cd server && go mod download

# Copy only Server and contract sources required by the release binary.
COPY contracts ./contracts
COPY engine-go ./engine-go
COPY server ./server
COPY extensions/workflows ./extensions/workflows

# Build (static where possible)
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    cd server && CGO_ENABLED=0 go build -o /out/server ./cmd/server

# ============================================
# Go Server - runtime
# ============================================
FROM debian:bookworm-20260112-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
	ffmpeg \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /out/server /usr/local/bin/server
COPY --from=builder /src/extensions/workflows /usr/local/share/lunafox/workflows

# Compose resolves the stable service name; this is the image-local operational
# command used by `docker compose exec server resetadmin`.
RUN ln -s /usr/local/bin/server /usr/local/bin/resetadmin

EXPOSE 8080

CMD ["server"]
