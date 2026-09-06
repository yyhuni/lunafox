import { beforeEach, describe, expect, it, vi } from 'vitest'

const apiMocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  patch: vi.fn(),
  delete: vi.fn(),
}))

vi.mock('@/lib/api-client', () => ({
  api: apiMocks,
}))

vi.mock('@/mock', () => ({
  USE_MOCK: false,
  mockDelay: vi.fn(),
  getMockScans: vi.fn(),
  getMockScanById: vi.fn(),
  mockScanStatistics: {},
}))

import { api } from '@/lib/api-client'
import { getScanLogs } from '@/services/scan.service'
import type { GetScanLogsResponse } from '@/types/scan.types'

describe('scan.service task progress log contract', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('returns required task-owned progress logs through the taskProgressLogs resource', async () => {
    const response: GetScanLogsResponse = {
      results: [
        {
          id: 701,
          taskId: 7001,
          level: 'info',
          content: 'runtime: execute subdomain discovery runtime engine',
          createdAt: '2026-06-19T01:02:03Z',
        },
      ],
      nextPageToken: 'cursor-2',
    }

    vi.mocked(api.get).mockResolvedValue({
      data: response,
    } as never)

    const result = await getScanLogs(7, { pageSize: 20, pageToken: 'cursor-1' })

    expect(api.get).toHaveBeenCalledWith('/scans/7/taskProgressLogs', {
      params: { pageSize: 20, pageToken: 'cursor-1' },
    })
    expect(result.results[0]).toHaveProperty('taskId', 7001)
    expect(result.results[0]).toMatchObject({ id: 701, content: 'runtime: execute subdomain discovery runtime engine' })
    expect(result.nextPageToken).toBe('cursor-2')
    expect(vi.mocked(api.get).mock.calls[0]?.[0]).not.toBe('/scans/7/logs')
  })
})
