import type { Endpoint, GetEndpointsResponse } from '@/types/endpoint.types'

export const mockEndpoints: Endpoint[] = [
  {
    id: 1,
    url: 'https://acme.com/',
    method: 'GET',
    statusCode: 200,
    title: 'Acme Corporation - Home',
    contentLength: 45678,
    contentType: 'text/html; charset=utf-8',
    host: 'acme.com',
    webserver: 'nginx/1.24.0',
    tech: ['React', 'Next.js', 'Node.js'],
    createdAt: '2024-12-28T10:00:00Z',
  },
  {
    id: 2,
    url: 'https://acme.com/login',
    method: 'GET',
    statusCode: 200,
    title: 'Login - Acme',
    contentLength: 12345,
    contentType: 'text/html; charset=utf-8',
    host: 'acme.com',
    webserver: 'nginx/1.24.0',
    tech: ['React', 'Next.js'],
    createdAt: '2024-12-28T10:01:00Z',
  },
  {
    id: 3,
    url: 'https://api.acme.com/v1/users',
    method: 'GET',
    statusCode: 200,
    title: '',
    contentLength: 8923,
    contentType: 'application/json',
    host: 'api.acme.com',
    webserver: 'nginx/1.24.0',
    tech: ['Django', 'Python'],
    createdAt: '2024-12-28T10:02:00Z',
  },
  {
    id: 4,
    url: 'https://api.acme.com/v1/products',
    method: 'GET',
    statusCode: 200,
    title: '',
    contentLength: 23456,
    contentType: 'application/json',
    host: 'api.acme.com',
    webserver: 'nginx/1.24.0',
    tech: ['Django', 'Python'],
    createdAt: '2024-12-28T10:03:00Z',
  },
  {
    id: 16,
    url: 'https://api.acme.com/a',
    method: 'GET',
    statusCode: 200,
    title: '',
    contentLength: 512,
    contentType: 'application/json',
    host: 'api.acme.com',
    webserver: 'nginx/1.24.0',
    tech: ['Django', 'Python'],
    createdAt: '2024-12-28T10:03:10Z',
  },
  {
    id: 17,
    url: 'https://api.acme.com/a/child',
    method: 'GET',
    statusCode: 200,
    title: '',
    contentLength: 512,
    contentType: 'application/json',
    host: 'api.acme.com',
    webserver: 'nginx/1.24.0',
    tech: ['Django', 'Python'],
    createdAt: '2024-12-28T10:03:20Z',
  },
  {
    id: 18,
    url: 'https://api.acme.com/ab',
    method: 'GET',
    statusCode: 200,
    title: '',
    contentLength: 512,
    contentType: 'application/json',
    host: 'api.acme.com',
    webserver: 'nginx/1.24.0',
    tech: ['Django', 'Python'],
    createdAt: '2024-12-28T10:03:30Z',
  },
  {
    id: 19,
    url: 'https://api.acme.com.evil/a',
    method: 'GET',
    statusCode: 200,
    title: '',
    contentLength: 512,
    contentType: 'application/json',
    host: 'api.acme.com.evil',
    webserver: 'nginx/1.24.0',
    tech: ['Django', 'Python'],
    createdAt: '2024-12-28T10:03:40Z',
  },
  {
    id: 20,
    url: 'http://api.acme.com/a',
    method: 'GET',
    statusCode: 200,
    title: '',
    contentLength: 512,
    contentType: 'application/json',
    host: 'api.acme.com',
    webserver: 'nginx/1.24.0',
    tech: ['Django', 'Python'],
    createdAt: '2024-12-28T10:03:50Z',
  },
  {
    id: 21,
    url: 'https://api.acme.com:8443/a',
    method: 'GET',
    statusCode: 200,
    title: '',
    contentLength: 512,
    contentType: 'application/json',
    host: 'api.acme.com',
    webserver: 'nginx/1.24.0',
    tech: ['Django', 'Python'],
    createdAt: '2024-12-28T10:04:00Z',
  },
  {
    id: 5,
    url: 'https://acme.io/docs',
    method: 'GET',
    statusCode: 200,
    title: 'Documentation - Acme.io',
    contentLength: 67890,
    contentType: 'text/html; charset=utf-8',
    host: 'acme.io',
    webserver: 'cloudflare',
    tech: ['Vue.js', 'Vitepress'],
    createdAt: '2024-12-27T14:30:00Z',
  },
  {
    id: 6,
    url: 'https://acme.io/api/config',
    method: 'GET',
    statusCode: 200,
    title: '',
    contentLength: 1234,
    contentType: 'application/json',
    host: 'acme.io',
    webserver: 'cloudflare',
    tech: ['Node.js', 'Express'],
    createdAt: '2024-12-27T14:31:00Z',
  },
  {
    id: 7,
    url: 'https://techstart.io/',
    method: 'GET',
    statusCode: 200,
    title: 'TechStart - Innovation Hub',
    contentLength: 34567,
    contentType: 'text/html; charset=utf-8',
    host: 'techstart.io',
    webserver: 'Apache/2.4.54',
    tech: ['WordPress', 'PHP'],
    createdAt: '2024-12-26T08:45:00Z',
  },
  {
    id: 8,
    url: 'https://techstart.io/admin',
    method: 'GET',
    statusCode: 302,
    title: '',
    contentLength: 0,
    contentType: 'text/html',
    location: 'https://techstart.io/admin/login',
    host: 'techstart.io',
    webserver: 'Apache/2.4.54',
    tech: ['WordPress', 'PHP'],
    createdAt: '2024-12-26T08:46:00Z',
  },
  {
    id: 9,
    url: 'https://globalfinance.com/',
    method: 'GET',
    statusCode: 200,
    title: 'Global Finance - Your Financial Partner',
    contentLength: 56789,
    contentType: 'text/html; charset=utf-8',
    host: 'globalfinance.com',
    webserver: 'Microsoft-IIS/10.0',
    tech: ['ASP.NET', 'C#', 'jQuery'],
    createdAt: '2024-12-25T16:20:00Z',
  },
  {
    id: 10,
    url: 'https://globalfinance.com/.git/config',
    method: 'GET',
    statusCode: 200,
    title: '',
    contentLength: 456,
    contentType: 'text/plain',
    host: 'globalfinance.com',
    webserver: 'Microsoft-IIS/10.0',
    createdAt: '2024-12-25T16:21:00Z',
  },
  {
    id: 11,
    url: 'https://retailmax.com/',
    method: 'GET',
    statusCode: 200,
    title: 'RetailMax - Shop Everything',
    contentLength: 89012,
    contentType: 'text/html; charset=utf-8',
    host: 'retailmax.com',
    webserver: 'nginx/1.22.0',
    tech: ['React', 'Redux', 'Node.js'],
    createdAt: '2024-12-21T10:45:00Z',
  },
  {
    id: 12,
    url: 'https://retailmax.com/product?id=1',
    method: 'GET',
    statusCode: 200,
    title: 'Product Detail - RetailMax',
    contentLength: 23456,
    contentType: 'text/html; charset=utf-8',
    host: 'retailmax.com',
    webserver: 'nginx/1.22.0',
    tech: ['React', 'Redux'],
    createdAt: '2024-12-21T10:46:00Z',
  },
  {
    id: 13,
    url: 'https://healthcareplus.com/',
    method: 'GET',
    statusCode: 200,
    title: 'HealthCare Plus - Digital Health',
    contentLength: 45678,
    contentType: 'text/html; charset=utf-8',
    host: 'healthcareplus.com',
    webserver: 'nginx/1.24.0',
    tech: ['Angular', 'TypeScript'],
    createdAt: '2024-12-23T11:00:00Z',
  },
  {
    id: 14,
    url: 'https://edutech.io/',
    method: 'GET',
    statusCode: 200,
    title: 'EduTech - Learn Anywhere',
    contentLength: 67890,
    contentType: 'text/html; charset=utf-8',
    host: 'edutech.io',
    webserver: 'cloudflare',
    tech: ['Vue.js', 'Nuxt.js'],
    createdAt: '2024-12-22T13:30:00Z',
  },
  {
    id: 15,
    url: 'https://cloudnine.host/',
    method: 'GET',
    statusCode: 200,
    title: 'CloudNine Hosting',
    contentLength: 34567,
    contentType: 'text/html; charset=utf-8',
    host: 'cloudnine.host',
    webserver: 'LiteSpeed',
    tech: ['PHP', 'Laravel'],
    createdAt: '2024-12-19T16:00:00Z',
  },
]

