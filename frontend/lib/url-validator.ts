/**
 * Batch-add validation for observed asset URLs.
 *
 * URL text is evidence. This client-side guard applies only the shared
 * transport rules plus the derived Host/Target check needed by this form; it
 * never parses and rebuilds a URL value.
 */

export type TargetType = 'domain' | 'ip' | 'cidr'
export type URLValidationErrorCode = 'empty' | 'too_long' | 'too_many_lines' | 'protocol' | 'control' | 'authority' | 'encoding'

export interface ParsedURLLine {
  url: string
  lineNumber: number
}

export interface URLValidationResult {
  isValid: boolean
  url: string
  error?: string
  errorCode?: URLValidationErrorCode
  index: number
  lineNumber: number
  isMatched?: boolean  // Whether it matches the target (only valid when targetName is provided)
}

export interface URLDuplicateResult extends URLValidationResult {
  duplicateOfIndex: number
  duplicateOfLine: number
}

export interface ParseResult {
  urls: string[]
  validCount: number
  invalidCount: number
  duplicateCount: number
  mismatchedCount: number  // Number of URLs that don't match the target
  invalidItems: URLValidationResult[]
  duplicateItems: URLDuplicateResult[]
  mismatchedItems: URLValidationResult[]  // List of URLs that don't match the target
}

// This matches contracts/results.ObservedAssetURLMaxBytes.
const MAX_URL_LENGTH = 2000
// Backend batch URL endpoints enforce max=5000 via struct tag.
const MAX_URL_BATCH_LINES = 5000

// URL format regex: must start with http:// or https://
const URL_PROTOCOL_REGEX = /^https?:\/\//i
const textEncoder = new TextEncoder()

function hasUnpairedSurrogate(value: string) {
  for (let index = 0; index < value.length; index += 1) {
    const code = value.charCodeAt(index)
    if (code < 0xD800 || code > 0xDFFF) continue
    if (code <= 0xDBFF && index + 1 < value.length) {
      const next = value.charCodeAt(index + 1)
      if (next >= 0xDC00 && next <= 0xDFFF) {
        index += 1
        continue
      }
    }
    return true
  }
  return false
}

function normalizeDerivedHost(value: string): string | undefined {
  const normalized = value.toLowerCase().replace(/\.$/u, "")
  if (!normalized || /\s/u.test(normalized)) return undefined

  // Server accepts DNS names and IPv4 only for observed-URL ownership. Keep
  // this permissive for non-ASCII labels so UI validation does not reject an
  // IDNA-capable Server value before the authoritative boundary sees it.
  if (!normalized.includes(".")) return undefined
  return normalized
}

function deriveObservedAssetURLHost(raw: string): string | undefined {
  const scheme = raw.match(URL_PROTOCOL_REGEX)
  if (!scheme) return undefined

  const authorityStart = scheme[0].length
  let authorityEnd = raw.length
  for (let index = authorityStart; index < raw.length; index += 1) {
    if (raw[index] === "/" || raw[index] === "?" || raw[index] === "#") {
      authorityEnd = index
      break
    }
  }

  const authority = raw.slice(authorityStart, authorityEnd)
  if (!authority) return undefined

  const hostPort = authority.slice(authority.lastIndexOf("@") + 1)
  if (!hostPort || hostPort.trim() !== hostPort) return undefined
  if (hostPort.startsWith("[") || (hostPort.match(/:/g) ?? []).length > 1) return undefined

  const portSeparator = hostPort.indexOf(":")
  const host = portSeparator < 0 ? hostPort : hostPort.slice(0, portSeparator)
  return normalizeDerivedHost(host)
}

function parseIPv4(value: string): number | undefined {
  const parts = value.split(".")
  if (parts.length !== 4) return undefined

  let result = 0
  for (const part of parts) {
    if (!/^\d+$/u.test(part)) return undefined
    const octet = Number(part)
    if (!Number.isInteger(octet) || octet < 0 || octet > 255) return undefined
    result = (result * 256) + octet
  }
  return result >>> 0
}

function matchesCIDRTarget(host: string, targetName: string) {
  const [networkRaw, prefixRaw, ...extra] = targetName.trim().split("/")
  if (extra.length > 0 || prefixRaw === undefined) return false
  const hostValue = parseIPv4(host)
  const networkValue = parseIPv4(networkRaw ?? "")
  const prefix = Number(prefixRaw)
  if (hostValue === undefined || networkValue === undefined || !Number.isInteger(prefix) || prefix < 0 || prefix > 32) {
    return false
  }
  const mask = prefix === 0 ? 0 : (0xFFFFFFFF << (32 - prefix)) >>> 0
  return (hostValue & mask) === (networkValue & mask)
}

export class URLValidator {
  /**
   * Parse input text, only supports newline separation (one URL per line)
   */
  static parse(input: string): string[] {
    return this.parseLines(input).map((item) => item.url)
  }

