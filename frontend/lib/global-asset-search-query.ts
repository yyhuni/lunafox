import type { AssetType } from "@/types/search.types"

export const GLOBAL_ASSET_SEARCH_DEFAULT_PAGE_SIZE = 10
export const GLOBAL_ASSET_SEARCH_MAX_PAGE_SIZE = 100
export const GLOBAL_ASSET_SEARCH_MAX_QUERY_BYTES = 2048
export const GLOBAL_ASSET_SEARCH_MAX_CONDITIONS = 10
export const GLOBAL_ASSET_SEARCH_MIN_CONTAINS_RUNES = 3

export type GlobalAssetSearchField = "url" | "host" | "title" | "statusCode" | "tech"
export type GlobalAssetSearchOperator = "=" | "=="

export const GLOBAL_ASSET_SEARCH_FIELDS = ["url", "host", "title", "statusCode", "tech"] as const satisfies readonly GlobalAssetSearchField[]

export interface GlobalAssetSearchCondition {
  field: GlobalAssetSearchField
  operator: GlobalAssetSearchOperator
  value: string | number
}

export type GlobalAssetSearchQuery =
  | { mode: "plainUrl"; value: string }
  | { mode: "structured"; conditions: GlobalAssetSearchCondition[] }

export class GlobalAssetSearchQueryError extends Error {
  constructor(message: string) {
    super(message)
    this.name = "GlobalAssetSearchQueryError"
  }
}

const textEncoder = new TextEncoder()
const allowedFields = new Set<GlobalAssetSearchField>(GLOBAL_ASSET_SEARCH_FIELDS)

export function parseGlobalAssetSearchQuery(raw: string): GlobalAssetSearchQuery {
  if (textEncoder.encode(raw).byteLength > GLOBAL_ASSET_SEARCH_MAX_QUERY_BYTES) {
    throw new GlobalAssetSearchQueryError(`q must not exceed ${GLOBAL_ASSET_SEARCH_MAX_QUERY_BYTES} UTF-8 bytes`)
  }

  // A plain query is an observed URL identity. Its bytes must reach Server
  // unchanged so an accepted URL with significant trailing payload bytes can
  // still be retrieved exactly. Structured syntax keeps its own whitespace
  // grammar below.
  const query = raw
  if (!query.trim()) {
    throw new GlobalAssetSearchQueryError("q is required")
  }

  if (isPlainObservedURL(query) || !looksStructured(query)) {
    assertContainsLength(query, "url")
    return { mode: "plainUrl", value: query }
  }

  return { mode: "structured", conditions: new StrictQueryParser(query).parse() }
}

export function isGlobalAssetSearchQueryValid(raw: string): boolean {
  try {
    parseGlobalAssetSearchQuery(raw)
    return true
  } catch {
    return false
  }
}

export function normalizeGlobalAssetSearchPageSize(value: number | undefined): number {
  if (value === undefined) {
    return GLOBAL_ASSET_SEARCH_DEFAULT_PAGE_SIZE
  }
  if (!Number.isInteger(value) || value < 1 || value > GLOBAL_ASSET_SEARCH_MAX_PAGE_SIZE) {
    throw new GlobalAssetSearchQueryError(`pageSize must be between 1 and ${GLOBAL_ASSET_SEARCH_MAX_PAGE_SIZE}`)
  }
  return value
}

export function globalAssetSearchQueryFingerprint(query: GlobalAssetSearchQuery, assetType: AssetType, pageSize: number): string {
  const canonical = query.mode === "plainUrl"
    ? `plain:url=${JSON.stringify(query.value)}`
    : `structured:${query.conditions
      .map((condition) => `${condition.field}${condition.operator}${JSON.stringify(String(condition.value))}`)
      .sort()
      .join("&&")}`
  return `${assetType}:${pageSize}:${canonical}`
}

function looksStructured(query: string): boolean {
  if (query.includes("&&") || query.includes("||") || query.includes("!=") || /[()]/.test(query)) {
    return true
  }
  return /(?:^|\s)[A-Za-z][A-Za-z0-9_]*\s*=/.test(query)
}

function isPlainObservedURL(query: string): boolean {
  return /^https?:\/\//iu.test(query)
}

function assertContainsLength(value: string, field: string) {
  if (Array.from(value).length < GLOBAL_ASSET_SEARCH_MIN_CONTAINS_RUNES) {
    throw new GlobalAssetSearchQueryError(`${field} contains values must have at least ${GLOBAL_ASSET_SEARCH_MIN_CONTAINS_RUNES} Unicode characters`)
  }
}

