/**
 * Environment variables and runtime configuration utilities
 */

const DEFAULT_DEV_BACKEND_URL = 'http://localhost:8080'

const stripTrailingSlash = (url: string) => url.replace(/\/+$/, '')

/**
 * Get backend base URL (used to bypass Next.js proxy, ensuring SSE and other long connections work)
 */
export function getBackendBaseUrl(): string {
  const envUrl = process.env.NEXT_PUBLIC_BACKEND_URL?.trim()
  if (envUrl) {
    return stripTrailingSlash(envUrl)
  }

  if (typeof window !== 'undefined') {
    const origin = window.location.origin
    // In local development, backend runs on port 8080 by default
    if (window.location.hostname === 'localhost' && window.location.port === '3000') {
      return stripTrailingSlash(DEFAULT_DEV_BACKEND_URL)
    }
    return stripTrailingSlash(origin)
  }

  return stripTrailingSlash(DEFAULT_DEV_BACKEND_URL)
}

/**
 * Build backend API URL (automatically handles extra slashes)
 */
export function buildBackendUrl(path: string): string {
  const base = getBackendBaseUrl()
  const normalizedPath = path.startsWith('/') ? path : `/${path}`
  return `${base}${normalizedPath}`
}