export function getMockEndpoints(params?: {
  pageSize?: number
  pageToken?: string
  filter?: string
  orderBy?: string
}): GetEndpointsResponse {
  const pageTokenMatch = params?.pageToken?.match(/^mock-endpoint-page-(\d+)$/)
  const page = pageTokenMatch ? Number.parseInt(pageTokenMatch[1], 10) : 1
  const pageSize = params?.pageSize || 10
  const search = params?.filter?.match(/url(?:==|=)"((?:\\.|[^"\\])*)"/)?.[1]
    ?.replace(/\\"/g, '"')
    .replace(/\\\\/g, "\\")

  let filtered = mockEndpoints

  if (search !== undefined) {
    filtered = mockEndpoints.filter(
      ep => ep.url === search
    )
  }

  filtered = [...filtered].sort((left, right) => {
    const leftTime = new Date(left.createdAt ?? '').getTime()
    const rightTime = new Date(right.createdAt ?? '').getTime()
    return rightTime - leftTime || right.id - left.id
  })

  const total = filtered.length
  const totalPages = Math.ceil(total / pageSize)
  const start = (page - 1) * pageSize
  const results = filtered.slice(start, start + pageSize)
  const nextPage = page < totalPages ? page + 1 : undefined

  return {
    results,
    total,
    page,
    pageSize,
    totalPages,
    totalSize: total,
    ...(nextPage && { nextPageToken: `mock-endpoint-page-${nextPage}` }),
  }
}

export function getMockEndpointById(id: number): Endpoint | undefined {
  return mockEndpoints.find(ep => ep.id === id)
}

export function deleteMockEndpoint(id: number): boolean {
  const index = mockEndpoints.findIndex(endpoint => endpoint.id === id)
  if (index === -1) {
    return false
  }

  mockEndpoints.splice(index, 1)
  return true
}

export function batchDeleteMockEndpoints(ids: number[]) {
  let deletedCount = 0

  for (const id of ids) {
    if (deleteMockEndpoint(id)) {
      deletedCount += 1
    }
  }

  return {
    deletedCount,
  }
}
