import type { Target, TargetsResponse, TargetDetail } from '@/types/target.types'
import { mockDirectories } from './directories'
import { mockEndpoints } from './endpoints'
import { mockIPAddresses } from './ip-addresses'
import { mockScreenshots } from './screenshots'
import { mockSubdomains } from './subdomains'
import { mockVulnerabilities } from './vulnerabilities'
import { mockWebsites } from './websites'

export const mockTargets: Target[] = [
  {
    id: 1,
    name: 'acme.com',
    type: 'domain',
    description: 'Acme Corporation main site',
    createdAt: '2024-01-15T08:30:00Z',
    lastScannedAt: '2024-12-28T10:00:00Z',
    organizations: [
      { id: 1, name: 'Acme Corporation' },
      { id: 9, name: 'Acme Global Shared Services Platform' },
    ],
  },
  {
    id: 2,
    name: 'acme.io',
    type: 'domain',
    description: 'Acme Corporation developer platform',
    createdAt: '2024-01-16T09:00:00Z',
    lastScannedAt: '2024-12-27T14:30:00Z',
    organizations: [
      { id: 1, name: 'Acme Corporation' },
      { id: 10, name: 'Acme Developer Experience and Partner Enablement' },
    ],
  },
  {
    id: 3,
    name: 'techstart-enterprise-platform.io',
    type: 'domain',
    description: 'TechStart main website',
    createdAt: '2024-02-20T10:15:00Z',
    lastScannedAt: '2024-12-26T08:45:00Z',
    organizations: [
      { id: 2, name: 'TechStart Inc' },
      { id: 11, name: 'TechStart Venture Studio Holdings' },
      { id: 12, name: 'TechStart International Accelerator Network' },
    ],
  },
  {
    id: 4,
    name: 'globalfinance.com',
    type: 'domain',
    description: 'Global Finance main site',
    createdAt: '2024-03-10T14:00:00Z',
    lastScannedAt: '2024-12-25T16:20:00Z',
    organizations: [{ id: 3, name: 'Global Finance Ltd' }],
  },
  {
    id: 5,
    name: '192.168.1.0/24',
    type: 'cidr',
    description: 'Internal IP range',
    createdAt: '2024-03-15T11:30:00Z',
    lastScannedAt: '2024-12-24T09:15:00Z',
    organizations: [{ id: 3, name: 'Global Finance Ltd' }],
  },
  {
    id: 6,
    name: 'healthcareplus-member-portal.com',
    type: 'domain',
    description: 'HealthCare Plus main website',
    createdAt: '2024-04-05T09:20:00Z',
    lastScannedAt: '2024-12-23T11:00:00Z',
    organizations: [
      { id: 4, name: 'HealthCare Plus' },
      { id: 13, name: 'HealthCare Plus Clinical Operations Group' },
    ],
  },
  {
    id: 7,
    name: 'edutech.io',
    type: 'domain',
    description: 'EduTech online education platform',
    createdAt: '2024-05-12T11:45:00Z',
    organizations: [{ id: 5, name: 'EduTech Solutions International Group' }],
  },
  {
    id: 8,
    name: 'retailmax.com',
    type: 'domain',
    description: 'RetailMax e-commerce site',
    createdAt: '2024-06-08T16:30:00Z',
    lastScannedAt: '2024-12-21T10:45:00Z',
    organizations: [{ id: 6, name: 'RetailMax' }],
  },
  {
    id: 9,
    name: '10.0.0.1',
    type: 'ip',
    description: 'Core server IP',
    createdAt: '2024-07-01T08:00:00Z',
    lastScannedAt: '2024-12-20T14:20:00Z',
    organizations: [
      { id: 7, name: 'CloudNine Hosting' },
      { id: 14, name: 'CloudNine Managed Infrastructure and Edge Operations' },
    ],
  },
  {
    id: 10,
    name: 'cloudnine.host',
    type: 'domain',
    description: 'CloudNine hosting service',
    createdAt: '2024-07-20T08:00:00Z',
    lastScannedAt: '2024-12-19T16:00:00Z',
    organizations: [{ id: 7, name: 'CloudNine Hosting' }],
  },
  {
    id: 11,
    name: 'mediastream.tv',
    type: 'domain',
    description: 'MediaStream streaming platform',
    createdAt: '2024-08-15T12:10:00Z',
    lastScannedAt: '2024-12-18T09:30:00Z',
    organizations: [{ id: 8, name: 'MediaStream Corp' }],
  },
  {
    id: 12,
    name: 'api.acme.com',
    type: 'domain',
    description: 'Acme API service',
    createdAt: '2024-09-01T10:00:00Z',
    lastScannedAt: '2024-12-17T11:15:00Z',
    organizations: [{ id: 1, name: 'Acme Corporation' }],
  },
]

