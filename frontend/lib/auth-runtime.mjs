import {
  AUTH_PRIMARY_TOKEN_KEY,
  AUTH_RENEWAL_TOKEN_KEY,
} from "./auth-storage-keys.mjs"

export const SMOKE_AUTH_SESSION_KEY = "__lunafox_smoke_auth__"
export const SMOKE_AUTH_SESSION_ENABLED = "enabled"
export const SMOKE_LOCALE_COOKIE_KEY = "NEXT_LOCALE"
export const PUBLIC_AUTH_PATHS = ["/login"]
export const DEFAULT_AUTH_RETURN_TO = "/overview/"

export function isPublicPathname(pathname) {
  return PUBLIC_AUTH_PATHS.some((route) => {
    return pathname === route || pathname.startsWith(`${route}/`)
  })
}

export function buildLoginPath() {
  return "/login/"
}

export function resolveSafeReturnTo(returnTo) {
  if (typeof returnTo !== "string") return null

  const trimmed = returnTo.trim()
  if (!trimmed.startsWith("/")) return null
  if (trimmed.startsWith("//")) return null
  return trimmed === "/" ? DEFAULT_AUTH_RETURN_TO : trimmed
}

export function buildReturnTo(pathname, search = "", hash = "") {
  const normalizedPathname = !pathname || pathname === "/" ? DEFAULT_AUTH_RETURN_TO : pathname
  const normalizedSearch =
    typeof search === "string" ? search.replace(/^\?/, "") : String(search ?? "")
  const normalizedHash =
    typeof hash === "string" ? hash.replace(/^#/, "") : String(hash ?? "")

  return `${normalizedPathname}${normalizedSearch ? `?${normalizedSearch}` : ""}${normalizedHash ? `#${normalizedHash}` : ""}`
}

export function buildLoginRedirectPath(returnTo) {
  const safeReturnTo = resolveSafeReturnTo(returnTo)
  if (!safeReturnTo) {
    return buildLoginPath()
  }

  const params = new URLSearchParams({ returnTo: safeReturnTo })
  return `${buildLoginPath()}?${params.toString()}`
}

export function buildLoginRedirectPathForLocation(pathname, search = "", hash = "") {
  if (isPublicPathname(pathname)) {
    return buildLoginPath()
  }

  return buildLoginRedirectPath(buildReturnTo(pathname, search, hash))
}

export function createSmokeLocaleCookie(baseUrl, locale, cookieName = SMOKE_LOCALE_COOKIE_KEY) {
  return {
    name: cookieName,
    value: locale,
    url: new URL(baseUrl).origin,
    sameSite: "Lax",
    secure: String(baseUrl).startsWith("https://"),
  }
}

export async function primeSmokeLocaleCookie(context, baseUrl, locale, cookieName = SMOKE_LOCALE_COOKIE_KEY) {
  if (!locale) return
  await context.addCookies([createSmokeLocaleCookie(baseUrl, locale, cookieName)])
}

export function createSmokeAuthBootstrap(overrides = {}) {
  return {
    primaryTokenKey: AUTH_PRIMARY_TOKEN_KEY,
    renewalTokenKey: AUTH_RENEWAL_TOKEN_KEY,
    sessionKey: SMOKE_AUTH_SESSION_KEY,
    sessionEnabledValue: SMOKE_AUTH_SESSION_ENABLED,
    primaryToken: "mock-access-token",
    renewalToken: "mock-refresh-token",
    ...overrides,
  }
}

export function primeSmokeAuthSession(storageOrBootstrap, bootstrap) {
  const hasExplicitStorage = typeof storageOrBootstrap?.setItem === "function"
  const storage = hasExplicitStorage ? storageOrBootstrap : globalThis.localStorage
  const session = hasExplicitStorage
    ? (bootstrap ?? createSmokeAuthBootstrap())
    : storageOrBootstrap
  storage.setItem(session.primaryTokenKey, session.primaryToken)
  storage.setItem(session.renewalTokenKey, session.renewalToken)
  storage.setItem(session.sessionKey, session.sessionEnabledValue)
}