class StrictQueryParser {
  private position = 0

  constructor(private readonly input: string) {}

  parse(): GlobalAssetSearchCondition[] {
    const conditions: GlobalAssetSearchCondition[] = []
    this.skipWhitespace()

    while (!this.atEnd()) {
      conditions.push(this.parseCondition())
      if (conditions.length > GLOBAL_ASSET_SEARCH_MAX_CONDITIONS) {
        throw new GlobalAssetSearchQueryError(`at most ${GLOBAL_ASSET_SEARCH_MAX_CONDITIONS} conditions are allowed`)
      }
      this.skipWhitespace()
      if (this.atEnd()) break
      if (!this.consume("&&")) {
        throw new GlobalAssetSearchQueryError(`expected && at character ${this.position}`)
      }
      this.skipWhitespace()
      if (this.atEnd()) {
        throw new GlobalAssetSearchQueryError("missing condition after &&")
      }
    }

    if (conditions.length === 0) {
      throw new GlobalAssetSearchQueryError("q is required")
    }
    return conditions
  }

  private parseCondition(): GlobalAssetSearchCondition {
    const field = this.parseField()
    this.skipWhitespace()
    const operator = this.parseOperator()
    this.skipWhitespace()
    const rawValue = this.parseQuotedValue()

    if (field === "statusCode") {
      if (!/^\d+$/.test(rawValue)) {
        throw new GlobalAssetSearchQueryError("statusCode must be an integer")
      }
      const value = Number(rawValue)
      if (!Number.isSafeInteger(value)) {
        throw new GlobalAssetSearchQueryError("statusCode must be an integer")
      }
      return { field, operator, value }
    }

    if (field === "url") {
      // URL equality is exact even when callers use the legacy `=` spelling.
      // This mirrors the Server parser and prevents a UI-only contains branch.
      return { field, operator: "==", value: rawValue }
    }

    if ((field === "host" || field === "title") && operator === "=") {
      assertContainsLength(rawValue, field)
    }
    return { field, operator, value: rawValue }
  }

  private parseField(): GlobalAssetSearchField {
    if (this.atEnd() || !isIdentifierStart(this.input[this.position]!)) {
      throw new GlobalAssetSearchQueryError(`expected a field name at character ${this.position}`)
    }
    const start = this.position
    this.position += 1
    while (!this.atEnd() && isIdentifierPart(this.input[this.position]!)) {
      this.position += 1
    }
    const field = this.input.slice(start, this.position) as GlobalAssetSearchField
    if (!allowedFields.has(field)) {
      throw new GlobalAssetSearchQueryError(`unsupported field ${field}`)
    }
    return field
  }

  private parseOperator(): GlobalAssetSearchOperator {
    if (this.consume("==")) return "=="
    if (this.consume("=")) return "="
    throw new GlobalAssetSearchQueryError(`expected = or == at character ${this.position}`)
  }

  private parseQuotedValue(): string {
    if (this.atEnd() || this.input[this.position] !== "\"") {
      throw new GlobalAssetSearchQueryError(`expected a quoted value at character ${this.position}`)
    }
    const start = this.position
    this.position += 1
    let escaped = false
    while (!this.atEnd()) {
      const character = this.input[this.position]!
      this.position += 1
      if (escaped) {
        escaped = false
        continue
      }
      if (character === "\\") {
        escaped = true
        continue
      }
      if (character === "\"") {
        try {
          return JSON.parse(this.input.slice(start, this.position)) as string
        } catch {
          throw new GlobalAssetSearchQueryError("invalid quoted value")
        }
      }
    }
    throw new GlobalAssetSearchQueryError("unterminated quoted value")
  }

  private skipWhitespace() {
    while (!this.atEnd() && /\s/.test(this.input[this.position]!)) {
      this.position += 1
    }
  }

  private consume(value: string): boolean {
    if (!this.input.startsWith(value, this.position)) return false
    this.position += value.length
    return true
  }

  private atEnd(): boolean {
    return this.position >= this.input.length
  }
}

function isIdentifierStart(value: string): boolean {
  return /^[A-Za-z]$/.test(value)
}

function isIdentifierPart(value: string): boolean {
  return /^[A-Za-z0-9_]$/.test(value)
}
