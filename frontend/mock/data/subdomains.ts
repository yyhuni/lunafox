import type { Subdomain, GetAllSubdomainsResponse } from '@/types/subdomain.types'

const mockSubdomainRows: Array<Omit<Subdomain, 'name'>> = [
  { id: 1, dnsName: 'acme.com', createdAt: '2024-12-28T10:00:00Z' },
  { id: 2, dnsName: 'www.acme.com', createdAt: '2024-12-28T10:01:00Z' },
  { id: 3, dnsName: 'api.acme.com', createdAt: '2024-12-28T10:02:00Z' },
  { id: 4, dnsName: 'admin.acme.com', createdAt: '2024-12-28T10:03:00Z' },
  { id: 5, dnsName: 'mail.acme.com', createdAt: '2024-12-28T10:04:00Z' },
  { id: 6, dnsName: 'blog.acme.com', createdAt: '2024-12-28T10:05:00Z' },
  { id: 7, dnsName: 'shop.acme.com', createdAt: '2024-12-28T10:06:00Z' },
  { id: 8, dnsName: 'cdn.acme.com', createdAt: '2024-12-28T10:07:00Z' },
  { id: 9, dnsName: 'static.acme.com', createdAt: '2024-12-28T10:08:00Z' },
  { id: 10, dnsName: 'dev.acme.com', createdAt: '2024-12-28T10:09:00Z' },
  { id: 11, dnsName: 'staging.acme.com', createdAt: '2024-12-28T10:10:00Z' },
  { id: 12, dnsName: 'test.acme.com', createdAt: '2024-12-28T10:11:00Z' },
  { id: 13, dnsName: 'acme.io', createdAt: '2024-12-27T14:30:00Z' },
  { id: 14, dnsName: 'docs.acme.io', createdAt: '2024-12-27T14:31:00Z' },
  { id: 15, dnsName: 'api.acme.io', createdAt: '2024-12-27T14:32:00Z' },
  { id: 16, dnsName: 'status.acme.io', createdAt: '2024-12-27T14:33:00Z' },
  { id: 17, dnsName: 'techstart.io', createdAt: '2024-12-26T08:45:00Z' },
  { id: 18, dnsName: 'www.techstart.io', createdAt: '2024-12-26T08:46:00Z' },
  { id: 19, dnsName: 'app.techstart.io', createdAt: '2024-12-26T08:47:00Z' },
  { id: 20, dnsName: 'globalfinance.com', createdAt: '2024-12-25T16:20:00Z' },
  { id: 21, dnsName: 'www.globalfinance.com', createdAt: '2024-12-25T16:21:00Z' },
  { id: 22, dnsName: 'secure.globalfinance.com', createdAt: '2024-12-25T16:22:00Z' },
  { id: 23, dnsName: 'portal.globalfinance.com', createdAt: '2024-12-25T16:23:00Z' },
  { id: 24, dnsName: 'healthcareplus.com', createdAt: '2024-12-23T11:00:00Z' },
  { id: 25, dnsName: 'www.healthcareplus.com', createdAt: '2024-12-23T11:01:00Z' },
  { id: 26, dnsName: 'patient.healthcareplus.com', createdAt: '2024-12-23T11:02:00Z' },
  { id: 27, dnsName: 'edutech.io', createdAt: '2024-12-22T13:30:00Z' },
  { id: 28, dnsName: 'learn.edutech.io', createdAt: '2024-12-22T13:31:00Z' },
  { id: 29, dnsName: 'retailmax.com', createdAt: '2024-12-21T10:45:00Z' },
  { id: 30, dnsName: 'www.retailmax.com', createdAt: '2024-12-21T10:46:00Z' },
  { id: 31, dnsName: 'm.retailmax.com', createdAt: '2024-12-21T10:47:00Z' },
  { id: 32, dnsName: 'api.retailmax.com', createdAt: '2024-12-21T10:48:00Z' },
  { id: 33, dnsName: 'cloudnine.host', createdAt: '2024-12-19T16:00:00Z' },
  { id: 34, dnsName: 'panel.cloudnine.host', createdAt: '2024-12-19T16:01:00Z' },
  { id: 35, dnsName: 'mediastream.tv', createdAt: '2024-12-18T09:30:00Z' },
  { id: 36, dnsName: 'www.mediastream.tv', createdAt: '2024-12-18T09:31:00Z' },
  { id: 37, dnsName: 'cdn.mediastream.tv', createdAt: '2024-12-18T09:32:00Z' },
  { id: 38, dnsName: 'stream.mediastream.tv', createdAt: '2024-12-18T09:33:00Z' },
]

export const mockSubdomains: Subdomain[] = mockSubdomainRows.map((subdomain) => ({
  ...subdomain,
  name: subdomain.dnsName,
}))

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
      sub.dnsName.toLowerCase().includes(search)
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

export function deleteMockSubdomain(id: number): Subdomain | undefined {
  const index = mockSubdomains.findIndex(subdomain => subdomain.id === id)
  if (index === -1) {
    return undefined
  }

  const [deleted] = mockSubdomains.splice(index, 1)
  return deleted
}

export function bulkDeleteMockSubdomains(ids: number[]) {
  let deletedCount = 0

  for (const id of ids) {
    if (deleteMockSubdomain(id)) {
      deletedCount += 1
    }
  }

  return {
    message: "Deleted subdomains",
    deletedCount,
    requestedIds: ids,
    cascadeDeleted: {},
  }
}

export function bulkRemoveMockOrganizationSubdomains(ids: number[]) {
  return {
    message: "Removed subdomains from organization",
    successCount: ids.length,
    failedCount: 0,
  }
}
