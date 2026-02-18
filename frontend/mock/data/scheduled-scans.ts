import type { ScheduledScan, GetScheduledScansResponse } from '@/types/scheduled-scan.types'

export const mockScheduledScans: ScheduledScan[] = [
  {
    id: 1,
    name: 'Daily Acme Scan',
    engineIds: [1],
    engineNames: ['Full Scan'],
    organizationId: 1,
    organizationName: 'Acme Corporation',
    targetId: null,
    targetName: null,
    scanMode: 'organization',
    cronExpression: '0 2 * * *',
    isEnabled: true,
    nextRunTime: '2024-12-30T02:00:00Z',
    lastRunTime: '2024-12-29T02:00:00Z',
    runCount: 45,
    createdAt: '2024-11-15T08:00:00Z',
    updatedAt: '2024-12-29T02:00:00Z',
  },
  {
    id: 2,
    name: 'Weekly TechStart Vuln Scan',
    engineIds: [3],
    engineNames: ['Vulnerability Only'],
    organizationId: 2,
    organizationName: 'TechStart Inc',
    targetId: null,
    targetName: null,
    scanMode: 'organization',
    cronExpression: '0 3 * * 0',
    isEnabled: true,
    nextRunTime: '2025-01-05T03:00:00Z',
    lastRunTime: '2024-12-29T03:00:00Z',
    runCount: 12,
    createdAt: '2024-10-01T10:00:00Z',
    updatedAt: '2024-12-29T03:00:00Z',
  },
  {
    id: 3,
    name: 'Hourly API Monitoring',
    engineIds: [2],
    engineNames: ['Quick Scan'],
    organizationId: null,
    organizationName: null,
    targetId: 12,
    targetName: 'api.acme.com',
    scanMode: 'target',
    cronExpression: '0 * * * *',
    isEnabled: true,
    nextRunTime: '2024-12-29T12:00:00Z',
    lastRunTime: '2024-12-29T11:00:00Z',
    runCount: 720,
    createdAt: '2024-12-01T00:00:00Z',
    updatedAt: '2024-12-29T11:00:00Z',
  },
  {
    id: 4,
    name: 'Monthly Full Scan - Finance',
    engineIds: [1],
    engineNames: ['Full Scan'],
    organizationId: 3,
    organizationName: 'Global Finance Ltd',
    targetId: null,
    targetName: null,
    scanMode: 'organization',
    cronExpression: '0 0 1 * *',
    isEnabled: false,
    nextRunTime: '2025-01-01T00:00:00Z',
    lastRunTime: '2024-12-01T00:00:00Z',
    runCount: 6,
    createdAt: '2024-06-01T08:00:00Z',
    updatedAt: '2024-12-20T15:00:00Z',
  },
  {
    id: 5,
    name: 'RetailMax Daily Quick',
    engineIds: [2, 3],
    engineNames: ['Quick Scan', 'Vulnerability Only'],
    organizationId: null,
    organizationName: null,
    targetId: 8,
    targetName: 'retailmax.com',
    scanMode: 'target',
    cronExpression: '0 4 * * *',
    isEnabled: true,
    nextRunTime: '2024-12-30T04:00:00Z',
    lastRunTime: '2024-12-29T04:00:00Z',
    runCount: 30,
    createdAt: '2024-11-29T09:00:00Z',
    updatedAt: '2024-12-29T04:00:00Z',
  },
]

export function getMockScheduledScans(params?: {
  page?: number
  pageSize?: number
  search?: string
}): GetScheduledScansResponse {
  const page = params?.page || 1
  const pageSize = params?.pageSize || 10
  const search = params?.search?.toLowerCase() || ''

  let filtered = mockScheduledScans

  if (search) {
    filtered = filtered.filter(
      s =>
        s.name.toLowerCase().includes(search) ||
        s.organizationName?.toLowerCase().includes(search) ||
        s.targetName?.toLowerCase().includes(search)
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

export function getMockScheduledScanById(id: number): ScheduledScan | undefined {
  return mockScheduledScans.find(s => s.id === id)
}
