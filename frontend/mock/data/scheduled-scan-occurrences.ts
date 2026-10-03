import type {
  GetScheduledScanOccurrencesResponse,
  ScheduledScanOccurrence,
} from '@/types/scheduled-scan.types'

// Occurrence history fixtures mirror the server projection contract: the
// status is carried explicitly and durationMs exists only on SUCCEEDED rows.
const initialMockOccurrences: Record<number, ScheduledScanOccurrence[]> = {
  1: [
    {
      name: `scheduledScans/1/occurrences/101`,
      id: 101,
      scheduledFor: '2024-12-29T02:00:00Z',
      attemptedAt: '2024-12-29T02:00:01Z',
      dispatchedAt: null,
      status: 'FAILED',
      failureKind: 'scan_create_failed',
      failureCause: 'ENGINE_UNAVAILABLE',
      failureMessage: 'Scan creation did not complete: engine package unavailable on the execution node.',
      retryCount: 3,
      nextRetryAt: null,
      durationMs: null,
    },
    {
      name: `scheduledScans/1/occurrences/102`,
      id: 102,
      scheduledFor: '2024-12-28T02:00:00Z',
      attemptedAt: '2024-12-28T02:00:00Z',
      dispatchedAt: '2024-12-28T02:00:01Z',
      status: 'SUCCEEDED',
      failureKind: null,
      failureCause: null,
      failureMessage: null,
      retryCount: 0,
      nextRetryAt: null,
      durationMs: 840,
    },
    {
      name: `scheduledScans/1/occurrences/103`,
      id: 103,
      scheduledFor: '2024-12-27T02:00:00Z',
      attemptedAt: '2024-12-27T02:00:00Z',
      dispatchedAt: '2024-12-27T02:00:01Z',
      status: 'SUCCEEDED',
      failureKind: null,
      failureCause: null,
      failureMessage: null,
      retryCount: 0,
      nextRetryAt: null,
      durationMs: 910,
    },
    {
      name: `scheduledScans/1/occurrences/104`,
      id: 104,
      scheduledFor: '2024-12-26T02:00:00Z',
      attemptedAt: '2024-12-26T02:00:00Z',
      dispatchedAt: null,
      status: 'FAILED',
      failureKind: 'scan_create_failed',
      failureCause: 'CONFIG_RESOURCE_UNAVAILABLE',
      failureMessage: 'Scan creation did not complete: referenced configuration resource is unavailable.',
      retryCount: 3,
      nextRetryAt: null,
      durationMs: null,
    },
    {
      name: `scheduledScans/1/occurrences/105`,
      id: 105,
      scheduledFor: '2024-12-25T02:00:00Z',
      attemptedAt: '2024-12-25T02:00:00Z',
      dispatchedAt: '2024-12-25T02:00:01Z',
      status: 'SUCCEEDED',
      failureKind: null,
      failureCause: null,
      failureMessage: null,
      retryCount: 0,
      nextRetryAt: null,
      durationMs: 780,
    },
  ],
  3: [
    {
      name: `scheduledScans/3/occurrences/301`,
      id: 301,
      scheduledFor: '2024-12-29T11:00:00Z',
      attemptedAt: '2024-12-29T11:00:02Z',
      dispatchedAt: null,
      status: 'RETRYING',
      failureKind: null,
      failureCause: 'AGENT_NOT_FOUND',
      failureMessage: 'Scan creation did not complete: selected agent no longer exists.',
      retryCount: 2,
      nextRetryAt: '2024-12-29T11:04:00Z',
      durationMs: null,
    },
    {
      name: `scheduledScans/3/occurrences/302`,
      id: 302,
      scheduledFor: '2024-12-29T10:00:00Z',
      attemptedAt: '2024-12-29T10:00:00Z',
      dispatchedAt: '2024-12-29T10:00:01Z',
      status: 'SUCCEEDED',
      failureKind: null,
      failureCause: null,
      failureMessage: null,
      retryCount: 0,
      nextRetryAt: null,
      durationMs: 620,
    },
    {
      name: `scheduledScans/3/occurrences/303`,
      id: 303,
      scheduledFor: '2024-12-29T09:00:00Z',
      attemptedAt: null,
      dispatchedAt: null,
      status: 'PENDING',
      failureKind: null,
      failureCause: null,
      failureMessage: null,
      retryCount: 0,
      nextRetryAt: null,
      durationMs: null,
    },
  ],
  5: [
    {
      name: `scheduledScans/5/occurrences/501`,
      id: 501,
      scheduledFor: '2024-12-29T04:00:00Z',
      attemptedAt: '2024-12-29T04:00:01Z',
      dispatchedAt: null,
      status: 'FAILED',
      failureKind: 'scan_create_failed',
      failureCause: 'TARGET_UNAVAILABLE',
      failureMessage: 'Scan creation did not complete: scan target is deleted or invalid.',
      retryCount: 3,
      nextRetryAt: null,
      durationMs: null,
    },
    {
      name: `scheduledScans/5/occurrences/502`,
      id: 502,
      scheduledFor: '2024-12-28T04:00:00Z',
      attemptedAt: '2024-12-28T04:00:00Z',
      dispatchedAt: '2024-12-28T04:00:01Z',
      status: 'SUCCEEDED',
      failureKind: null,
      failureCause: null,
      failureMessage: null,
      retryCount: 0,
      nextRetryAt: null,
      durationMs: 730,
    },
  ],
}

export const mockScheduledScanOccurrences = structuredClone(initialMockOccurrences)

export function resetMockScheduledScanOccurrences() {
  for (const key of Object.keys(mockScheduledScanOccurrences)) {
    delete mockScheduledScanOccurrences[Number(key)]
  }
  Object.assign(mockScheduledScanOccurrences, structuredClone(initialMockOccurrences))
}

export function getMockScheduledScanOccurrences(
  scheduledScanId: number,
  params?: { page?: number; pageSize?: number }
): GetScheduledScanOccurrencesResponse {
  const rows = mockScheduledScanOccurrences[scheduledScanId] ?? []
  const pageSize = params?.pageSize || 10
  const page = params?.page || 1
  const start = (page - 1) * pageSize
  const statusCounts = {
    pending: 0,
    dispatching: 0,
    retrying: 0,
    succeeded: 0,
    failed: 0,
  }
  for (const row of rows) {
    statusCounts[row.status.toLowerCase() as keyof typeof statusCounts] += 1
  }
  return {
    occurrences: rows.slice(start, start + pageSize),
    statusCounts,
    totalSize: rows.length,
    ...(start + pageSize < rows.length ? { nextPageToken: String(page + 1) } : {}),
  }
}
