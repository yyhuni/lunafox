import type { FilterOptionsResponse, IPAddress, GetIPAddressesResponse } from '@/types/ip-address.types'

// Use function to generate IP address
const ip = (a: number, b: number, c: number, d: number) => `${a}.${b}.${c}.${d}`

export const mockIPAddresses: IPAddress[] = [
  {
    ip: ip(192, 0, 2, 1),
    hosts: ['router.local', 'gateway.lan'],
    ports: [80, 443, 22, 53],
    createdAt: '2024-12-28T10:00:00Z',
  },
  {
    ip: ip(192, 0, 2, 10),
    hosts: ['api.acme.com', 'backend.acme.com'],
    ports: [80, 443, 8080, 3306],
    createdAt: '2024-12-28T10:01:00Z',
  },
  {
    ip: ip(192, 0, 2, 12),
    hosts: ['api.acme.com.evil'],
    ports: [443],
    createdAt: '2024-12-28T10:01:30Z',
  },
  {
    ip: ip(192, 0, 2, 11),
    hosts: ['web.acme.com', 'www.acme.com'],
    ports: [80, 443],
    createdAt: '2024-12-28T10:02:00Z',
  },
  {
    ip: ip(198, 51, 100, 50),
    hosts: ['db.internal.acme.com'],
    ports: [3306, 5432, 27017],
    createdAt: '2024-12-28T10:03:00Z',
  },
  {
    ip: ip(203, 0, 113, 50),
    hosts: ['cdn.acme.com'],
    ports: [80, 443],
    createdAt: '2024-12-28T10:04:00Z',
  },
  {
    ip: ip(198, 51, 100, 10),
    hosts: ['mail.acme.com', 'smtp.acme.com'],
    ports: [25, 465, 587, 993, 995],
    createdAt: '2024-12-28T10:05:00Z',
  },
  {
    ip: ip(192, 0, 2, 100),
    hosts: ['jenkins.acme.com'],
    ports: [8080, 50000],
    createdAt: '2024-12-28T10:06:00Z',
  },
  {
    ip: ip(192, 0, 2, 101),
    hosts: ['gitlab.acme.com'],
    ports: [80, 443, 22],
    createdAt: '2024-12-28T10:07:00Z',
  },
  {
    ip: ip(192, 0, 2, 102),
    hosts: ['k8s.acme.com', 'kubernetes.acme.com'],
    ports: [6443, 10250, 10251, 10252],
    createdAt: '2024-12-28T10:08:00Z',
  },
  {
    ip: ip(192, 0, 2, 103),
    hosts: ['elastic.acme.com'],
    ports: [9200, 9300, 5601],
    createdAt: '2024-12-28T10:09:00Z',
  },
  {
    ip: ip(192, 0, 2, 104),
    hosts: ['redis.acme.com'],
    ports: [6379],
    createdAt: '2024-12-28T10:10:00Z',
  },
  {
    ip: ip(192, 0, 2, 105),
    hosts: ['mq.acme.com', 'rabbitmq.acme.com'],
    ports: [5672, 15672],
    createdAt: '2024-12-28T10:11:00Z',
  },
]

const SMART_FILTER_CLAUSE_PATTERN = /^(\w+)(==|!=|=)"([^"]*)"$/

function ipAddressMatchesFreeText(ipAddr: IPAddress, filter: string): boolean {
  const value = filter.toLowerCase()

  return (
    ipAddr.ip.toLowerCase().includes(value) ||
    ipAddr.hosts.some((host) => host.toLowerCase().includes(value)) ||
    ipAddr.ports.some((port) => String(port) === value)
  )
}

function ipAddressMatchesClause(ipAddr: IPAddress, rawClause: string): boolean {
  const match = rawClause.trim().match(SMART_FILTER_CLAUSE_PATTERN)
  if (!match) return ipAddressMatchesFreeText(ipAddr, rawClause)

  const [, field, operator, rawValue] = match
  const value = rawValue.toLowerCase()
  let matched = false

  if (field === 'ip') {
    matched = ipAddr.ip.toLowerCase().includes(value)
  } else if (field === 'host') {
    matched = ipAddr.hosts.some((host) => host.toLowerCase().includes(value))
  } else if (field === 'port') {
    matched = ipAddr.ports.some((port) => String(port) === value)
  } else {
    matched = ipAddressMatchesFreeText(ipAddr, value)
  }

  return operator === '!=' ? !matched : matched
}

export function mockIPAddressMatchesFilter(ipAddr: IPAddress, rawFilter?: string): boolean {
  const filter = rawFilter?.trim()
  if (!filter) return true

  return filter.split(/\s+\|\|\s+/).some((orGroup) =>
    orGroup.split(/\s+&&\s+/).every((clause) => ipAddressMatchesClause(ipAddr, clause))
  )
}

export function getMockIPAddresses(params?: {
  page?: number
  pageSize?: number
  filter?: string
  targetId?: number
  scanId?: number
}): GetIPAddressesResponse {
  const page = params?.page || 1
  const pageSize = params?.pageSize || 10
  const filter = params?.filter || ''

  let filtered = mockIPAddresses

  if (filter) {
    filtered = filtered.filter((ipAddr) => mockIPAddressMatchesFilter(ipAddr, filter))
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

export function getMockIPAddressByIP(ipStr: string): IPAddress | undefined {
  return mockIPAddresses.find(addr => addr.ip === ipStr)
}

export function getMockPortOptions(params?: { domain?: string }): FilterOptionsResponse {
  const domain = params?.domain?.toLowerCase()
  const scoped = domain
    ? mockIPAddresses.filter((ipAddr) => ipAddr.hosts.some((host) => host.toLowerCase().includes(domain)))
    : mockIPAddresses

  const counts = new Map<number, number>()
  for (const ipAddr of scoped) {
    for (const port of new Set(ipAddr.ports)) {
      counts.set(port, (counts.get(port) ?? 0) + 1)
    }
  }

  return {
    results: Array.from(counts.entries())
      .sort(([left], [right]) => left - right)
      .map(([port, count]) => ({ value: String(port), label: String(port), count })),
  }
}

export function bulkDeleteMockIPAddresses(ips: string[]) {
  const ipSet = new Set(ips)
  const originalCount = mockIPAddresses.length

  for (let index = mockIPAddresses.length - 1; index >= 0; index -= 1) {
    if (ipSet.has(mockIPAddresses[index].ip)) {
      mockIPAddresses.splice(index, 1)
    }
  }

  return {
    deletedCount: originalCount - mockIPAddresses.length,
  }
}
