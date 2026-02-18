/**
 * Fingerprint related type definitions
 */

// EHole fingerprint type
export interface EholeFingerprint {
  id: number
  cms: string
  method: string
  location: string
  keyword: string[]
  isImportant: boolean
  type: string
  createdAt: string
}

// Goby rule type
export interface GobyRule {
  label: string
  feature: string
  is_equal: boolean
}

// Goby fingerprint type
export interface GobyFingerprint {
  id: number
  name: string
  logic: string
  rule: GobyRule[]
  createdAt: string
}

// Wappalyzer fingerprint type
export interface WappalyzerFingerprint {
  id: number
  name: string
  cats: number[]
  cookies: Record<string, string>
  headers: Record<string, string>
  scriptSrc: string[]
  js: string[]
  implies: string[]
  meta: Record<string, string[]>
  html: string[]
  description: string
  website: string
  cpe: string
  createdAt: string
}

// Fingers rule type
export interface FingersRule {
  regexps?: {
    regexp: string
    group?: number
  }[]
  vuln?: string
  version?: string
  body?: string
  header?: string
  title?: string
  send_data_type?: string
  send_data?: string
  favicon_hash?: string[]
}

// Fingers fingerprint type
export interface FingersFingerprint {
  id: number
  name: string
  link: string
  rule: FingersRule[]
  tag: string[]
  focus: boolean
  defaultPort: number[]
  createdAt: string
}

// FingerPrintHub HTTP matcher type
export interface FingerPrintHubHttpMatcher {
  method?: string
  path?: string
  matchers?: {
    type: string
    part?: string
    words?: string[]
    regex?: string[]
    status?: number[]
    condition?: string
  }[]
  extractors?: {
    type: string
    part?: string
    regex?: string[]
    group?: number
  }[]
}

// FingerPrintHub metadata type
export interface FingerPrintHubMetadata {
  product?: string
  vendor?: string
  verified?: boolean
  shodan_query?: string
  fofa_query?: string
}

// FingerPrintHub fingerprint type
export interface FingerPrintHubFingerprint {
  id: number
  fpId: string
  name: string
  author: string
  tags: string
  severity: string
  metadata: FingerPrintHubMetadata
  http: FingerPrintHubHttpMatcher[]
  sourceFile: string
  createdAt: string
}

// ARL fingerprint type
export interface ARLFingerprint {
  id: number
  name: string
  rule: string
  createdAt: string
}

// Batch create response
export interface BatchCreateResponse {
  created: number
  failed: number
}

// Bulk delete response
export interface BulkDeleteResponse {
  deleted: number
}

// Fingerprint statistics
export interface FingerprintStats {
  ehole: number
  goby: number
  wappalyzer: number
  fingers: number
  fingerprinthub: number
  arl: number
}
