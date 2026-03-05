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
import { initiateScan, quickScan } from '@/services/scan.service'

describe('scan.service workflowNames contract', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('normal scan 请求只发送 workflowNames', async () => {
    vi.mocked(api.post).mockResolvedValue({
      data: {
        id: 1,
        targetId: 7,
        workflowNames: ['subdomain_discovery'],
        status: 'pending',
        createdAt: '2026-01-01T00:00:00Z',
      },
    } as never)

    await initiateScan({
      targetId: 7,
      configuration: 'subdomain_discovery: {}',
      workflowNames: ['subdomain_discovery'],
    })

    expect(api.post).toHaveBeenCalledWith('/scans/normal', {
      targetId: 7,
      workflowNames: ['subdomain_discovery'],
      configuration: 'subdomain_discovery: {}',
    })
    expect(vi.mocked(api.post).mock.calls[0]?.[1]).not.toHaveProperty('workflowIds')
  })

  it('quick scan 请求只发送 workflowNames', async () => {
    vi.mocked(api.post).mockResolvedValue({
      data: {
        count: 1,
        targetStats: { created: 1, skipped: 0, failed: 0 },
        assetStats: { websites: 0, endpoints: 0 },
        errors: [],
        scans: [],
      },
    } as never)

    await quickScan({
      targets: [{ name: 'example.com' }],
      configuration: 'subdomain_discovery: {}',
      workflowNames: ['subdomain_discovery'],
    })

    expect(api.post).toHaveBeenCalledWith('/scans/quick', {
      targets: ['example.com'],
      workflowNames: ['subdomain_discovery'],
      configuration: 'subdomain_discovery: {}',
    })
    expect(vi.mocked(api.post).mock.calls[0]?.[1]).not.toHaveProperty('workflowIds')
  })
})
