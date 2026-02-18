export interface FilterField {
  key: string
  label: string
  description: string
}

export interface ParsedFilter {
  field: string
  operator: string
  value: string
  raw: string
}

// Predefined field configurations
export const PREDEFINED_FIELDS: Record<string, FilterField> = {
  ip: { key: "ip", label: "IP", description: "IP address" },
  port: { key: "port", label: "Port", description: "Port number" },
  host: { key: "host", label: "Host", description: "Hostname" },
  domain: { key: "domain", label: "Domain", description: "Domain name" },
  url: { key: "url", label: "URL", description: "Full URL" },
  status: { key: "status", label: "Status", description: "HTTP status code" },
  title: { key: "title", label: "Title", description: "Page title" },
  source: { key: "source", label: "Source", description: "Data source" },
  path: { key: "path", label: "Path", description: "URL path" },
  severity: { key: "severity", label: "Severity", description: "Vulnerability severity" },
  name: { key: "name", label: "Name", description: "Name" },
  type: { key: "type", label: "Type", description: "Type" },
}

export function getTranslatedFields(
  t: (key: string) => string
): Record<string, FilterField> {
  return {
    ip: { key: "ip", label: "IP", description: t("fields.ip") },
    port: { key: "port", label: "Port", description: t("fields.port") },
    host: { key: "host", label: "Host", description: t("fields.host") },
    domain: { key: "domain", label: "Domain", description: t("fields.domain") },
    url: { key: "url", label: "URL", description: t("fields.url") },
    status: { key: "status", label: "Status", description: t("fields.status") },
    title: { key: "title", label: "Title", description: t("fields.title") },
    source: { key: "source", label: "Source", description: t("fields.source") },
    path: { key: "path", label: "Path", description: t("fields.path") },
    severity: { key: "severity", label: "Severity", description: t("fields.severity") },
    name: { key: "name", label: "Name", description: t("fields.name") },
    type: { key: "type", label: "Type", description: t("fields.type") },
  }
}

const FILTER_HISTORY_KEY = "smart_filter_history"
const MAX_HISTORY_PER_FIELD = 10
const FILTER_EXPRESSION_REGEX = /(\w+)(==|!=|=)"([^"]+)"/g

export function getFieldHistory(field: string): string[] {
  if (typeof window === "undefined") return []
  try {
    const history = JSON.parse(localStorage.getItem(FILTER_HISTORY_KEY) || "{}")
    return history[field] || []
  } catch {
    return []
  }
}

export function saveFieldHistory(field: string, value: string) {
  if (typeof window === "undefined" || !value.trim()) return
  try {
    const history = JSON.parse(localStorage.getItem(FILTER_HISTORY_KEY) || "{}")
    const fieldHistory = (history[field] || []).filter((v: string) => v !== value)
    fieldHistory.unshift(value)
    history[field] = fieldHistory.slice(0, MAX_HISTORY_PER_FIELD)
    localStorage.setItem(FILTER_HISTORY_KEY, JSON.stringify(history))
  } catch {
    // ignore
  }
}

export function saveQueryHistory(query: string) {
  let match: RegExpExecArray | null
  while ((match = FILTER_EXPRESSION_REGEX.exec(query)) !== null) {
    const [, field, , value] = match
    saveFieldHistory(field, value)
  }
}

export function parseFilterExpression(input: string): ParsedFilter[] {
  const filters: ParsedFilter[] = []
  let match: RegExpExecArray | null

  while ((match = FILTER_EXPRESSION_REGEX.exec(input)) !== null) {
    const [raw, field, operator, value] = match
    filters.push({ field, operator, value, raw })
  }

  return filters
}

export function getGhostText(inputValue: string, fields: FilterField[]): string {
  if (!inputValue) return ""

  const lastSpaceIndex = inputValue.lastIndexOf(" ")
  const currentToken =
    lastSpaceIndex === -1 ? inputValue : inputValue.slice(lastSpaceIndex + 1)
  const lowerToken = currentToken.toLowerCase()

  if (!currentToken && inputValue.trim()) {
    if (inputValue.trimEnd().endsWith('"')) {
      return "&& "
    }
    return ""
  }

  if (!currentToken) return ""

  if (!currentToken.includes("=") && !currentToken.includes("!")) {
    const matchingField = fields.find(
      (field) =>
        field.key.toLowerCase().startsWith(lowerToken) &&
        field.key.toLowerCase() !== lowerToken
    )
    if (matchingField) {
      return matchingField.key.slice(currentToken.length) + '="'
    }

    const exactField = fields.find((field) => field.key.toLowerCase() === lowerToken)
    if (exactField) {
      return '="'
    }

    if ("&&".startsWith(currentToken) && currentToken.startsWith("&")) {
      return "&&".slice(currentToken.length) + " "
    }
    if ("||".startsWith(currentToken) && currentToken.startsWith("|")) {
      return "||".slice(currentToken.length) + " "
    }
    if (!matchingField) {
      if (
        "and".startsWith(lowerToken) &&
        lowerToken.length > 0 &&
        !fields.some((field) => field.key.toLowerCase().startsWith(lowerToken))
      ) {
        return "and".slice(lowerToken.length) + " "
      }
      if (
        "or".startsWith(lowerToken) &&
        lowerToken.length > 0 &&
        !fields.some((field) => field.key.toLowerCase().startsWith(lowerToken))
      ) {
        return "or".slice(lowerToken.length) + " "
      }
    }

    return ""
  }

  if (currentToken.match(/^(\w+)!$/)) {
    return '="'
  }

  const singleEqMatch = currentToken.match(/^(\w+)=$/)
  if (singleEqMatch) {
    return '"'
  }

  const doubleOpMatch = currentToken.match(/^(\w+)(==|!=)$/)
  if (doubleOpMatch) {
    return '"'
  }

  const eqMatch = currentToken.match(/^(\w+)(==|!=|=)"([^"]*)$/)
  if (eqMatch) {
    const [, field, , partialValue] = eqMatch
    const history = getFieldHistory(field)
    const matchingValue = history.find(
      (value) =>
        value.toLowerCase().startsWith(partialValue.toLowerCase()) &&
        value.toLowerCase() !== partialValue.toLowerCase()
    )
    if (matchingValue) {
      return matchingValue.slice(partialValue.length) + '"'
    }
    if (partialValue.length > 0) {
      return '"'
    }
  }

  if (currentToken.match(/^\w+(==|!=|=)"[^"]+"$/)) {
    return " && "
  }

  return ""
}

export function buildDefaultPlaceholder(
  fields: FilterField[],
  examples?: string[]
): string {
  if (examples && examples.length > 0) {
    return examples[0]
  }
  return fields
    .slice(0, 2)
    .map((field) => `${field.key}="..."`)
    .join(" && ")
}
