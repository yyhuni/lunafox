import type { Target, TargetsResponse, TargetDetail } from '@/types/target.types'

export const mockTargets: Target[] = [
  {
    id: 1,
    name: 'acme.com',
    type: 'domain',
    description: 'Acme Corporation 主站',
    createdAt: '2024-01-15T08:30:00Z',
    lastScannedAt: '2024-12-28T10:00:00Z',
    organizations: [{ id: 1, name: 'Acme Corporation' }],
  },
  {
    id: 2,
    name: 'acme.io',
    type: 'domain',
    description: 'Acme Corporation 开发者平台',
    createdAt: '2024-01-16T09:00:00Z',
    lastScannedAt: '2024-12-27T14:30:00Z',
    organizations: [{ id: 1, name: 'Acme Corporation' }],
  },
  {
    id: 3,
    name: 'techstart.io',
    type: 'domain',
    description: 'TechStart 官网',
    createdAt: '2024-02-20T10:15:00Z',
    lastScannedAt: '2024-12-26T08:45:00Z',
    organizations: [{ id: 2, name: 'TechStart Inc' }],
  },
  {
    id: 4,
    name: 'globalfinance.com',
    type: 'domain',
    description: 'Global Finance 主站',
    createdAt: '2024-03-10T14:00:00Z',
    lastScannedAt: '2024-12-25T16:20:00Z',
    organizations: [{ id: 3, name: 'Global Finance Ltd' }],
  },
  {
    id: 5,
    name: '192.168.1.0/24',
    type: 'cidr',
    description: '内网 IP 段',
    createdAt: '2024-03-15T11:30:00Z',
    lastScannedAt: '2024-12-24T09:15:00Z',
    organizations: [{ id: 3, name: 'Global Finance Ltd' }],
  },
  {
    id: 6,
    name: 'healthcareplus.com',
    type: 'domain',
    description: 'HealthCare Plus 官网',
    createdAt: '2024-04-05T09:20:00Z',
    lastScannedAt: '2024-12-23T11:00:00Z',
    organizations: [{ id: 4, name: 'HealthCare Plus' }],
  },
  {
    id: 7,
    name: 'edutech.io',
    type: 'domain',
    description: 'EduTech 在线教育平台',
    createdAt: '2024-05-12T11:45:00Z',
    lastScannedAt: '2024-12-22T13:30:00Z',
    organizations: [{ id: 5, name: 'EduTech Solutions' }],
  },
  {
    id: 8,
    name: 'retailmax.com',
    type: 'domain',
    description: 'RetailMax 电商主站',
    createdAt: '2024-06-08T16:30:00Z',
    lastScannedAt: '2024-12-21T10:45:00Z',
    organizations: [{ id: 6, name: 'RetailMax' }],
  },
  {
    id: 9,
    name: '10.0.0.1',
    type: 'ip',
    description: '核心服务器 IP',
    createdAt: '2024-07-01T08:00:00Z',
    lastScannedAt: '2024-12-20T14:20:00Z',
    organizations: [{ id: 7, name: 'CloudNine Hosting' }],
  },
  {
    id: 10,
    name: 'cloudnine.host',
    type: 'domain',
    description: 'CloudNine 托管服务',
    createdAt: '2024-07-20T08:00:00Z',
    lastScannedAt: '2024-12-19T16:00:00Z',
    organizations: [{ id: 7, name: 'CloudNine Hosting' }],
  },
  {
    id: 11,
    name: 'mediastream.tv',
    type: 'domain',
    description: 'MediaStream 流媒体平台',
    createdAt: '2024-08-15T12:10:00Z',
    lastScannedAt: '2024-12-18T09:30:00Z',
    organizations: [{ id: 8, name: 'MediaStream Corp' }],
  },
  {
    id: 12,
    name: 'api.acme.com',
    type: 'domain',
    description: 'Acme API 服务',
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

export function getMockTargets(params?: {
  page?: number
  pageSize?: number
  search?: string
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

  const total = filtered.length
  const totalPages = Math.ceil(total / pageSize)
  const start = (page - 1) * pageSize
  const results = filtered.slice(start, start + pageSize)

  return {
    results,
    total,
    page,
    pageSize,
    totalPages,
  }
}

export function getMockTargetById(id: number): TargetDetail | undefined {
  if (mockTargetDetails[id]) {
    return mockTargetDetails[id]
  }
  const target = mockTargets.find(t => t.id === id)
  if (target) {
    return {
      ...target,
      summary: {
        subdomains: Math.floor(Math.random() * 100) + 10,
        websites: Math.floor(Math.random() * 50) + 5,
        endpoints: Math.floor(Math.random() * 1000) + 100,
        ips: Math.floor(Math.random() * 30) + 5,
        directories: Math.floor(Math.random() * 50) + 10,
        screenshots: Math.floor(Math.random() * 30) + 5,
        vulnerabilities: {
          total: Math.floor(Math.random() * 20) + 1,
          critical: Math.floor(Math.random() * 2),
          high: Math.floor(Math.random() * 5),
          medium: Math.floor(Math.random() * 8),
          low: Math.floor(Math.random() * 10),
        },
      },
    }
  }
  return undefined
}
