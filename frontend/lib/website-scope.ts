import type { WebsiteAssetScope } from "@/types/website.types"

export type WebsiteScopeFilterField = "host" | "websiteUrl"

function quoteFilterValue(value: string) {
  return value.replace(/\\/g, "\\\\").replace(/\"/g, "\\\"")
}

function unquoteFilterValue(value: string) {
  return value.replace(/\\\"/g, "\"").replace(/\\\\/g, "\\")
}

function filterFieldMatches(filter: string, field: WebsiteScopeFilterField) {
  return Array.from(filter.matchAll(new RegExp(
    `(?:^|[^A-Za-z0-9_])${field}(==|=)"((?:\\\\.|[^"\\\\])*)"`,
    "g",
  )))
}

function extractLeadingScopeClause(
  filter: string | undefined,
  field: WebsiteScopeFilterField,
  options: { requireExact: boolean; rejectAnyAdditionalField: boolean },
) {
  const trimmed = filter?.trim()
  if (!trimmed) return undefined

  const matches = filterFieldMatches(trimmed, field)
  if (matches.length === 0) return undefined

  const leading = trimmed.match(new RegExp(
    `^${field}(==|=)"((?:\\\\.|[^"\\\\])*)"(?:\\s*&&\\s*(.+))?$`,
  ))
  if (!leading) {
    throw new TypeError(`${field} scope must be the leading AND clause`)
  }
  if (options.requireExact && leading[1] !== "==") {
    throw new TypeError(`${field} scope requires ==`)
  }
  if (matches.length !== 1 && options.rejectAnyAdditionalField) {
    throw new TypeError(`${field} scope must appear exactly once`)
  }

  const value = unquoteFilterValue(leading[2] ?? "")
  const remainder = leading[3]?.trim()
  if (
    options.rejectAnyAdditionalField &&
    remainder &&
    filterFieldMatches(remainder, field).length > 0
  ) {
    throw new TypeError(`${field} scope must appear exactly once`)
  }

  return { value, remainder }
}

type ObservedWebsiteScopeParts = {
  scheme: "http" | "https"
  hostname: string
  effectivePort: string
  path: string
}

