import type { Subdomain, GetAllSubdomainsResponse } from '@/types/subdomain.types'

export const mockSubdomains: Subdomain[] = [
  { id: 1, name: 'acme.com', createdAt: '2024-12-28T10:00:00Z' },
  { id: 2, name: 'www.acme.com', createdAt: '2024-12-28T10:01:00Z' },
  { id: 3, name: 'api.acme.com', createdAt: '2024-12-28T10:02:00Z' },
  { id: 4, name: 'admin.acme.com', createdAt: '2024-12-28T10:03:00Z' },
  { id: 5, name: 'mail.acme.com', createdAt: '2024-12-28T10:04:00Z' },
  { id: 6, name: 'blog.acme.com', createdAt: '2024-12-28T10:05:00Z' },
  { id: 7, name: 'shop.acme.com', createdAt: '2024-12-28T10:06:00Z' },
  { id: 8, name: 'cdn.acme.com', createdAt: '2024-12-28T10:07:00Z' },
  { id: 9, name: 'static.acme.com', createdAt: '2024-12-28T10:08:00Z' },
  { id: 10, name: 'dev.acme.com', createdAt: '2024-12-28T10:09:00Z' },
  { id: 11, name: 'staging.acme.com', createdAt: '2024-12-28T10:10:00Z' },
  { id: 12, name: 'test.acme.com', createdAt: '2024-12-28T10:11:00Z' },
  { id: 13, name: 'acme.io', createdAt: '2024-12-27T14:30:00Z' },
  { id: 14, name: 'docs.acme.io', createdAt: '2024-12-27T14:31:00Z' },
  { id: 15, name: 'api.acme.io', createdAt: '2024-12-27T14:32:00Z' },
  { id: 16, name: 'status.acme.io', createdAt: '2024-12-27T14:33:00Z' },
  { id: 17, name: 'techstart.io', createdAt: '2024-12-26T08:45:00Z' },
  { id: 18, name: 'www.techstart.io', createdAt: '2024-12-26T08:46:00Z' },
  { id: 19, name: 'app.techstart.io', createdAt: '2024-12-26T08:47:00Z' },
  { id: 20, name: 'globalfinance.com', createdAt: '2024-12-25T16:20:00Z' },
  { id: 21, name: 'www.globalfinance.com', createdAt: '2024-12-25T16:21:00Z' },
  { id: 22, name: 'secure.globalfinance.com', createdAt: '2024-12-25T16:22:00Z' },
  { id: 23, name: 'portal.globalfinance.com', createdAt: '2024-12-25T16:23:00Z' },
  { id: 24, name: 'healthcareplus.com', createdAt: '2024-12-23T11:00:00Z' },
  { id: 25, name: 'www.healthcareplus.com', createdAt: '2024-12-23T11:01:00Z' },
  { id: 26, name: 'patient.healthcareplus.com', createdAt: '2024-12-23T11:02:00Z' },
  { id: 27, name: 'edutech.io', createdAt: '2024-12-22T13:30:00Z' },
  { id: 28, name: 'learn.edutech.io', createdAt: '2024-12-22T13:31:00Z' },
  { id: 29, name: 'retailmax.com', createdAt: '2024-12-21T10:45:00Z' },
  { id: 30, name: 'www.retailmax.com', createdAt: '2024-12-21T10:46:00Z' },
  { id: 31, name: 'm.retailmax.com', createdAt: '2024-12-21T10:47:00Z' },
  { id: 32, name: 'api.retailmax.com', createdAt: '2024-12-21T10:48:00Z' },
  { id: 33, name: 'cloudnine.host', createdAt: '2024-12-19T16:00:00Z' },
  { id: 34, name: 'panel.cloudnine.host', createdAt: '2024-12-19T16:01:00Z' },
  { id: 35, name: 'mediastream.tv', createdAt: '2024-12-18T09:30:00Z' },
  { id: 36, name: 'www.mediastream.tv', createdAt: '2024-12-18T09:31:00Z' },
  { id: 37, name: 'cdn.mediastream.tv', createdAt: '2024-12-18T09:32:00Z' },
  { id: 38, name: 'stream.mediastream.tv', createdAt: '2024-12-18T09:33:00Z' },
]

export function getMockSubdomains(params?: {
  page?: number
  pageSize?: number
  search?: string
  organizationId?: number
}): GetAllSubdomainsResponse {
  const page = params?.page || 1
  const pageSize = params?.pageSize || 10
  const search = params?.search?.toLowerCase() || ''

  let filtered = mockSubdomains

  if (search) {
    filtered = mockSubdomains.filter(sub =>
      sub.name.toLowerCase().includes(search)
    )
  }

  const total = filtered.length
  const totalPages = Math.ceil(total / pageSize)
  const start = (page - 1) * pageSize
  const domains = filtered.slice(start, start + pageSize)

  return {
    domains,
    total,
    page,
    pageSize,
    totalPages,
  }
}

export function getMockSubdomainById(id: number): Subdomain | undefined {
  return mockSubdomains.find(sub => sub.id === id)
}
