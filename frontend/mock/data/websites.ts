import type { FilterOptionsResponse, WebSite, WebSiteListResponse, WebsiteFilterOptionField } from '@/types/website.types'
import { mockScreenshots } from './screenshots'

export const mockWebsites: WebSite[] = [
  {
    id: 1,
    target: 1,
    url: 'https://acme.com',
    host: 'acme.com',
    location: '',
    title: 'Acme Corporation - Home',
    webserver: 'nginx/1.24.0',
    contentType: 'text/html; charset=utf-8',
    statusCode: 200,
    contentLength: 45678,
    responseBody: '<!DOCTYPE html>...',
    tech: ['React', 'Next.js', 'Node.js', 'Tailwind CSS', 'TypeScript', 'nginx', 'Vercel', 'Cloudflare', 'GraphQL', 'Sentry', 'PostgreSQL', 'Redis'],
    vhost: false,
    subdomain: 'acme.com',
    createdAt: '2024-12-28T10:00:00Z',
  },
  {
    id: 2,
    target: 1,
    url: 'https://www.acme.com',
    host: 'www.acme.com',
    location: 'https://acme.com',
    title: 'Acme Corporation - Home',
    webserver: 'nginx/1.24.0',
    contentType: 'text/html; charset=utf-8',
    statusCode: 301,
    contentLength: 0,
    responseBody: '',
    tech: [],
    vhost: false,
    subdomain: 'www.acme.com',
    createdAt: '2024-12-28T10:01:00Z',
  },
  {
    id: 3,
    target: 1,
    url: 'https://api.acme.com',
    host: 'api.acme.com',
    location: '',
    title: 'Acme API',
    webserver: 'nginx/1.24.0',
    contentType: 'application/json',
    statusCode: 200,
    contentLength: 234,
    responseBody: '{"status":"ok","version":"1.0"}',
    tech: ['Django', 'Python', 'PostgreSQL'],
    vhost: false,
    subdomain: 'api.acme.com',
    createdAt: '2024-12-28T10:02:00Z',
  },
  {
    id: 4,
    target: 1,
    url: 'https://admin.acme.com',
    host: 'admin.acme.com',
    location: '',
    title: 'Admin Panel - Acme',
    webserver: 'nginx/1.24.0',
    contentType: 'text/html; charset=utf-8',
    statusCode: 200,
    contentLength: 23456,
    responseBody: '<!DOCTYPE html>...',
    tech: ['React', 'Ant Design'],
    vhost: false,
    subdomain: 'admin.acme.com',
    createdAt: '2024-12-28T10:03:00Z',
  },
  {
    id: 5,
    target: 2,
    url: 'https://acme.io',
    host: 'acme.io',
    location: '',
    title: 'Acme Developer Platform',
    webserver: 'cloudflare',
    contentType: 'text/html; charset=utf-8',
    statusCode: 200,
    contentLength: 56789,
    responseBody: '<!DOCTYPE html>...',
    tech: ['Vue.js', 'Vitepress', 'CloudFlare'],
    vhost: false,
    subdomain: 'acme.io',
    createdAt: '2024-12-27T14:30:00Z',
  },
  {
    id: 6,
    target: 2,
    url: 'https://docs.acme.io',
    host: 'docs.acme.io',
    location: '',
    title: 'Documentation - Acme.io',
    webserver: 'cloudflare',
    contentType: 'text/html; charset=utf-8',
    statusCode: 200,
    contentLength: 67890,
    responseBody: '<!DOCTYPE html>...',
    tech: ['Vue.js', 'Vitepress'],
    vhost: false,
    subdomain: 'docs.acme.io',
    createdAt: '2024-12-27T14:31:00Z',
  },
  {
    id: 7,
    target: 3,
    url: 'https://techstart.io',
    host: 'techstart.io',
    location: '',
    title: 'TechStart - Innovation Hub',
    webserver: 'Apache/2.4.54',
    contentType: 'text/html; charset=utf-8',
    statusCode: 200,
    contentLength: 34567,
    responseBody: '<!DOCTYPE html>...',
    tech: ['WordPress', 'PHP', 'MySQL'],
    vhost: false,
    subdomain: 'techstart.io',
    createdAt: '2024-12-26T08:45:00Z',
  },
  {
    id: 8,
    target: 4,
    url: 'https://globalfinance.com',
    host: 'globalfinance.com',
    location: '',
    title: 'Global Finance - Your Financial Partner',
    webserver: 'Microsoft-IIS/10.0',
    contentType: 'text/html; charset=utf-8',
    statusCode: 200,
    contentLength: 56789,
    responseBody: '<!DOCTYPE html>...',
    tech: ['ASP.NET', 'C#', 'jQuery', 'SQL Server'],
    vhost: false,
    subdomain: 'globalfinance.com',
    createdAt: '2024-12-25T16:20:00Z',
  },
  {
    id: 9,
    target: 6,
    url: 'https://healthcareplus.com',
    host: 'healthcareplus.com',
    location: '',
    title: 'HealthCare Plus - Digital Health',
    webserver: 'nginx/1.24.0',
    contentType: 'text/html; charset=utf-8',
    statusCode: 200,
    contentLength: 45678,
    responseBody: '<!DOCTYPE html>...',
    tech: ['Angular', 'TypeScript', 'Node.js'],
    vhost: false,
    subdomain: 'healthcareplus.com',
    createdAt: '2024-12-23T11:00:00Z',
  },
  {
    id: 10,
    target: 7,
    url: 'https://edutech.io',
    host: 'edutech.io',
    location: '',
    title: 'EduTech - Learn Anywhere',
    webserver: 'cloudflare',
    contentType: 'text/html; charset=utf-8',
    statusCode: 200,
    contentLength: 67890,
    responseBody: '<!DOCTYPE html>...',
    tech: ['Vue.js', 'Nuxt.js', 'PostgreSQL'],
    vhost: false,
    subdomain: 'edutech.io',
    createdAt: '2024-12-22T13:30:00Z',
  },
  {
    id: 11,
    target: 8,
    url: 'https://retailmax.com',
    host: 'retailmax.com',
    location: '',
    title: 'RetailMax - Shop Everything',
    webserver: 'nginx/1.22.0',
    contentType: 'text/html; charset=utf-8',
    statusCode: 200,
    contentLength: 89012,
    responseBody: '<!DOCTYPE html>...',
    tech: ['React', 'Redux', 'Node.js', 'MongoDB'],
    vhost: false,
    subdomain: 'retailmax.com',
    createdAt: '2024-12-21T10:45:00Z',
  },
  {
    id: 12,
    target: 10,
    url: 'https://cloudnine.host',
    host: 'cloudnine.host',
    location: '',
    title: 'CloudNine Hosting',
    webserver: 'LiteSpeed',
    contentType: 'text/html; charset=utf-8',
    statusCode: 200,
    contentLength: 34567,
    responseBody: '<!DOCTYPE html>...',
    tech: ['PHP', 'Laravel', 'MySQL'],
    vhost: false,
    subdomain: 'cloudnine.host',
    createdAt: '2024-12-19T16:00:00Z',
  },
]

