export type ProductionRuntimeFlags = {
  nodeEnv?: string
  isVercel?: boolean
  demoSite?: boolean
  useMock?: boolean
  skipAuth?: boolean
}

/**
 * Reject public development switches before a production bundle can be emitted.
 * These flags are compiled into the browser, so a runtime warning would be too late.
 *
 * DEMO_SITE is the only production exception, and only on Vercel. It must enable
 * mock and skip-auth together so a showcase cannot skip authentication against
 * a real API or leave visitors at a login wall.
 */
export function assertProductionRuntimeFlags({
  nodeEnv = process.env.NODE_ENV,
  isVercel = process.env.VERCEL === "1",
  demoSite = process.env.DEMO_SITE === "true",
  useMock = process.env.NEXT_PUBLIC_USE_MOCK === "true",
  skipAuth = process.env.NEXT_PUBLIC_SKIP_AUTH === "true",
}: ProductionRuntimeFlags = {}): void {
  if (nodeEnv !== "production") return

  if (demoSite) {
    if (!isVercel) {
      throw new Error("DEMO_SITE is only valid for a Vercel demo deployment.")
    }
    if (!useMock) {
      throw new Error("Vercel demo builds must enable NEXT_PUBLIC_USE_MOCK.")
    }
    if (!skipAuth) {
      throw new Error("Vercel demo builds must enable NEXT_PUBLIC_SKIP_AUTH.")
    }
    return
  }

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
 * Keep the no-auth demo bundle from being pointed at a real API.
 * Mock mode owns /v1 in the browser, so a configured origin is a failed demo setup.
 */
export function assertVercelDemoHasNoBackend(
  backendUrl = process.env.BACKEND_URL,
  publicBackendUrl = process.env.NEXT_PUBLIC_BACKEND_URL,
): void {
  if (backendUrl?.trim() || publicBackendUrl?.trim()) {
    throw new Error(
      "Vercel demo builds must not configure BACKEND_URL or NEXT_PUBLIC_BACKEND_URL; mock mode owns /v1 requests.",
    )
  }
}

/**
 * Return the backend origin used by Vercel's same-origin `/v1` and `/mcp` rewrites.
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