// This is a read-time relation parser only. It inspects the authority and
// path delimiters needed for Website Scope, while leaving payload bytes and
// percent sequences untouched. It must not become URL admission or storage
// identity logic.
function parseObservedWebsiteScope(rawURL: string): ObservedWebsiteScopeParts {
  if (!rawURL || /[\u0000\r\n]/u.test(rawURL)) {
    throw new TypeError("Website URL must be an absolute HTTP(S) URL")
  }

  const schemeMatch = rawURL.match(/^(https?):\/\//iu)
  if (!schemeMatch) {
    throw new TypeError("Website URL must be an absolute HTTP(S) URL")
  }
  const scheme = schemeMatch[1].toLowerCase() as "http" | "https"
  const authorityStart = schemeMatch[0].length
  let authorityEnd = rawURL.length
  for (let index = authorityStart; index < rawURL.length; index += 1) {
    if (rawURL[index] === "/" || rawURL[index] === "?" || rawURL[index] === "#") {
      authorityEnd = index
      break
    }
  }

  const authority = rawURL.slice(authorityStart, authorityEnd)
  if (!authority || authority.includes("@") || authority.startsWith("[") || (authority.match(/:/g) ?? []).length > 1) {
    throw new TypeError("Website URL cannot form a relation scope")
  }
  const separator = authority.indexOf(":")
  const hostname = (separator < 0 ? authority : authority.slice(0, separator)).toLowerCase()
  if (!hostname || /\s/u.test(hostname)) {
    throw new TypeError("Website URL cannot form a relation scope")
  }
  const explicitPort = separator < 0 ? "" : authority.slice(separator + 1)
  if (separator >= 0 && (!explicitPort || explicitPort.trim() !== explicitPort)) {
    throw new TypeError("Website URL cannot form a relation scope")
  }

  let pathEnd = rawURL.length
  for (let index = authorityEnd; index < rawURL.length; index += 1) {
    if (rawURL[index] === "?" || rawURL[index] === "#") {
      pathEnd = index
      break
    }
  }
  let path = authorityEnd < pathEnd && rawURL[authorityEnd] === "/"
    ? rawURL.slice(authorityEnd, pathEnd)
    : "/"
  if (path !== "/") path = path.replace(/\/+$/u, "") || "/"

  return {
    scheme,
    hostname,
    effectivePort: explicitPort || (scheme === "http" ? "80" : "443"),
    path,
  }
}

/**
 * Builds the mandatory Website-detail filter before any user supplied filter.
 * The range stays server enforced; callers only use this helper to preserve it
 * through search, facets, ordering, and page-token changes.
 */
export function composeWebsiteScopeFilter(
  scope: WebsiteAssetScope,
  field: WebsiteScopeFilterField,
  userFilter?: string
) {
  const value = field === "host"
    ? scope.host.trim()
    : scope.url

  if (!value) {
    throw new TypeError(`Website ${field} scope is required`)
  }
  if (field === "websiteUrl") {
    // Validate only the derived relation boundary; the quoted value remains
    // the exact stored URL, including payload and trailing bytes.
    parseObservedWebsiteScope(value)
  }

  const mandatoryFilter = `${field}=="${quoteFilterValue(value)}"`
  const trimmedUserFilter = userFilter?.trim()
  return trimmedUserFilter ? `${mandatoryFilter} && (${trimmedUserFilter})` : mandatoryFilter
}

/** Returns true only when the candidate shares the Website origin and path boundary. */
export function matchesWebsiteURLScope(scopeURL: string, candidateURL: string) {
  const scope = parseObservedWebsiteScope(scopeURL)
  let candidate: ObservedWebsiteScopeParts
  try {
    candidate = parseObservedWebsiteScope(candidateURL)
  } catch {
    return false
  }

  if (
    scope.scheme !== candidate.scheme ||
    scope.hostname !== candidate.hostname ||
    scope.effectivePort !== candidate.effectivePort
  ) {
    return false
  }

  const scopePath = scope.path
  if (scopePath === "/") return true

  const candidatePath = candidate.path
  return candidatePath === scopePath || candidatePath.startsWith(`${scopePath}/`)
}

/** Extracts the only permitted Website URL filter used by the mock transport. */
export function extractWebsiteURLScopeFilter(filter?: string) {
  const scope = extractLeadingScopeClause(filter, "websiteUrl", {
    requireExact: true,
    rejectAnyAdditionalField: true,
  })
  if (!scope) return undefined

  const scopeURL = scope.value
  parseObservedWebsiteScope(scopeURL)
  return scopeURL
}

/** Extracts the exact host scope emitted for Website-detail IP observations. */
export function extractExactWebsiteHostFilter(filter?: string) {
  const exactMatches = filterFieldMatches(filter?.trim() ?? "", "host")
    .filter((match) => match[1] === "==")
  if (exactMatches.length === 0) return undefined
  if (exactMatches.length !== 1) {
    throw new TypeError("host scope must appear exactly once with ==")
  }

  const scope = extractLeadingScopeClause(filter, "host", {
    requireExact: true,
    rejectAnyAdditionalField: false,
  })
  if (!scope) return undefined

  const host = scope.value.trim()
  if (!host) {
    throw new TypeError("host scope is required")
  }
  return host
}

/** Removes only the mandatory leading scope clause before mock user-filter evaluation. */
export function removeLeadingWebsiteScopeFilter(
  filter: string | undefined,
  field: WebsiteScopeFilterField,
) {
  if (!filter) return undefined

  const prefix = new RegExp(
    `^\\s*${field}=="(?:\\\\.|[^"\\\\])*"\\s*&&\\s*`,
  )
  return filter.replace(prefix, "").trim() || undefined
}

export function matchesExactWebsiteHost(scopeHost: string, candidateHost: string) {
  return scopeHost.trim().toLowerCase() === candidateHost.trim().toLowerCase()
}
