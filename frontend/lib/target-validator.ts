import validator from 'validator'
import { parse as parseDomain } from 'tldts'

/**
 * Target validation utility class
 * Supports validation of four target types: domain, IP, CIDR, URL
 */

export type InputType = 'url' | 'domain' | 'ip' | 'cidr'

export interface TargetValidationResult {
  isValid: boolean
  error?: string
  type?: InputType
  isEmptyLine?: boolean  // Mark empty lines, frontend filters them out and doesn't send
}

export class TargetValidator {
  /**
   * Validate domain format (e.g. example.com)
   */
  static validateDomain(domain: string): TargetValidationResult {
    if (!domain || domain.trim().length === 0) {
      return {
        isValid: false,
        error: 'Target cannot be empty'
      }
    }

    const trimmedDomain = domain.trim()

    if (trimmedDomain.includes(' ')) {
      return {
        isValid: false,
        error: 'Target cannot contain spaces'
      }
    }

    if (!validator.isLength(trimmedDomain, { min: 1, max: 253 })) {
      return {
        isValid: false,
        error: 'Target length cannot exceed 253 characters'
      }
    }

    const info = parseDomain(trimmedDomain)
    if (!info.domain || info.isIp === true) {
      return {
        isValid: false,
        error: 'Invalid domain format'
      }
    }

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

    return { isValid: true, type: 'domain' }
  }

  /**
   * Validate IPv4 address (e.g. 192.168.1.1)
   */
  static validateIPv4(ip: string): TargetValidationResult {
    if (!ip || ip.trim().length === 0) {
      return {
        isValid: false,
        error: 'Target cannot be empty'
      }
    }

    const trimmedIP = ip.trim()

    if (!validator.isIP(trimmedIP, 4)) {
      return {
        isValid: false,
        error: 'Invalid IPv4 address format'
      }
    }

    return { isValid: true, type: 'ip' }
  }

  /**
   * Validate IPv6 address (e.g. 2001:db8::1)
   */
  static validateIPv6(ip: string): TargetValidationResult {
    if (!ip || ip.trim().length === 0) {
      return {
        isValid: false,
        error: 'Target cannot be empty'
      }
    }

    const trimmedIP = ip.trim()

    if (!validator.isIP(trimmedIP, 6)) {
      return {
        isValid: false,
        error: 'Invalid IPv6 address format'
      }
    }

    return { isValid: true, type: 'ip' }
  }

  /**
   * Validate IP address (IPv4 or IPv6)
   */
  static validateIP(ip: string): TargetValidationResult {
    if (!ip || ip.trim().length === 0) {
      return {
        isValid: false,
        error: 'Target cannot be empty'
      }
    }

    const trimmedIP = ip.trim()

    if (!validator.isIP(trimmedIP)) {
      return {
        isValid: false,
        error: 'Invalid IP address format'
      }
    }

    return { isValid: true, type: 'ip' }
  }

  /**
   * Validate CIDR network segment (e.g. 10.0.0.0/8, 192.168.0.0/16)
   */
  static validateCIDR(cidr: string): TargetValidationResult {
    if (!cidr || cidr.trim().length === 0) {
      return {
        isValid: false,
        error: 'Target cannot be empty'
      }
    }

    const trimmedCIDR = cidr.trim()

    // Check if contains /
    if (!trimmedCIDR.includes('/')) {
      return {
        isValid: false,
        error: 'Invalid CIDR format, should contain /'
      }
    }

    const [ip, prefix] = trimmedCIDR.split('/')

    // Validate IP part
    if (!validator.isIP(ip.trim())) {
      return {
        isValid: false,
        error: 'Invalid IP address format in CIDR'
      }
    }

    // Validate prefix length
    const prefixNum = parseInt(prefix, 10)
    if (isNaN(prefixNum) || prefixNum < 0 || prefixNum > 32) {
      return {
        isValid: false,
        error: 'CIDR prefix length must be between 0-32'
      }
    }

    return { isValid: true, type: 'cidr' }
  }

