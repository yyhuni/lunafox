/**
 * URL validation utility class
 * 
 * Provides URL parsing and validation functionality for frontend validation when adding URLs in batch.
 * Supports three asset types: Endpoints, Websites, Directories.
 */

export type TargetType = 'domain' | 'ip' | 'cidr'

export interface URLValidationResult {
  isValid: boolean
  url: string
  error?: string
  index: number
  isMatched?: boolean  // Whether it matches the target (only valid when targetName is provided)
}

export interface ParseResult {
  urls: string[]
  validCount: number
  invalidCount: number
  duplicateCount: number
  mismatchedCount: number  // Number of URLs that don't match the target
  invalidItems: URLValidationResult[]
  mismatchedItems: URLValidationResult[]  // List of URLs that don't match the target
}

// Maximum URL length
const MAX_URL_LENGTH = 2000

// URL format regex: must start with http:// or https://
const URL_PROTOCOL_REGEX = /^https?:\/\//i

export class URLValidator {
  /**
   * Parse input text, only supports newline separation (one URL per line)
   */
  static parse(input: string): string[] {
    if (!input || typeof input !== 'string') {
      return []
    }
    
    return input
      .split('\n')
      .map(s => s.trim())
      .filter(s => s.length > 0)
  }

  /**
   * Check if URL hostname matches the target
   * 
   * Matching rules (simple frontend validation, for hints only, does not block submission):
   * - Domain type: hostname === targetName or hostname.endsWith('.'+targetName)
   * - IP type: hostname === targetName
   * - CIDR type: skip validation (frontend cannot determine if IP is within CIDR range)
   */
  static checkMatch(url: string, targetName: string, targetType: TargetType): boolean {
    // Skip frontend validation for CIDR type
    if (targetType === 'cidr') {
      return true
    }
    
    try {
      const parsed = new URL(url)
      const hostname = parsed.hostname.toLowerCase()
      const target = targetName.toLowerCase()
      
      if (targetType === 'domain') {
        // Domain type: hostname equals target or ends with .target
        return hostname === target || hostname.endsWith('.' + target)
      } else if (targetType === 'ip') {
        // IP type: hostname must exactly equal target
        return hostname === target
      }
      
      return true
    } catch {
      return false
    }
  }

  /**
   * Validate single URL format
   */
  static validate(url: string, index: number = 0, targetName?: string, targetType?: TargetType): URLValidationResult {
    const trimmed = url.trim()
    
    // Empty value check
    if (!trimmed) {
      return {
        isValid: false,
        url: url,
        error: 'URL cannot be empty',
        index,
      }
    }
    
    // Length check
    if (trimmed.length > MAX_URL_LENGTH) {
      return {
        isValid: false,
        url: trimmed,
        error: `URL length cannot exceed ${MAX_URL_LENGTH} characters`,
        index,
      }
    }
    
    // Protocol check
    if (!URL_PROTOCOL_REGEX.test(trimmed)) {
      return {
        isValid: false,
        url: trimmed,
        error: 'URL must start with http:// or https://',
        index,
      }
    }
    
    // Try to parse URL
    try {
      const parsed = new URL(trimmed)
      if (!parsed.hostname) {
        return {
          isValid: false,
          url: trimmed,
          error: 'URL must contain hostname',
          index,
        }
      }
      
      // Check if it matches the target
      let isMatched = true
      if (targetName && targetType) {
        isMatched = this.checkMatch(trimmed, targetName, targetType)
      }
      
      return {
        isValid: true,
        url: trimmed,
        index,
        isMatched,
      }
    } catch {
      return {
        isValid: false,
        url: trimmed,
        error: 'Invalid URL format',
        index,
      }
    }
  }

  /**
   * Batch validation with deduplication
   */
  static validateBatch(urls: string[], targetName?: string, targetType?: TargetType): ParseResult {
    const seen = new Set<string>()
    const validUrls: string[] = []
    const invalidItems: URLValidationResult[] = []
    const mismatchedItems: URLValidationResult[] = []
    let duplicateCount = 0
    
    urls.forEach((url, index) => {
      const result = this.validate(url, index, targetName, targetType)
      
      if (!result.isValid) {
        invalidItems.push(result)
        return
      }
      
      // Deduplication check
      if (seen.has(result.url)) {
        duplicateCount++
        return
      }
      
      seen.add(result.url)
      validUrls.push(result.url)
      
      // Record mismatched URLs (but still add to valid list)
      if (result.isMatched === false) {
        mismatchedItems.push(result)
      }
    })
    
    return {
      urls: validUrls,
      validCount: validUrls.length,
      invalidCount: invalidItems.length,
      duplicateCount,
      mismatchedCount: mismatchedItems.length,
      invalidItems,
      mismatchedItems,
    }
  }
}
