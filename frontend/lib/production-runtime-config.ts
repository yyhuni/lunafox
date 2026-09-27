export type ProductionRuntimeFlags = {
  nodeEnv?: string
  useMock?: boolean
  skipAuth?: boolean
}

/**
 * Reject public development switches before a production bundle can be emitted.
 * These flags are compiled into the browser, so a runtime warning would be too late.
 */
export function assertProductionRuntimeFlags({
  nodeEnv = process.env.NODE_ENV,
  useMock = process.env.NEXT_PUBLIC_USE_MOCK === "true",
  skipAuth = process.env.NEXT_PUBLIC_SKIP_AUTH === "true",
}: ProductionRuntimeFlags = {}): void {
  if (nodeEnv !== "production") return

  if (useMock) {
    throw new Error(
      "Production frontend builds must not enable NEXT_PUBLIC_USE_MOCK; use an explicit development mock script instead.",
    )
  }

  if (skipAuth) {
    throw new Error(
      "Production frontend builds must not enable NEXT_PUBLIC_SKIP_AUTH; use an explicit development no-auth script instead.",
    )
  }
}

/**
 * Return the external API origin used by Vercel's same-origin `/v1` rewrite.
 * Keeping this server-side avoids expanding the browser's CORS/auth boundary.
 */
export function getRequiredVercelBackendOrigin(
  backendUrl = process.env.BACKEND_URL,
  publicBackendUrl = process.env.NEXT_PUBLIC_BACKEND_URL,
): string {
  if (publicBackendUrl?.trim()) {
    throw new Error(
      "Vercel frontend builds must not enable NEXT_PUBLIC_BACKEND_URL; use the server-side BACKEND_URL rewrite instead.",
    )
  }

  const configuredUrl = backendUrl?.trim()
  if (!configuredUrl) {
    throw new Error(
      "Vercel frontend builds require BACKEND_URL to route /v1 requests to the real backend.",
    )
  }

  let parsedUrl: URL
  try {
    parsedUrl = new URL(configuredUrl)
  } catch {
    throw new Error("Vercel BACKEND_URL must be an HTTPS origin without a path, query, or fragment.")
  }

  if (
    parsedUrl.protocol !== "https:"
    || parsedUrl.username
    || parsedUrl.password
    || parsedUrl.pathname !== "/"
    || parsedUrl.search
    || parsedUrl.hash
  ) {
    throw new Error("Vercel BACKEND_URL must be an HTTPS origin without a path, query, or fragment.")
  }

  return parsedUrl.origin
}
