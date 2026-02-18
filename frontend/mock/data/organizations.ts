import type { Organization, OrganizationsResponse } from '@/types/organization.types'

export const mockOrganizations: Organization[] = [
  {
    id: 1,
    name: 'Acme Corporation',
    description: '全球领先的科技公司，专注于云计算和人工智能领域',
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
    description: '创新型初创企业，主营 SaaS 产品开发',
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
    description: '国际金融服务公司，提供银行和投资解决方案',
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
    description: '医疗健康科技公司，专注于数字化医疗解决方案',
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
    description: '在线教育平台，提供 K-12 和职业培训课程',
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
    description: '电子商务零售平台，覆盖多品类商品销售',
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
    description: '云托管服务提供商，提供 VPS 和专用服务器',
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
    description: '流媒体内容分发平台，提供视频和音频服务',
    createdAt: '2024-08-15T12:10:00Z',
    updatedAt: '2024-12-21T08:25:00Z',
    targetCount: 7,
    domainCount: 123,
    endpointCount: 1234,
    targets: [
      { id: 11, name: 'mediastream.tv' },
    ],
  },
]

export function getMockOrganizations(params?: {
  page?: number
  pageSize?: number
  search?: string
}): OrganizationsResponse<Organization> {
  const page = params?.page || 1
  const pageSize = params?.pageSize || 10
  const search = params?.search?.toLowerCase() || ''

  // Filter search
  let filtered = mockOrganizations
  if (search) {
    filtered = mockOrganizations.filter(
      org =>
        org.name.toLowerCase().includes(search) ||
        org.description.toLowerCase().includes(search)
    )
  }

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
