/**
 * Subdomain validation utility class
 * 
 * Provides subdomain parsing and validation functionality for frontend validation during bulk subdomain addition.
 * Note: Whether subdomains match Target name is validated by backend.
 */

export interface SubdomainValidationResult {
  isValid: boolean
  subdomain: string
  error?: string
  errorCode?: SubdomainValidationErrorCode
  index: number
  lineNumber: number
}

export type SubdomainValidationErrorCode = 'empty' | 'too_long' | 'too_many_lines' | 'format'

export interface ParsedSubdomainLine {
  subdomain: string
  lineNumber: number
}

export interface SubdomainDuplicateResult extends SubdomainValidationResult {
  duplicateOfIndex: number
  duplicateOfLine: number
}

export interface ParseResult {
  subdomains: string[]
  validCount: number
  invalidCount: number
  duplicateCount: number
  invalidItems: SubdomainValidationResult[]
  duplicateItems: SubdomainDuplicateResult[]
}

// Subdomain format regex: allows letters, numbers, hyphens, separated by dots
const SUBDOMAIN_REGEX = /^(?!-)[a-zA-Z0-9-]{1,63}(?<!-)(\.[a-zA-Z0-9-]{1,63})*$/
// Backend BatchCreateSubdomainsRequest enforces max=5000 via struct tag.
const MAX_SUBDOMAIN_BATCH_LINES = 5000

export class SubdomainValidator {
  /**
   * Parse input text, only supports newline separation (one subdomain per line)
   */
  static parse(input: string): string[] {
    if (!input || typeof input !== 'string') {
      return []
    }
    
    return input
      .split('\n')
      .map(s => s.trim().toLowerCase())
      .filter(s => s.length > 0)
  }

  /**
   * Parse input text while preserving the original 1-based line number.
   */
  static parseLines(input: string): ParsedSubdomainLine[] {
    if (!input || typeof input !== 'string') {
      return []
    }

    return input
      .split('\n')
      .map((value, index) => ({
        subdomain: value.trim().toLowerCase(),
        lineNumber: index + 1,
      }))
      .filter(item => item.subdomain.length > 0)
  }

  /**
   * Validate single subdomain format (does not validate if it matches Target)
   */
  static validate(subdomain: string, index: number = 0, lineNumber: number = index + 1): SubdomainValidationResult {
    const trimmed = subdomain.trim().toLowerCase()
    
    // Empty value check
    if (!trimmed) {
      return {
        isValid: false,
        subdomain: subdomain,
        error: 'Subdomain cannot be empty',
        errorCode: 'empty',
        index,
        lineNumber,
      }
    }
    
    // Length check (DNS standard limit)
    if (trimmed.length > 253) {
      return {
        isValid: false,
        subdomain: trimmed,
        error: 'Subdomain length cannot exceed 253 characters',
        errorCode: 'too_long',
        index,
        lineNumber,
      }
    }
    
    // Format check
    if (!SUBDOMAIN_REGEX.test(trimmed)) {
      return {
        isValid: false,
        subdomain: trimmed,
        error: 'Invalid subdomain format',
        errorCode: 'format',
        index,
        lineNumber,
      }
    }
    
    return {
      isValid: true,
      subdomain: trimmed,
      index,
      lineNumber,
    }
  }

  /**
   * Batch validate and deduplicate
   */
  static validateBatch(subdomains: Array<string | ParsedSubdomainLine>): ParseResult {
    const seen = new Map<string, { index: number; lineNumber: number }>()
    const validSubdomains: string[] = []
    const invalidItems: SubdomainValidationResult[] = []
    const duplicateItems: SubdomainDuplicateResult[] = []
    let duplicateCount = 0
    
    subdomains.forEach((item, index) => {
      const input = typeof item === 'string'
        ? { subdomain: item, lineNumber: index + 1 }
        : item

      if (index >= MAX_SUBDOMAIN_BATCH_LINES) {
        invalidItems.push({
          isValid: false,
          subdomain: input.subdomain,
          error: `Subdomain list cannot exceed ${MAX_SUBDOMAIN_BATCH_LINES} lines`,
          errorCode: 'too_many_lines',
          index,
          lineNumber: input.lineNumber,
        })
        return
      }

      const result = this.validate(input.subdomain, index, input.lineNumber)
      
      if (!result.isValid) {
        invalidItems.push(result)
        return
      }
      
      // Duplicate check
      const duplicatedFrom = seen.get(result.subdomain)
      if (duplicatedFrom) {
        duplicateCount++
        duplicateItems.push({
          ...result,
          duplicateOfIndex: duplicatedFrom.index,
          duplicateOfLine: duplicatedFrom.lineNumber,
        })
        return
      }
      
      seen.set(result.subdomain, {
        index,
        lineNumber: result.lineNumber,
      })
      validSubdomains.push(result.subdomain)
    })
    
    return {
      subdomains: validSubdomains,
      validCount: validSubdomains.length,
      invalidCount: invalidItems.length,
      duplicateCount,
      invalidItems,
      duplicateItems,
    }
  }
}
