import validator from 'validator'
import { parse as parseDomain } from 'tldts'

/**
 * Domain validation utility class
 * Uses validator.js for reliable domain validation
 */

export interface DomainValidationResult {
  isValid: boolean
  error?: string
}

export class DomainValidator {
  /**
   * Validate domain format (e.g. example.com)
   * @param domain - Domain string to validate
   * @returns Validation result
   */
  static validateDomain(domain: string): DomainValidationResult {
    // 1. Check if empty
    if (!domain || domain.trim().length === 0) {
      return {
        isValid: false,
        error: 'Domain cannot be empty'
      }
    }

    const trimmedDomain = domain.trim()

    // 2. Check if contains spaces
    if (trimmedDomain.includes(' ')) {
      return {
        isValid: false,
        error: 'Domain cannot contain spaces'
      }
    }

    // 3. Check length (using validator package)
    if (!validator.isLength(trimmedDomain, { min: 1, max: 253 })) {
      return {
        isValid: false,
        error: 'Domain length cannot exceed 253 characters'
      }
    }

    // 4. Use tldts for domain semantic validation (priority)
    const info = parseDomain(trimmedDomain)
    if (!info.domain || info.isIp === true) {
      return {
        isValid: false,
        error: 'Invalid domain format'
      }
    }

    // 5. Use validator.js isFQDN as fallback to ensure strictness
    if (!validator.isFQDN(trimmedDomain, {
      require_tld: true,
      allow_underscores: false,
      allow_trailing_dot: false,
      allow_numeric_tld: false,
      allow_wildcard: false,
    })) {
      return {
        isValid: false,
        error: 'Invalid domain format'
      }
    }

    return { isValid: true }
  }

  /**
   * Validate subdomain format (e.g. www.example.com, api.test.org)
   * @param subdomain - Subdomain string to validate
   * @returns Validation result
   */
  static validateSubdomain(subdomain: string): DomainValidationResult {
    // First perform basic domain validation
    const basicValidation = this.validateDomain(subdomain)
    if (!basicValidation.isValid) {
      return basicValidation
    }

    // Subdomain must contain at least 3 parts (e.g. www.example.com)
    const labels = subdomain.trim().split('.')
    if (labels.length < 3) {
      return {
        isValid: false,
        error: 'Subdomain must contain at least 3 parts (e.g. www.example.com)'
      }
    }

    return {
      isValid: true
    }
  }

  /**
   * Batch validate domain list
   * @param domains - Array of domain strings
   * @returns Array of validation results
   */
  static validateDomainBatch(domains: string[]): Array<DomainValidationResult & { index: number; originalDomain: string }> {
    return domains.map((domain, index) => ({
      ...this.validateDomain(domain),
      index,
      originalDomain: domain
    }))
  }

  /**
   * Batch validate subdomain list
   * @param subdomains - Array of subdomain strings
   * @returns Array of validation results
   */
  static validateSubdomainBatch(subdomains: string[]): Array<DomainValidationResult & { index: number; originalDomain: string }> {
    return subdomains.map((subdomain, index) => ({
      ...this.validateSubdomain(subdomain),
      index,
      originalDomain: subdomain
    }))
  }

  /**
   * Normalize domain (convert to lowercase)
   */
  static normalize(domain: string): string | null {
    const result = this.validateDomain(domain)
    if (!result.isValid) {
      return null
    }
    return domain.trim().toLowerCase()
  }

  /**
   * Extract root domain from subdomain (using PSL - Public Suffix List)
   * @param subdomain - Subdomain (e.g. www.example.com, blog.github.io)
   * @returns Root domain (e.g. example.com, blog.github.io) or null
   * 
   * Examples:
   * - www.example.com → example.com
   * - api.test.example.com → example.com
   * - blog.github.io → blog.github.io (correctly handles public suffix)
   * - www.bbc.co.uk → bbc.co.uk (correctly handles multi-level TLD)
   */
  static extractRootDomain(subdomain: string): string | null {
    const trimmed = subdomain.trim().toLowerCase()
    if (!trimmed) return null

    // Use tldts to parse domain
    const parsed = parseDomain(trimmed)
    if (!parsed.domain) {
      return null
    }
    return parsed.domain
  }

  /**
   * Group subdomain list by root domain
   * @param subdomains - Subdomain list
   * @returns { grouped: Map<root domain, subdomain[]>, invalid: invalid subdomains[] }
   */
  static groupSubdomainsByRootDomain(subdomains: string[]): {
    grouped: Map<string, string[]>
    invalid: string[]
  } {
    const grouped = new Map<string, string[]>()
    const invalid: string[] = []
    
    for (const subdomain of subdomains) {
      const rootDomain = this.extractRootDomain(subdomain)
      
      if (!rootDomain) {
        invalid.push(subdomain)
        continue
      }
      
      if (!grouped.has(rootDomain)) {
        grouped.set(rootDomain, [])
      }
      
      grouped.get(rootDomain)!.push(subdomain)
    }
    
    return { grouped, invalid }
  }

  /**
   * Check if subdomain belongs to specified root domain
   * @param subdomain - Subdomain (e.g. www.example.com, api.example.com)
   * @param rootDomain - Root domain (e.g. example.com)
   * @returns Whether it belongs to that root domain
   * 
   * Examples:
   * - isSubdomainOf('www.example.com', 'example.com') → true
   * - isSubdomainOf('api.test.example.com', 'example.com') → true
   * - isSubdomainOf('www.test.com', 'example.com') → false
   */
  static isSubdomainOf(subdomain: string, rootDomain: string): boolean {
    const trimmedSubdomain = subdomain.trim().toLowerCase()
    const trimmedRootDomain = rootDomain.trim().toLowerCase()
    
    if (!trimmedSubdomain || !trimmedRootDomain) {
      return false
    }
    
    // Extract root domain from subdomain
    const extractedRoot = this.extractRootDomain(trimmedSubdomain)
    
    // Compare extracted root domain with target root domain
    return extractedRoot === trimmedRootDomain
  }
}
