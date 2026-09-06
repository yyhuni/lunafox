import type {
  GetScheduledScansResponse,
  ScheduledScan,
  ScheduledScanOverviewSummary,
} from '@/types/scheduled-scan.types'
import { getMockCanonicalWorkflowConfiguration } from '@/mock/data/scan-workflows'
import { getNextCronExecutions } from '@/lib/scheduled-scan-helpers'

function createMockScheduledConfiguration() {
  return getMockCanonicalWorkflowConfiguration()
}

const mockScheduledScanClock = new Date()

function getMockNextRunTime(cronExpression: string): string | null {
  return getNextCronExecutions(cronExpression, mockScheduledScanClock, 1)[0]?.toISOString() ?? null
}

const initialMockScheduledScans: ScheduledScan[] = [
  {
    id: 1,
    name: 'scheduledScans/1',
    displayName: 'Daily Acme Scan',
    scanWorkflow: 'scanWorkflows/default',
    inputSource: 'scanSnapshot',
    configuration: createMockScheduledConfiguration(),
    organizationId: 1,
    organizationName: 'Acme Corporation',
    targetId: null,
    targetName: null,
    scanMode: 'organization',
    cronExpression: '0 2 * * *',
    isEnabled: true,
    nextRunTime: getMockNextRunTime('0 2 * * *'),
    lastRunTime: '2024-12-29T02:00:00Z',
    runCount: 45,
    successfulHandoffCount: 43,
    failedHandoffCount: 2,
    createdAt: '2024-11-15T08:00:00Z',
    updatedAt: '2024-12-29T02:00:00Z',
  },
  {
    id: 2,
    name: 'scheduledScans/2',
    displayName: 'Weekly TechStart Vuln Scan',
    scanWorkflow: 'scanWorkflows/default',
    inputSource: 'scanSnapshot',
    configuration: createMockScheduledConfiguration(),
    organizationId: 2,
    organizationName: 'TechStart Inc',
    targetId: null,
    targetName: null,
    scanMode: 'organization',
    cronExpression: '0 3 * * 0',
    isEnabled: true,
    nextRunTime: getMockNextRunTime('0 3 * * 0'),
    lastRunTime: '2024-12-29T03:00:00Z',
    runCount: 12,
    successfulHandoffCount: 12,
    failedHandoffCount: 0,
    createdAt: '2024-10-01T10:00:00Z',
    updatedAt: '2024-12-29T03:00:00Z',
  },
  {
    id: 3,
    name: 'scheduledScans/3',
    displayName: 'Hourly API Monitoring',
    scanWorkflow: 'scanWorkflows/default',
    inputSource: 'scanSnapshot',
    configuration: createMockScheduledConfiguration(),
    organizationId: null,
    organizationName: null,
    targetId: 12,
    targetName: 'api.acme.com',
    scanMode: 'target',
    cronExpression: '0 * * * *',
    isEnabled: true,
    nextRunTime: getMockNextRunTime('0 * * * *'),
    lastRunTime: '2024-12-29T11:00:00Z',
    runCount: 720,
    successfulHandoffCount: 718,
    failedHandoffCount: 2,
    createdAt: '2024-12-01T00:00:00Z',
    updatedAt: '2024-12-29T11:00:00Z',
  },
  {
    id: 4,
    name: 'scheduledScans/4',
    displayName: 'Monthly Full Scan - Finance',
    scanWorkflow: 'scanWorkflows/default',
    inputSource: 'scanSnapshot',
    configuration: createMockScheduledConfiguration(),
    organizationId: 3,
    organizationName: 'Global Finance Ltd',
    targetId: null,
    targetName: null,
    scanMode: 'organization',
    cronExpression: '0 0 1 * *',
    isEnabled: false,
    nextRunTime: null,
    lastRunTime: '2024-12-01T00:00:00Z',
    runCount: 6,
    successfulHandoffCount: 5,
    failedHandoffCount: 1,
    createdAt: '2024-06-01T08:00:00Z',
    updatedAt: '2024-12-20T15:00:00Z',
  },
  {
    id: 5,
    name: 'scheduledScans/5',
    displayName: 'RetailMax Daily Quick',
    scanWorkflow: 'scanWorkflows/default',
    inputSource: 'scanSnapshot',
    configuration: createMockScheduledConfiguration(),
    organizationId: null,
    organizationName: null,
    targetId: 8,
    targetName: 'retailmax.com',
    scanMode: 'target',
    cronExpression: '0 4 * * *',
    isEnabled: true,
    nextRunTime: getMockNextRunTime('0 4 * * *'),
    lastRunTime: '2024-12-29T04:00:00Z',
    runCount: 30,
    successfulHandoffCount: 29,
    failedHandoffCount: 1,
    createdAt: '2024-11-29T09:00:00Z',
    updatedAt: '2024-12-29T04:00:00Z',
  },
]

