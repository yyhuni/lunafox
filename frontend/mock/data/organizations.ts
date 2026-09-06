import type { Organization, OrganizationsResponse } from '@/types/organization.types'

export const mockOrganizations: Organization[] = [
  {
    id: 1,
    name: 'Acme Corporation',
    description: 'Leading technology company focused on cloud computing and artificial intelligence',
    createdAt: '2024-01-15T08:30:00Z',
    updatedAt: '2024-12-28T14:20:00Z',
    targetCount: 12,
    domainCount: 156,
    endpointCount: 2341,
    targets: [
      { id: 1, name: 'acme.com' },
      { id: 2, name: 'acme.io' },
    ],
  },
  {
    id: 2,
    name: 'TechStart Inc',
    description: 'Innovative startup focused on SaaS product development',
    createdAt: '2024-02-20T10:15:00Z',
    updatedAt: '2024-12-27T09:45:00Z',
    targetCount: 5,
    domainCount: 78,
    endpointCount: 892,
    targets: [
      { id: 3, name: 'techstart.io' },
    ],
  },
  {
    id: 3,
    name: 'Global Finance Ltd',
    description: 'International financial services company providing banking and investment solutions',
    createdAt: '2024-03-10T14:00:00Z',
    updatedAt: '2024-12-26T16:30:00Z',
    targetCount: 8,
    domainCount: 234,
    endpointCount: 1567,
    targets: [
      { id: 4, name: 'globalfinance.com' },
      { id: 5, name: 'gf-bank.net' },
    ],
  },
  {
    id: 4,
    name: 'HealthCare Plus',
    description: 'Healthcare technology company focused on digital health solutions',
    createdAt: '2024-04-05T09:20:00Z',
    updatedAt: '2024-12-25T11:10:00Z',
    targetCount: 6,
    domainCount: 89,
    endpointCount: 723,
    targets: [
      { id: 6, name: 'healthcareplus.com' },
    ],
  },
  {
    id: 5,
    name: 'EduTech Solutions',
    description: 'Online education platform offering K-12 and professional training courses',
    createdAt: '2024-05-12T11:45:00Z',
    updatedAt: '2024-12-24T13:55:00Z',
    targetCount: 4,
    domainCount: 45,
    endpointCount: 456,
    targets: [
      { id: 7, name: 'edutech.io' },
    ],
  },
  {
    id: 6,
    name: 'RetailMax',
    description: 'E-commerce retail platform covering a wide range of product categories',
    createdAt: '2024-06-08T16:30:00Z',
    updatedAt: '2024-12-23T10:20:00Z',
    targetCount: 15,
    domainCount: 312,
    endpointCount: 4521,
    targets: [
      { id: 8, name: 'retailmax.com' },
      { id: 9, name: 'retailmax.cn' },
    ],
  },
  {
    id: 7,
    name: 'CloudNine Hosting',
    description: 'Cloud hosting provider offering VPS and dedicated servers',
    createdAt: '2024-07-20T08:00:00Z',
    updatedAt: '2024-12-22T15:40:00Z',
    targetCount: 3,
    domainCount: 67,
    endpointCount: 389,
    targets: [
      { id: 10, name: 'cloudnine.host' },
    ],
  },
  {
    id: 8,
    name: 'MediaStream Corp',
    description: 'Streaming media distribution platform providing video and audio services',
    createdAt: '2024-08-15T12:10:00Z',
    updatedAt: '2024-12-21T08:25:00Z',
    targetCount: 7,
    domainCount: 123,
    endpointCount: 1234,
    targets: [
      { id: 11, name: 'mediastream.tv' },
    ],
  },
  {
    id: 9,
    name: 'North America Enterprise Security Integration Office',
    description: 'Cross-regional security operations and M&A integration organization used to validate long-name truncation and pagination in the organization selector',
    createdAt: '2024-09-03T09:10:00Z',
    updatedAt: '2024-12-20T17:05:00Z',
    targetCount: 18,
    domainCount: 428,
    endpointCount: 6504,
    targets: [
      { id: 12, name: 'na-security.example.com' },
      { id: 13, name: 'integration.example.com' },
    ],
  },
  {
    id: 10,
    name: 'FinOps Risk Lab',
    description: 'Financial risk lab covering payment, clearing, and open-platform assets',
    createdAt: '2024-09-18T13:35:00Z',
    updatedAt: '2024-12-19T12:15:00Z',
    targetCount: 9,
    domainCount: 141,
    endpointCount: 1742,
    targets: [
      { id: 14, name: 'risklab.finance' },
    ],
  },
  {
    id: 11,
    name: 'Partner Cloud Exchange',
    description: 'Partner cloud exchange platform with a multi-tenant console and API gateway',
    createdAt: '2024-10-02T08:50:00Z',
    updatedAt: '2024-12-18T15:30:00Z',
    targetCount: 11,
    domainCount: 203,
    endpointCount: 2870,
    targets: [
      { id: 15, name: 'partners-cloud.net' },
    ],
  },
  {
    id: 12,
    name: 'Zero Trust Pilot Group',
    description: 'Zero-trust pilot organization used to validate low-asset counts and the edge around an empty description',
    createdAt: '2024-10-21T07:40:00Z',
    updatedAt: '2024-12-17T10:45:00Z',
    targetCount: 2,
    domainCount: 12,
    endpointCount: 64,
    targets: [
      { id: 16, name: 'ztna-pilot.internal' },
    ],
  },
]

type MockOrganizationListParams = {
  page?: number
  pageSize?: number
  search?: string
  filter?: string
}

// Keep mock organization search aligned with GET /v1/organizations, whose only supported filter field is displayName.
function getOrganizationNameSearch(filterOrSearch?: string) {
  const value = filterOrSearch?.trim() || ''
  if (!value) return ''

  const displayNameFilter = value.match(/^displayName="((?:\\.|[^"\\])*)"$/)
  if (!displayNameFilter) return value

  return displayNameFilter[1].replace(/\\"/g, '"').replace(/\\\\/g, '\\')
}

export function filterMockOrganizations(organizations: Organization[], filterOrSearch?: string) {
  const search = getOrganizationNameSearch(filterOrSearch).toLowerCase()
  if (!search) return organizations

  return organizations.filter((organization) => organization.name.toLowerCase().includes(search))
}

export function getMockOrganizations(params?: MockOrganizationListParams): OrganizationsResponse<Organization> {
  const page = params?.page || 1
  const pageSize = params?.pageSize || 10
  const filtered = filterMockOrganizations(mockOrganizations, params?.filter ?? params?.search)

  // Pagination
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

export function deleteMockOrganization(id: number): Organization | undefined {
  const index = mockOrganizations.findIndex(org => org.id === id)
  if (index === -1) {
    return undefined
  }

  const [deleted] = mockOrganizations.splice(index, 1)
  return deleted
}

export function bulkDeleteMockOrganizations(ids: number[]) {
  let deletedCount = 0

  for (const id of ids) {
    if (deleteMockOrganization(id)) {
      deletedCount += 1
    }
  }

  return {
    deletedCount,
  }
}