  /**
   * Parse input text while preserving the original 1-based line number.
   */
  static parseLines(input: string): ParsedURLLine[] {
    if (!input || typeof input !== 'string') {
      return []
    }

    // CRLF is an input-record delimiter, not an embedded URL control. A bare
    // CR remains part of the line and is rejected by validate below.
    return input
      .split(/\r?\n/u)
      .map((value, index) => ({
        url: value,
        lineNumber: index + 1,
      }))
      .filter(item => item.url !== "")
  }

  /**
   * Check a derived authority Host against the active target without changing
   * the submitted URL string.
   */
  static checkMatch(url: string, targetName: string, targetType: TargetType): boolean {
    const host = deriveObservedAssetURLHost(url)
    if (!host) return false

    const target = targetName.trim().toLowerCase().replace(/\.$/u, "")
    if (!target) return false

    if (targetType === "domain") {
      return host === target || host.endsWith(`.${target}`)
    }
    if (targetType === "ip") {
      return host === target
    }
    return matchesCIDRTarget(host, target)
  }

  /**
   * Validate single URL format
   */
  static validate(url: string, index: number = 0, targetName?: string, targetType?: TargetType, lineNumber: number = index + 1): URLValidationResult {
    if (url === "") {
      return {
        isValid: false,
        url: url,
        error: 'URL cannot be empty',
        errorCode: 'empty',
        index,
        lineNumber,
      }
    }
    
    // Length check
    if (hasUnpairedSurrogate(url)) {
      return {
        isValid: false,
        url,
        error: "URL must be valid UTF-8",
        errorCode: "encoding",
        index,
        lineNumber,
      }
    }

    if (textEncoder.encode(url).byteLength > MAX_URL_LENGTH) {
      return {
        isValid: false,
        url,
        error: `URL length cannot exceed ${MAX_URL_LENGTH} UTF-8 bytes`,
        errorCode: 'too_long',
        index,
        lineNumber,
      }
    }
    
    // Protocol check
    if (!URL_PROTOCOL_REGEX.test(url)) {
      return {
        isValid: false,
        url,
        error: 'URL must start with http:// or https://',
        errorCode: 'protocol',
        index,
        lineNumber,
      }
    }
    
    if (/[\u0000\r\n]/u.test(url)) {
      return {
        isValid: false,
        url,
        error: "URL cannot contain NUL, CR, or LF characters",
        errorCode: "control",
        index,
        lineNumber,
      }
    }

    if (!deriveObservedAssetURLHost(url)) {
      return {
        isValid: false,
        url,
        error: "URL must contain an unambiguous domain or IPv4 authority host",
        errorCode: "authority",
        index,
        lineNumber,
      }
    }

    let isMatched = true
    if (targetName && targetType) {
      isMatched = this.checkMatch(url, targetName, targetType)
    }

    return {
      isValid: true,
      url,
      index,
      lineNumber,
      isMatched,
    }
  }

  /**
   * Batch validation with deduplication
   */
  static validateBatch(urls: Array<string | ParsedURLLine>, targetName?: string, targetType?: TargetType): ParseResult {
    const seen = new Map<string, { index: number; lineNumber: number }>()
    const validUrls: string[] = []
    const invalidItems: URLValidationResult[] = []
    const duplicateItems: URLDuplicateResult[] = []
    const mismatchedItems: URLValidationResult[] = []
    let duplicateCount = 0
    const itemsToValidate = urls.slice(0, MAX_URL_BATCH_LINES)
    
    itemsToValidate.forEach((item, index) => {
      const input = typeof item === 'string'
        ? { url: item, lineNumber: index + 1 }
        : item

      const result = this.validate(input.url, index, targetName, targetType, input.lineNumber)
      
      if (!result.isValid) {
        invalidItems.push(result)
        return
      }
      
      // Deduplication check
      const duplicatedFrom = seen.get(result.url)
      if (duplicatedFrom) {
        duplicateCount++
        duplicateItems.push({
          ...result,
          duplicateOfIndex: duplicatedFrom.index,
          duplicateOfLine: duplicatedFrom.lineNumber,
        })
        return
      }
      
      seen.set(result.url, {
        index,
        lineNumber: result.lineNumber,
      })
      validUrls.push(result.url)
      
      // Record mismatched URLs (but still add to valid list)
      if (result.isMatched === false) {
        mismatchedItems.push(result)
      }
    })

    urls.slice(MAX_URL_BATCH_LINES).forEach((item, offset) => {
      const index = MAX_URL_BATCH_LINES + offset
      const input = typeof item === 'string'
        ? { url: item, lineNumber: index + 1 }
        : item

      invalidItems.push({
        isValid: false,
        url: input.url,
        error: `URL list cannot exceed ${MAX_URL_BATCH_LINES} lines`,
        errorCode: 'too_many_lines',
        index,
        lineNumber: input.lineNumber,
      })
    })
    
    return {
      urls: validUrls,
      validCount: validUrls.length,
      invalidCount: invalidItems.length,
      duplicateCount,
      mismatchedCount: mismatchedItems.length,
      invalidItems,
      duplicateItems,
      mismatchedItems,
    }
  }
}