export const mockScheduledScans = structuredClone(initialMockScheduledScans)

export function resetMockScheduledScans() {
  mockScheduledScans.splice(0, mockScheduledScans.length, ...structuredClone(initialMockScheduledScans))
}

export function getMockScheduledScans(params?: {
  pageSize?: number
  pageToken?: string
  search?: string
  targetId?: number
  organizationId?: number
}): GetScheduledScansResponse {
  const pageTokenMatch = params?.pageToken?.match(/^mock-scheduled-scan-page-(\d+)$/)
  const page = pageTokenMatch ? Number.parseInt(pageTokenMatch[1]!, 10) : 1
  const pageSize = params?.pageSize || 10
  const search = params?.search?.toLowerCase() || ''

  let filtered = mockScheduledScans

  if (params?.organizationId) {
    filtered = filtered.filter((scan) => scan.organizationId === params.organizationId)
  }

  if (params?.targetId) {
    filtered = filtered.filter((scan) => scan.targetId === params.targetId)
  }

  if (search) {
    filtered = filtered.filter(
      s =>
        s.displayName.toLowerCase().includes(search) ||
        s.organizationName?.toLowerCase().includes(search) ||
        s.targetName?.toLowerCase().includes(search)
    )
  }

  const total = filtered.length
  const start = (page - 1) * pageSize
  const scheduledScans = filtered.slice(start, start + pageSize)

  return {
    scheduledScans,
    totalSize: total,
    ...(start + pageSize < total
      ? { nextPageToken: `mock-scheduled-scan-page-${page + 1}` }
      : {}),
  }
}

type MockScheduledScanOverviewResponse = Omit<ScheduledScanOverviewSummary, 'upcomingScheduledScans'> & {
  upcomingScheduledScans: Array<{
    name: string
    displayName: string
    organization: string | null
    organizationDisplayName: string | null
    target: string | null
    targetDisplayName: string | null
    nextRunTime: string
  }>
}

export function getMockScheduledScanOverviewSummary(
  asOfTime: Date = new Date()
): MockScheduledScanOverviewResponse {
  const asOfTimestamp = asOfTime.getTime()
  const start = new Date(asOfTime)
  start.setUTCHours(0, 0, 0, 0)
  const end = start.getTime() + 24 * 60 * 60 * 1000
  const next24HoursEnd = asOfTimestamp + 24 * 60 * 60 * 1000
  const enabledScheduledScans = mockScheduledScans.filter((scan) => scan.isEnabled)
  const nextRunTimeOf = (scan: ScheduledScan) => scan.nextRunTime === null ? null : Date.parse(scan.nextRunTime)

  return {
    asOfTime: asOfTime.toISOString(),
    enabledScheduledScanCount: enabledScheduledScans.length,
    pausedScheduledScanCount: mockScheduledScans.length - enabledScheduledScans.length,
    todayScheduledScanCount: enabledScheduledScans.filter((scan) => {
      const nextRunTime = nextRunTimeOf(scan)
      return nextRunTime !== null && nextRunTime >= start.getTime() && nextRunTime < end
    }).length,
    next24HoursScheduledScanCount: enabledScheduledScans.filter((scan) => {
      const nextRunTime = nextRunTimeOf(scan)
      return nextRunTime !== null && nextRunTime >= asOfTimestamp && nextRunTime < next24HoursEnd
    }).length,
    upcomingScheduledScans: enabledScheduledScans
      .filter((scan) => nextRunTimeOf(scan) !== null)
      .sort((left, right) => {
        const runTimeOrder = nextRunTimeOf(left)! - nextRunTimeOf(right)!
        return runTimeOrder || left.id - right.id
      })
      .slice(0, 5)
      .map((scan) => ({
        name: scan.name,
        displayName: scan.displayName,
        organization: scan.organizationId === null ? null : `organizations/${scan.organizationId}`,
        organizationDisplayName: scan.organizationName,
        target: scan.targetId === null ? null : `targets/${scan.targetId}`,
        targetDisplayName: scan.targetName,
        nextRunTime: scan.nextRunTime!,
      })),
  }
}

export function getMockScheduledScanById(id: number): ScheduledScan | undefined {
  return mockScheduledScans.find(s => s.id === id)
}

export function deleteMockScheduledScan(id: number): boolean {
  const index = mockScheduledScans.findIndex(scan => scan.id === id)
  if (index === -1) {
    return false
  }

  mockScheduledScans.splice(index, 1)
  return true
}
