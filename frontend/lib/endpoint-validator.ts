import validator from 'validator'
import { isIP } from 'is-ip'

/**
 * Endpoint validation utility class
 * Provides strict URL format validation
 * Uses validator.js for reliable URL validation
 */

export interface EndpointValidationResult {
  isValid: boolean
  error?: string
  url?: URL
}

export class EndpointValidator {
  /**
   * Validate if Endpoint is a valid HTTP/HTTPS URL
   * @param urlString - URL string to validate
   * @returns Validation result
   */
  static validate(urlString: string): EndpointValidationResult {
    // 1. Check if empty
    if (!urlString || urlString.trim().length === 0) {
      return {
        isValid: false,
        error: 'Endpoint cannot be empty'
      }
    }

    const trimmedUrl = urlString.trim()

    // 2. Check if contains spaces
    if (trimmedUrl.includes(' ')) {
      return {
        isValid: false,
        error: 'Endpoint cannot contain spaces'
      }
    }

    // 3. Use validator.js for strict validation
    if (!validator.isURL(trimmedUrl, {
      protocols: ['http', 'https'],
      require_protocol: true,
      require_valid_protocol: true,
      require_host: true,
      allow_underscores: false,
      allow_trailing_dot: false,
      allow_protocol_relative_urls: false,
    })) {
      return {
        isValid: false,
        error: 'Invalid Endpoint format, must be a valid HTTP/HTTPS URL'
      }
    }

    // 4. Try to parse URL (double validation)
    let parsedUrl: URL
    try {
      parsedUrl = new URL(trimmedUrl)
    } catch {
      return {
        isValid: false,
        error: 'Invalid Endpoint format, cannot parse'
      }
    }

    // 5. Validate protocol
    if (parsedUrl.protocol !== 'http:' && parsedUrl.protocol !== 'https:') {
      return {
        isValid: false,
        error: 'Only HTTP and HTTPS protocols are supported'
      }
    }

    // 6. Validate hostname
    if (!parsedUrl.hostname || parsedUrl.hostname.length === 0) {
      return {
        isValid: false,
        error: 'Endpoint must contain a valid hostname'
      }
    }

    // 7. Check hostname format (domain or IP)
    if (!this.isValidHostname(parsedUrl.hostname)) {
      return {
        isValid: false,
        error: 'Invalid hostname format'
      }
    }

    // 8. Check port number (if any)
    if (parsedUrl.port && !this.isValidPort(parsedUrl.port)) {
      return {
        isValid: false,
        error: 'Invalid port number (must be 1-65535)'
      }
    }

    // 9. Check path (optional, but must be valid if present)
    if (parsedUrl.pathname && parsedUrl.pathname.includes('..')) {
      return {
        isValid: false,
        error: 'Endpoint path cannot contain ".."'
      }
    }

    // 10. Check for dangerous characters
    if (this.containsDangerousCharacters(trimmedUrl)) {
      return {
        isValid: false,
        error: 'Endpoint contains unsafe characters'
      }
    }

    return {
      isValid: true,
      url: parsedUrl
    }
  }

  /**
   * Batch validate Endpoint list
   * @param urls - Array of URL strings
   * @returns Array of validation results
   */
  static validateBatch(urls: string[]): Array<EndpointValidationResult & { index: number; originalUrl: string }> {
    return urls.map((url, index) => ({
      ...this.validate(url),
      index,
      originalUrl: url
    }))
  }

  /**
   * Validate if hostname is valid (domain or IP address)
   */
  private static isValidHostname(hostname: string): boolean {
    // 1) IP validation (supports IPv4/IPv6)
    if (isIP(hostname)) {
      return true
    }

    // 2) Domain validation (using validator's FQDN validation)
    return validator.isFQDN(hostname, {
      require_tld: true,
      allow_underscores: false,
      allow_trailing_dot: false,
      allow_numeric_tld: false,
      allow_wildcard: false,
    })
  }

  /**
   * Validate if port number is valid
   */
  private static isValidPort(port: string): boolean {
    const portNum = parseInt(port, 10)
    return !isNaN(portNum) && portNum >= 1 && portNum <= 65535
  }

  /**
   * Check if URL contains dangerous characters
   */
  private static containsDangerousCharacters(url: string): boolean {
    // Check for control characters
    const controlCharRegex = /[\x00-\x1F\x7F]/
    if (controlCharRegex.test(url)) {
      return true
    }

    // Check for JavaScript protocol
    if (url.toLowerCase().includes('javascript:')) {
      return true
    }

    // Check for data protocol
    if (url.toLowerCase().includes('data:')) {
      return true
    }

    return false
  }

  /**
   * Format Endpoint (normalize)
   */
  static normalize(urlString: string): string | null {
    const result = this.validate(urlString)
    if (!result.isValid || !result.url) {
      return null
    }

    // Return normalized URL
    return result.url.href
  }

  /**
   * Extract parts of Endpoint
   */
  static parse(urlString: string): {
    protocol: string
    hostname: string
    port: string
    pathname: string
    search: string
    hash: string
  } | null {
    const result = this.validate(urlString)
    if (!result.isValid || !result.url) {
      return null
    }

    return {
      protocol: result.url.protocol,
      hostname: result.url.hostname,
      port: result.url.port,
      pathname: result.url.pathname,
      search: result.url.search,
      hash: result.url.hash
    }
  }
}
