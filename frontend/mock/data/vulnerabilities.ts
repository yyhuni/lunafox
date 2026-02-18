import type { Vulnerability, GetVulnerabilitiesResponse, VulnerabilitySeverity } from '@/types/vulnerability.types'

export const mockVulnerabilities: Vulnerability[] = [
  {
    id: 1,
    target: 1,
    url: 'https://acme.com/search?q=test',
    vulnType: 'xss-reflected',
    severity: 'critical',
    source: 'dalfox',
    cvssScore: 9.1,
    description: 'Reflected XSS in search parameter',
    rawOutput: {
      type: 'R',
      inject_type: 'inHTML-URL',
      method: 'GET',
      data: 'https://acme.com/search?q=<script>alert(1)</script>',
      param: 'q',
      payload: '<script>alert(1)</script>',
      evidence: '<script>alert(1)</script>',
      cwe: 'CWE-79',
    },
    createdAt: '2024-12-28T10:30:00Z',
    isReviewed: false,
  },
  {
    id: 2,
    target: 1,
    url: 'https://api.acme.com/v1/users',
    vulnType: 'CVE-2024-1234',
    severity: 'high',
    source: 'nuclei',
    cvssScore: 8.5,
    description: 'SQL Injection in user API endpoint',
    rawOutput: {
      'template-id': 'CVE-2024-1234',
      'matched-at': 'https://api.acme.com/v1/users',
      host: 'api.acme.com',
      info: {
        name: 'SQL Injection',
        description: 'SQL injection vulnerability in user endpoint',
        severity: 'high',
        tags: ['sqli', 'cve'],
        reference: ['https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2024-1234'],
        classification: {
          'cve-id': 'CVE-2024-1234',
          'cwe-id': ['CWE-89'],
        },
      },
    },
    createdAt: '2024-12-28T10:45:00Z',
    isReviewed: true,
  },
  {
    id: 3,
    target: 1,
    url: 'https://acme.com/login',
    vulnType: 'xss-stored',
    severity: 'high',
    source: 'dalfox',
    cvssScore: 8.2,
    description: 'Stored XSS in user profile',
    rawOutput: {
      type: 'S',
      inject_type: 'inHTML-TAG',
      method: 'POST',
      param: 'bio',
      payload: '<img src=x onerror=alert(1)>',
    },
    createdAt: '2024-12-27T14:20:00Z',
    isReviewed: false,
  },
  {
    id: 4,
    target: 2,
    url: 'https://acme.io/api/config',
    vulnType: 'information-disclosure',
    severity: 'medium',
    source: 'nuclei',
    cvssScore: 5.3,
    description: 'Exposed configuration file',
    rawOutput: {
      'template-id': 'exposed-config',
      'matched-at': 'https://acme.io/api/config',
      host: 'acme.io',
      info: {
        name: 'Exposed Configuration',
        description: 'Configuration file accessible without authentication',
        severity: 'medium',
        tags: ['exposure', 'config'],
      },
    },
    createdAt: '2024-12-27T15:00:00Z',
    isReviewed: true,
  },
  {
    id: 5,
    target: 3,
    url: 'https://techstart.io/admin',
    vulnType: 'open-redirect',
    severity: 'medium',
    source: 'nuclei',
    cvssScore: 4.7,
    description: 'Open redirect vulnerability',
    rawOutput: {
      'template-id': 'open-redirect',
      'matched-at': 'https://techstart.io/admin?redirect=evil.com',
      host: 'techstart.io',
      info: {
        name: 'Open Redirect',
        description: 'URL redirect without validation',
        severity: 'medium',
        tags: ['redirect'],
      },
    },
    createdAt: '2024-12-26T09:30:00Z',
    isReviewed: false,
  },
  {
    id: 6,
    target: 4,
    url: 'https://globalfinance.com/.git/config',
    vulnType: 'git-config-exposure',
    severity: 'high',
    source: 'nuclei',
    cvssScore: 7.5,
    description: 'Git configuration file exposed',
    rawOutput: {
      'template-id': 'git-config',
      'matched-at': 'https://globalfinance.com/.git/config',
      host: 'globalfinance.com',
      info: {
        name: 'Git Config Exposure',
        description: 'Git configuration file is publicly accessible',
        severity: 'high',
        tags: ['git', 'exposure'],
      },
    },
    createdAt: '2024-12-25T11:15:00Z',
    isReviewed: true,
  },
  {
    id: 7,
    target: 8,
    url: 'https://retailmax.com/product?id=1',
    vulnType: 'sqli',
    severity: 'critical',
    source: 'nuclei',
    cvssScore: 9.8,
    description: 'SQL Injection in product parameter',
    rawOutput: {
      'template-id': 'generic-sqli',
      'matched-at': "https://retailmax.com/product?id=1'",
      host: 'retailmax.com',
      info: {
        name: 'SQL Injection',
        description: 'SQL injection in product ID parameter',
        severity: 'critical',
        tags: ['sqli'],
        classification: {
          'cwe-id': ['CWE-89'],
        },
      },
    },
    createdAt: '2024-12-21T12:00:00Z',
    isReviewed: false,
  },
  {
    id: 8,
    target: 1,
    url: 'https://acme.com/robots.txt',
    vulnType: 'robots-txt-exposure',
    severity: 'info',
    source: 'nuclei',
    description: 'Robots.txt file found',
    rawOutput: {
      'template-id': 'robots-txt',
      'matched-at': 'https://acme.com/robots.txt',
      host: 'acme.com',
      info: {
        name: 'Robots.txt',
        description: 'Robots.txt file detected',
        severity: 'info',
        tags: ['misc'],
      },
    },
    createdAt: '2024-12-28T10:00:00Z',
    isReviewed: true,
  },
  {
    id: 9,
    target: 2,
    url: 'https://acme.io/sitemap.xml',
    vulnType: 'sitemap-exposure',
    severity: 'info',
    source: 'nuclei',
    description: 'Sitemap.xml file found',
    rawOutput: {
      'template-id': 'sitemap-xml',
      'matched-at': 'https://acme.io/sitemap.xml',
      host: 'acme.io',
      info: {
        name: 'Sitemap.xml',
        description: 'Sitemap.xml file detected',
        severity: 'info',
        tags: ['misc'],
      },
    },
    createdAt: '2024-12-27T14:00:00Z',
    isReviewed: true,
  },
  {
    id: 10,
    target: 3,
    url: 'https://techstart.io/api/v2/debug',
    vulnType: 'debug-endpoint',
    severity: 'low',
    source: 'nuclei',
    cvssScore: 3.1,
    description: 'Debug endpoint exposed',
    rawOutput: {
      'template-id': 'debug-endpoint',
      'matched-at': 'https://techstart.io/api/v2/debug',
      host: 'techstart.io',
      info: {
        name: 'Debug Endpoint',
        description: 'Debug endpoint accessible in production',
        severity: 'low',
        tags: ['debug', 'exposure'],
      },
    },
    createdAt: '2024-12-26T10:00:00Z',
    isReviewed: false,
  },
]

export function getMockVulnerabilities(params?: {
  page?: number
  pageSize?: number
  targetId?: number
  severity?: VulnerabilitySeverity
  search?: string
}): GetVulnerabilitiesResponse {
  const page = params?.page || 1
  const pageSize = params?.pageSize || 10
  const targetId = params?.targetId
  const severity = params?.severity
  const search = params?.search?.toLowerCase() || ''

  let filtered = mockVulnerabilities

  if (targetId) {
    filtered = filtered.filter(v => v.target === targetId)
  }

  if (severity) {
    filtered = filtered.filter(v => v.severity === severity)
  }

  if (search) {
    filtered = filtered.filter(
      v =>
        v.url.toLowerCase().includes(search) ||
        v.vulnType.toLowerCase().includes(search) ||
        v.description?.toLowerCase().includes(search)
    )
  }

  const total = filtered.length
  const totalPages = Math.ceil(total / pageSize)
  const start = (page - 1) * pageSize
  const vulnerabilities = filtered.slice(start, start + pageSize)

  return {
    vulnerabilities,
    total,
    page,
    pageSize,
    totalPages,
  }
}

export function getMockVulnerabilityById(id: number): Vulnerability | undefined {
  return mockVulnerabilities.find(v => v.id === id)
}
