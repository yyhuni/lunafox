# Frontend Next.js Dockerfile
# Multi-stage build with BuildKit caching

# ==================== Dependencies stage ====================
FROM node:24.14.0-alpine AS deps
WORKDIR /app

# Install pnpm (pinned for reproducible builds)
RUN corepack enable && corepack prepare pnpm@10.30.3 --activate

# Copy dependency manifests
COPY package.json pnpm-lock.yaml ./

ARG NPM_CONFIG_REGISTRY=https://registry.npmmirror.com
ENV NPM_CONFIG_REGISTRY=${NPM_CONFIG_REGISTRY}

# Install dependencies (BuildKit cache)
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    pnpm install --frozen-lockfile

# ==================== Development stage ====================
FROM deps AS development
WORKDIR /app

# The source tree is updated by Compose Watch after the container starts.
COPY . ./

ENV NODE_ENV=development
ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

EXPOSE 3000

CMD ["pnpm", "dev"]

# ==================== Build stage ====================
FROM node:24.14.0-alpine AS builder
WORKDIR /app

RUN corepack enable && corepack prepare pnpm@10.30.3 --activate

# Copy deps
COPY --from=deps /app/node_modules ./node_modules
COPY . ./

# Build-time env
ARG IMAGE_TAG=unknown
ENV NEXT_PUBLIC_IMAGE_TAG=${IMAGE_TAG}
# Use service name "server" inside Docker network
ENV API_HOST=server

# Build (BuildKit cache)
RUN --mount=type=cache,target=/app/.next/cache \
    pnpm build

# ==================== Runtime stage ====================
FROM node:24.14.0-alpine AS runner
WORKDIR /app

ENV NODE_ENV=production

# Create non-root user
RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 nextjs

# Copy build output
COPY --from=builder /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

USER nextjs

EXPOSE 3000

ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

CMD ["node", "server.js"]
