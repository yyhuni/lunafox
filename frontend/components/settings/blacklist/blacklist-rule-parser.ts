export type BlacklistRuleKind = "domain" | "ipv4" | "cidr" | "invalid"

export type ParsedBlacklistRule = {
  line: number
  value: string
  kind: BlacklistRuleKind
  valid: boolean
}

function isIPv4(value: string): boolean {
  const parts = value.split(".")
  return parts.length === 4 && parts.every((part) => {
    if (!/^(0|[1-9]\d{0,2})$/.test(part)) return false
    const parsed = Number(part)
    return parsed >= 0 && parsed <= 255
  })
}

function isIPv4CIDR(value: string): boolean {
  const [address, prefix, ...extra] = value.split("/")
  if (extra.length > 0 || !address || !prefix || !isIPv4(address) || !/^\d{1,2}$/.test(prefix)) {
    return false
  }
  const parsedPrefix = Number(prefix)
  return parsedPrefix >= 0 && parsedPrefix <= 32
}

function isDomain(value: string): boolean {
  const normalized = value.endsWith(".") ? value.slice(0, -1) : value
  if (!normalized || normalized.length > 253 || normalized.includes(":")) return false
  const labels = normalized.split(".")
  return labels.length >= 2 && labels.every((label) => (
    /^[\p{L}\p{N}](?:[\p{L}\p{N}-]*[\p{L}\p{N}])?$/u.test(label)
  ))
}

export function parseBlacklistRule(value: string): BlacklistRuleKind {
  if (isIPv4CIDR(value)) return "cidr"
  if (isIPv4(value)) return "ipv4"
  if (value.startsWith("*.")) {
    return value.slice(2).includes("*") || !isDomain(value.slice(2)) ? "invalid" : "domain"
  }
  if (value.includes("*") || value.includes("/")) return "invalid"
  return isDomain(value) ? "domain" : "invalid"
}

export function parseBlacklistRules(text: string): ParsedBlacklistRule[] {
  return text
    .split("\n")
    .map((rawValue, index) => ({ line: index + 1, value: rawValue.trim() }))
    .filter((rule) => rule.value.length > 0 && !rule.value.startsWith("#"))
    .map((rule) => {
      const kind = parseBlacklistRule(rule.value)
      return { ...rule, kind, valid: kind !== "invalid" }
    })
}

export function submittedBlacklistPatterns(rules: ParsedBlacklistRule[]): string[] {
  return rules.map((rule) => rule.value)
}