export const mockTargetDetails: Record<number, TargetDetail> = {
  1: {
    ...mockTargets[0],
    summary: {
      subdomains: 156,
      websites: 89,
      endpoints: 2341,
      ips: 45,
      directories: 67,
      screenshots: 34,
      vulnerabilities: {
        total: 23,
        critical: 1,
        high: 4,
        medium: 8,
        low: 10,
      },
    },
  },
  2: {
    ...mockTargets[1],
    summary: {
      subdomains: 78,
      websites: 45,
      endpoints: 892,
      ips: 23,
      directories: 32,
      screenshots: 18,
      vulnerabilities: {
        total: 12,
        critical: 0,
        high: 2,
        medium: 5,
        low: 5,
      },
    },
  },
}

function countMockWebsitesForTarget(targetId: number) {
  return mockWebsites.filter(website => website.target === targetId).length
}

export function getMockTargets(params?: {
  page?: number
  pageSize?: number
  search?: string
  orderBy?: string | null
}): TargetsResponse {
  const page = params?.page || 1
  const pageSize = params?.pageSize || 10
  const search = params?.search?.toLowerCase() || ''

  let filtered = mockTargets
  if (search) {
    filtered = mockTargets.filter(
      target =>
        target.name.toLowerCase().includes(search) ||
        target.description?.toLowerCase().includes(search)
    )
  }

  filtered = applyMockTargetOrder(filtered, params?.orderBy || '')

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
    ...(nextPage && { nextPageToken: `mock-target-page-${nextPage}` }),
  }
}

function applyMockTargetOrder(targets: Target[], orderBy: string) {
  const normalized = orderBy.trim()
  if (!normalized) return targets

  const [field, direction = 'asc'] = normalized.split(/\s+/)
  const desc = direction.toLowerCase() === 'desc'
  const sorted = [...targets]

  sorted.sort((left, right) => {
    const primary = compareMockTargetField(left, right, field, desc)
    return primary === 0 ? left.id - right.id : primary
  })

  return sorted
}

function compareMockTargetField(left: Target, right: Target, field: string, desc: boolean) {
  switch (field) {
    case 'displayName':
    case 'name': {
      const result = left.name.localeCompare(right.name)
      return desc ? -result : result
    }
    case 'lastScannedAt':
      return compareOptionalTargetTime(left.lastScannedAt, right.lastScannedAt, desc)
    case 'createdAt':
    default:
      return compareOptionalTargetTime(left.createdAt, right.createdAt, desc)
  }
}

function compareOptionalTargetTime(leftValue: string | undefined, rightValue: string | undefined, desc: boolean) {
  if (!leftValue && !rightValue) return 0
  if (!leftValue) return 1
  if (!rightValue) return -1

  const result = new Date(leftValue).getTime() - new Date(rightValue).getTime()
  return desc ? -result : result
}

export function getMockTargetById(id: number): TargetDetail | undefined {
  if (mockTargetDetails[id]) {
    const targetDetail = mockTargetDetails[id]
    return {
      ...targetDetail,
      summary: {
        ...targetDetail.summary,
        websites: countMockWebsitesForTarget(id),
      },
    }
  }
  const target = mockTargets.find(t => t.id === id)
  if (target) {
    const targetName = target.name.toLowerCase()
    const targetVulnerabilities = mockVulnerabilities.filter(vulnerability => vulnerability.target === target.id)

    return {
      ...target,
      summary: {
        subdomains: mockSubdomains.filter(subdomain => subdomain.name.toLowerCase().includes(targetName)).length,
        websites: countMockWebsitesForTarget(target.id),
        endpoints: mockEndpoints.filter(endpoint =>
          endpoint.url.toLowerCase().includes(targetName) ||
          endpoint.host?.toLowerCase().includes(targetName)
        ).length,
        ips: mockIPAddresses.filter(ipAddress =>
          ipAddress.hosts.some(host => host.toLowerCase().includes(targetName))
        ).length,
        directories: mockDirectories.filter(directory =>
          directory.url.toLowerCase().includes(targetName)
        ).length,
        screenshots: mockScreenshots.length,
        vulnerabilities: {
          total: targetVulnerabilities.length,
          critical: targetVulnerabilities.filter(vulnerability => vulnerability.severity === 'critical').length,
          high: targetVulnerabilities.filter(vulnerability => vulnerability.severity === 'high').length,
          medium: targetVulnerabilities.filter(vulnerability => vulnerability.severity === 'medium').length,
          low: targetVulnerabilities.filter(vulnerability => vulnerability.severity === 'low').length,
        },
      },
    }
  }
  return undefined
}

export function deleteMockTarget(id: number): boolean {
  const index = mockTargets.findIndex(target => target.id === id)
  if (index === -1) {
    return false
  }

  mockTargets.splice(index, 1)
  delete mockTargetDetails[id]
  return true
}

export function bulkDeleteMockTargets(ids: number[]) {
  let deletedCount = 0

  for (const id of ids) {
    if (deleteMockTarget(id)) {
      deletedCount += 1
    }
  }

  return {
    deletedCount,
  }
}
