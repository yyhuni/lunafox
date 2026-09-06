const defaultRouteParameterFixtures = Object.freeze({
  id: "1",
  websiteId: "1",
})

const optionalCatchAllPattern = /^\[\[\.\.\.([^/\[\]]+)\]\]$/
const catchAllPattern = /^\[\.\.\.([^/\[\]]+)\]$/
const parameterPattern = /^\[([^/\[\]]+)\]$/

function assertRoutePattern(routePattern) {
  if (typeof routePattern !== "string" || !routePattern.startsWith("/")) {
    throw new TypeError(`Route pattern must be an absolute path: ${String(routePattern)}`)
  }
}

function assertRouteParameters(params) {
  if (params === undefined) {
    return
  }
  if (!params || typeof params !== "object" || Array.isArray(params)) {
    throw new TypeError("Route parameters must be an object when provided")
  }
}

function defaultParameterValue(parameterName) {
  return defaultRouteParameterFixtures[parameterName] ?? "1"
}

function encodeRouteSegment(value, parameterName) {
  if (typeof value !== "string" && typeof value !== "number") {
    throw new TypeError(`Route parameter ${parameterName} must be a string or number`)
  }

  const normalized = String(value).trim()
  if (!normalized) {
    throw new TypeError(`Route parameter ${parameterName} must not be empty`)
  }

  return encodeURIComponent(normalized)
}

function resolveParameterValue(params, parameterName) {
  if (Object.hasOwn(params, parameterName)) {
    return params[parameterName]
  }
  return defaultParameterValue(parameterName)
}

function resolveCatchAllSegments(params, parameterName, { optional }) {
  const isExplicit = Object.hasOwn(params, parameterName)
  const value = isExplicit ? params[parameterName] : optional ? undefined : defaultParameterValue(parameterName)

  if (value === undefined || value === null) {
    if (optional) {
      return []
    }
    throw new TypeError(`Required catch-all route parameter ${parameterName} is missing`)
  }

  if (!Array.isArray(value) && typeof value !== "string" && typeof value !== "number") {
    throw new TypeError(`Catch-all route parameter ${parameterName} must be a string, number, or array`)
  }

  const segments = Array.isArray(value) ? value : String(value).split("/")
  if (segments.length === 0) {
    if (optional) {
      return []
    }
    throw new TypeError(`Required catch-all route parameter ${parameterName} must not be empty`)
  }

  return segments.map((segment) => encodeRouteSegment(segment, parameterName))
}

/**
 * Converts a Next.js route pattern into a deterministic smoke-test path.
 * Explicit parameters override fixtures; optional catch-all segments are omitted by default.
 */
export function instantiateRoutePattern(routePattern, params = {}) {
  assertRoutePattern(routePattern)
  assertRouteParameters(params)

  const path = routePattern
    .split("/")
    .flatMap((segment) => {
      const optionalCatchAll = segment.match(optionalCatchAllPattern)
      if (optionalCatchAll) {
        return resolveCatchAllSegments(params, optionalCatchAll[1], { optional: true })
      }

      const catchAll = segment.match(catchAllPattern)
      if (catchAll) {
        return resolveCatchAllSegments(params, catchAll[1], { optional: false })
      }

      const parameter = segment.match(parameterPattern)
      if (parameter) {
        return encodeRouteSegment(resolveParameterValue(params, parameter[1]), parameter[1])
      }

      if (segment.includes("[") || segment.includes("]")) {
        throw new TypeError(`Invalid route pattern segment: ${segment}`)
      }

      return segment
    })
    .join("/")

  return path || "/"
}
