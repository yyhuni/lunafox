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
  index: number
}

export interface ParseResult {
  subdomains: string[]
  validCount: number
  invalidCount: number
  duplicateCount: number
  invalidItems: SubdomainValidationResult[]
}

// Subdomain format regex: allows letters, numbers, hyphens, separated by dots
const SUBDOMAIN_REGEX = /^(?!-)[a-zA-Z0-9-]{1,63}(?<!-)(\.[a-zA-Z0-9-]{1,63})*$/

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
   * Validate single subdomain format (does not validate if it matches Target)
   */
  static validate(subdomain: string, index: number = 0): SubdomainValidationResult {
    const trimmed = subdomain.trim().toLowerCase()
    
    // Empty value check
    if (!trimmed) {
      return {
        isValid: false,
        subdomain: subdomain,
        error: 'Subdomain cannot be empty',
        index,
      }
    }
    
    // Length check (DNS standard limit)
    if (trimmed.length > 253) {
      return {
        isValid: false,
        subdomain: trimmed,
        error: 'Subdomain length cannot exceed 253 characters',
        index,
      }
    }
    
    // Format check
    if (!SUBDOMAIN_REGEX.test(trimmed)) {
      return {
        isValid: false,
        subdomain: trimmed,
        error: 'Invalid subdomain format',
        index,
      }
    }
    
    return {
      isValid: true,
      subdomain: trimmed,
      index,
    }
  }

  /**
   * Batch validate and deduplicate
   */
  static validateBatch(subdomains: string[]): ParseResult {
    const seen = new Set<string>()
    const validSubdomains: string[] = []
    const invalidItems: SubdomainValidationResult[] = []
    let duplicateCount = 0
    
    subdomains.forEach((subdomain, index) => {
      const result = this.validate(subdomain, index)
      
      if (!result.isValid) {
        invalidItems.push(result)
        return
      }
      
      // Duplicate check
      if (seen.has(result.subdomain)) {
        duplicateCount++
        return
      }
      
      seen.add(result.subdomain)
      validSubdomains.push(result.subdomain)
    })
    
    return {
      subdomains: validSubdomains,
      validCount: validSubdomains.length,
      invalidCount: invalidItems.length,
      duplicateCount,
      invalidItems,
    }
  }
}
