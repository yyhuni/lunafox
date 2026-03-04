import type { ScanRecord, GetScansResponse, ScanStatus } from '@/types/scan.types'
import type { ScanStatistics } from '@/services/scan.service'

export const mockScans: ScanRecord[] = [
  {
    id: 1,
    targetId: 1,
    target: { id: 1, name: 'acme.com', type: 'domain' },
    workerName: 'worker-01',
    cachedStats: {
      subdomainsCount: 156,
      websitesCount: 89,
      directoriesCount: 234,
      endpointsCount: 2341,
      ipsCount: 45,
      screenshotsCount: 50,
      vulnsTotal: 23,
      vulnsCritical: 1,
      vulnsHigh: 4,
      vulnsMedium: 8,
      vulnsLow: 10,
    },
    workflowIds: [1, 2, 3],
    workflowNames: ['Subdomain Discovery', 'Web Crawling', 'Nuclei Scanner'],
    scanMode: 'full',
    createdAt: '2024-12-28T10:00:00Z',
    status: 'completed',
    progress: 100,
  },
  {
    id: 2,
    targetId: 2,
    target: { id: 2, name: 'acme.io', type: 'domain' },
    workerName: 'worker-02',
    cachedStats: {
      subdomainsCount: 78,
      websitesCount: 45,
      directoriesCount: 123,
      endpointsCount: 892,
      ipsCount: 23,
      screenshotsCount: 30,
      vulnsTotal: 12,
      vulnsCritical: 0,
      vulnsHigh: 2,
      vulnsMedium: 5,
      vulnsLow: 5,
    },
    workflowIds: [1, 2],
    workflowNames: ['Subdomain Discovery', 'Web Crawling'],
    scanMode: 'full',
    createdAt: '2024-12-27T14:30:00Z',
    status: 'running',
    progress: 65,
    currentStage: 'web_crawling',
    stageProgress: {
      subdomain_discovery: {
        status: 'completed',
        order: 0,
        startedAt: '2024-12-27T14:30:00Z',
        duration: 1200,
        detail: 'Found 78 subdomains',
      },
      web_crawling: {
        status: 'running',
        order: 1,
        startedAt: '2024-12-27T14:50:00Z',
      },
    },
  },
  {
    id: 3,
    targetId: 3,
    target: { id: 3, name: 'techstart.io', type: 'domain' },
    workerName: 'worker-01',
    cachedStats: {
      subdomainsCount: 45,
      websitesCount: 28,
      directoriesCount: 89,
      endpointsCount: 567,
      ipsCount: 12,
      screenshotsCount: 20,
      vulnsTotal: 8,
      vulnsCritical: 0,
      vulnsHigh: 1,
      vulnsMedium: 3,
      vulnsLow: 4,
    },
    workflowIds: [1, 2, 3],
    workflowNames: ['Subdomain Discovery', 'Web Crawling', 'Nuclei Scanner'],
    scanMode: 'full',
    createdAt: '2024-12-26T08:45:00Z',
    status: 'completed',
    progress: 100,
  },
  {
    id: 4,
    targetId: 4,
    target: { id: 4, name: 'globalfinance.test.globalfinance.globalfinance.globalfinance.com', type: 'domain' },
    workerName: 'worker-03',
    cachedStats: {
      subdomainsCount: 0,
      websitesCount: 0,
      directoriesCount: 0,
      endpointsCount: 0,
      ipsCount: 0,
      screenshotsCount: 0,
      vulnsTotal: 0,
      vulnsCritical: 0,
      vulnsHigh: 0,
      vulnsMedium: 0,
      vulnsLow: 0,
    },
    workflowIds: [1],
    workflowNames: ['Subdomain Discovery'],
    scanMode: 'full',
    createdAt: '2024-12-25T16:20:00Z',
    status: 'failed',
    progress: 15,
    errorMessage: 'Connection timeout: Unable to reach target',
  },
  {
    id: 5,
    targetId: 6,
    target: { id: 6, name: 'healthcareplus.com', type: 'domain' },
    workerName: 'worker-02',
    cachedStats: {
      subdomainsCount: 34,
      websitesCount: 0,
      directoriesCount: 0,
      endpointsCount: 0,
      ipsCount: 8,
      screenshotsCount: 0,
      vulnsTotal: 0,
      vulnsCritical: 0,
      vulnsHigh: 0,
      vulnsMedium: 0,
      vulnsLow: 0,
    },
    workflowIds: [1, 2, 3],
    workflowNames: ['Subdomain Discovery', 'Web Crawling', 'Nuclei Scanner'],
    scanMode: 'full',
    createdAt: '2024-12-29T09:00:00Z',
    status: 'running',
    progress: 25,
    currentStage: 'subdomain_discovery',
    stageProgress: {
      subdomain_discovery: {
        status: 'running',
        order: 0,
        startedAt: '2024-12-29T09:00:00Z',
      },
      web_crawling: {
        status: 'pending',
        order: 1,
      },
      nuclei_scan: {
        status: 'pending',
        order: 2,
      },
    },
  },
  {
    id: 6,
    targetId: 7,
    target: { id: 7, name: 'edutech.io', type: 'domain' },
    workerName: null,
    cachedStats: {
      subdomainsCount: 0,
      websitesCount: 0,
      directoriesCount: 0,
      endpointsCount: 0,
      ipsCount: 0,
      screenshotsCount: 0,
      vulnsTotal: 0,
      vulnsCritical: 0,
      vulnsHigh: 0,
      vulnsMedium: 0,
      vulnsLow: 0,
    },
    workflowIds: [1, 2],
    workflowNames: ['Subdomain Discovery', 'Web Crawling'],
    scanMode: 'full',
    createdAt: '2024-12-29T10:30:00Z',
    status: 'pending',
    progress: 0,
  },
  {
    id: 7,
    targetId: 8,
    target: { id: 8, name: 'retailmax.com', type: 'domain' },
    workerName: 'worker-01',
    cachedStats: {
      subdomainsCount: 89,
      websitesCount: 56,
      directoriesCount: 178,
      endpointsCount: 1234,
      ipsCount: 28,
      screenshotsCount: 40,
      vulnsTotal: 15,
      vulnsCritical: 0,
      vulnsHigh: 3,
      vulnsMedium: 6,
      vulnsLow: 6,
    },
    workflowIds: [1, 2, 3],
    workflowNames: ['Subdomain Discovery', 'Web Crawling', 'Nuclei Scanner'],
    scanMode: 'full',
    createdAt: '2024-12-21T10:45:00Z',
    status: 'completed',
    progress: 100,
  },
  {
    id: 8,
    targetId: 11,
    target: { id: 11, name: 'mediastream.tv', type: 'domain' },
    workerName: 'worker-02',
    cachedStats: {
      subdomainsCount: 67,
      websitesCount: 0,
      directoriesCount: 0,
      endpointsCount: 0,
      ipsCount: 15,
      screenshotsCount: 0,
      vulnsTotal: 0,
      vulnsCritical: 0,
      vulnsHigh: 0,
      vulnsMedium: 0,
      vulnsLow: 0,
    },
    workflowIds: [1, 2, 3],
    workflowNames: ['Subdomain Discovery', 'Web Crawling', 'Nuclei Scanner'],
    scanMode: 'full',
    createdAt: '2024-12-29T08:00:00Z',
    status: 'running',
    progress: 45,
    currentStage: 'web_crawling',
    stageProgress: {
      subdomain_discovery: {
        status: 'completed',
        order: 0,
        startedAt: '2024-12-29T08:00:00Z',
        duration: 900,
        detail: 'Found 67 subdomains',
      },
      web_crawling: {
        status: 'running',
        order: 1,
        startedAt: '2024-12-29T08:15:00Z',
      },
      nuclei_scan: {
        status: 'pending',
        order: 2,
      },
    },
  },
]

export const mockScanStatistics: ScanStatistics = {
  total: 156,
  pending: 1,
  running: 3,
  completed: 139,
  failed: 11,
  cancelled: 2,
  totalVulns: 89,
  totalSubdomains: 4823,
  totalEndpoints: 12456,
  totalWebsites: 3421,
  totalAssets: 21638,
}

export function getMockScans(params?: {
  page?: number
  pageSize?: number
  status?: ScanStatus
  search?: string
}): GetScansResponse {
  const page = params?.page || 1
  const pageSize = params?.pageSize || 10
  const status = params?.status
  const search = params?.search?.toLowerCase() || ''

  let filtered = mockScans

  if (status) {
    filtered = filtered.filter(scan => scan.status === status)
  }

  if (search) {
    filtered = filtered.filter(scan =>
      scan.target?.name?.toLowerCase().includes(search)
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

export function getMockScanById(id: number): ScanRecord | undefined {
  return mockScans.find(scan => scan.id === id)
}