function getMockExactURLFilter(filter?: string) {
  return filter?.match(/url(?:==|=)"((?:\\.|[^"\\])*)"/)?.[1]
    ?.replace(/\\"/g, '"')
    .replace(/\\\\/g, "\\")
}

export function getMockWebsites(params?: {
  page?: number
  pageToken?: string
  pageSize?: number
  filter?: string
  targetId?: number
}): WebSiteListResponse {
  const page = params?.pageToken?.match(/^mock-website-page-(\d+)$/)
    ? Number.parseInt(params.pageToken.slice("mock-website-page-".length), 10)
    : params?.page || 1
  const pageSize = params?.pageSize || 10
  const url = getMockExactURLFilter(params?.filter)
  const targetId = params?.targetId

  let filtered = mockWebsites

  if (targetId) {
    filtered = filtered.filter(w => w.target === targetId)
  }

  if (url !== undefined) {
    filtered = filtered.filter((website) => website.url === url)
  }

  const total = filtered.length
  const totalPages = Math.ceil(total / pageSize)
  const start = (page - 1) * pageSize
  const results = filtered.slice(start, start + pageSize).map(toMockWebsiteReadModel)

  return {
    results,
    total,
    page,
    pageSize,
    totalPages,
    ...(page < totalPages && { nextPageToken: `mock-website-page-${page + 1}` }),
  }
}

export function getMockWebsiteById(id: number): WebSite | undefined {
  const website = mockWebsites.find(w => w.id === id)
  return website ? toMockWebsiteReadModel(website) : undefined
}

function toMockWebsiteReadModel(website: WebSite): WebSite {
  // Screenshot remains its own Target-scoped resource. The mock projection
  // mirrors the real read model by joining only its natural target + URL key.
  const screenshot = mockScreenshots.find(
    (candidate) => candidate.targetId === website.target && candidate.url === website.url,
  )

  return {
    ...website,
    resourceName: `targets/${website.target}/websites/${website.id}`,
    ...(screenshot ? {
      screenshot: {
        id: screenshot.id,
        resourceName: `targets/${website.target}/screenshots/${screenshot.id}`,
        url: website.url,
        statusCode: screenshot.statusCode,
        createdAt: screenshot.createdAt,
        updatedAt: screenshot.updatedAt ?? screenshot.createdAt,
      },
    } : {}),
  }
}

export function getMockWebsiteFilterOptions(params: {
  field: WebsiteFilterOptionField
  targetId?: number
  scanId?: number
}): FilterOptionsResponse {
  let scoped = mockWebsites

  if (params.targetId) {
    scoped = scoped.filter((website) => website.target === params.targetId)
  }
  if (params.scanId) {
    scoped = scoped.filter((website) => website.scan === params.scanId)
  }

  const counts = new Map<string, number>()
  for (const website of scoped) {
    const values = getWebsiteFilterValues(website, params.field)
    for (const value of values) {
      counts.set(value, (counts.get(value) ?? 0) + 1)
    }
  }

  return {
    results: Array.from(counts.entries())
      .sort(([left], [right]) => left.localeCompare(right, undefined, { numeric: true }))
      .map(([value, count]) => ({ value, label: value, count })),
  }
}

function getWebsiteFilterValues(website: WebSite, field: WebsiteFilterOptionField): string[] {
  switch (field) {
    case 'statusCode':
      return Number.isFinite(website.statusCode) ? [String(website.statusCode)] : []
    case 'tech':
      return website.tech.filter((value) => value !== '')
    case 'webserver':
      return website.webserver ? [website.webserver] : []
    case 'contentType':
      return website.contentType ? [website.contentType] : []
    case 'vhost':
      return website.vhost === null ? [] : [String(website.vhost)]
  }
}

export function deleteMockWebsite(id: number): WebSite | undefined {
  const index = mockWebsites.findIndex(website => website.id === id)
  if (index === -1) {
    return undefined
  }

  const [deleted] = mockWebsites.splice(index, 1)
  return deleted
}

export function bulkDeleteMockWebsites(ids: number[]) {
  let deletedCount = 0

  for (const id of ids) {
    if (deleteMockWebsite(id)) {
      deletedCount += 1
    }
  }

  return {
    message: "Deleted websites",
    deletedCount,
    requestedIds: ids,
    cascadeDeleted: {},
  }
}