  /**
   * Auto-detect target type and validate
   * Supports: domain, IPv4, IPv6, CIDR
   */
  static validateTarget(target: string): TargetValidationResult {
    if (!target || target.trim().length === 0) {
      return {
        isValid: false,
        error: 'Target cannot be empty'
      }
    }

    const trimmedTarget = target.trim()

    // 1. Try CIDR validation first (contains /)
    if (trimmedTarget.includes('/')) {
      return this.validateCIDR(trimmedTarget)
    }

    // 2. Try IP validation
    if (validator.isIP(trimmedTarget)) {
      return this.validateIP(trimmedTarget)
    }

    // 3. Try domain validation
    return this.validateDomain(trimmedTarget)
  }

  /**
   * Batch validate target list
   */
  static validateTargetBatch(targets: string[]): Array<TargetValidationResult & { index: number; originalTarget: string }> {
    return targets.map((target, index) => ({
      ...this.validateTarget(target),
      index,
      originalTarget: target
    }))
  }

  // ==================== URL Support Extension ====================

  /**
   * Detect input type
   * Used for quick scan input parsing
   */
  static detectInputType(input: string): InputType {
    // URL: contains :// 
    if (input.includes('://')) {
      return 'url'
    }
    
    // Contains / but not CIDR, treat as URL (URL missing scheme)
    if (input.includes('/')) {
      // CIDR format: IP/prefix, e.g. 10.0.0.0/8
      if (this.looksLikeCidr(input)) {
        return 'cidr'
      }
      return 'url'
    }
    
    // CIDR: matches IP/prefix format
    if (this.looksLikeCidr(input)) {
      return 'cidr'
    }
    
    // IP: matches IPv4 format
    if (this.looksLikeIp(input)) {
      return 'ip'
    }
    
    // Default to domain
    return 'domain'
  }

  /**
   * Check if it's CIDR format
   */
  static looksLikeCidr(input: string): boolean {
    return /^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\/\d{1,2}$/.test(input)
  }

  /**
   * Check if it's IP address format
   */
  static looksLikeIp(input: string): boolean {
    return /^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$/.test(input)
  }

  /**
   * Validate URL format
   * URL must contain scheme (http:// or https://)
   */
  static validateUrl(url: string): TargetValidationResult {
    // Check if contains scheme
    if (!url.startsWith('http://') && !url.startsWith('https://')) {
      return { isValid: false, error: 'URL must contain protocol (http:// or https://)' }
    }
    
    try {
      const parsed = new URL(url)
      if (!parsed.hostname) {
        return { isValid: false, error: 'URL must contain hostname' }
      }
      return { isValid: true, type: 'url' }
    } catch {
      return { isValid: false, error: 'Invalid URL format' }
    }
  }

  /**
   * Extended validation method, supports URL input
   * Only does type detection and basic format validation, detailed parsing is done by backend
   */
  static validateInput(input: string): TargetValidationResult {
    const trimmed = input.trim()
    
    // 1. Skip empty lines, no error
    if (trimmed.length === 0) {
      return { isValid: true, type: undefined, isEmptyLine: true }
    }
    
    // 2. Detect input type
    const inputType = this.detectInputType(trimmed)
    
    // 3. Validate based on type
    switch (inputType) {
      case 'url':
        return this.validateUrl(trimmed)
      case 'ip':
        return this.validateIP(trimmed)
      case 'cidr':
        return this.validateCIDR(trimmed)
      case 'domain':
        return this.validateDomain(trimmed)
      default:
        return this.validateDomain(trimmed)
    }
  }

  /**
   * Batch validate input (supports URL)
   */
  static validateInputBatch(inputs: string[]): Array<TargetValidationResult & { lineNumber: number; originalInput: string }> {
    return inputs.map((input, index) => ({
      ...this.validateInput(input),
      lineNumber: index + 1,
      originalInput: input
    }))
  }
}
