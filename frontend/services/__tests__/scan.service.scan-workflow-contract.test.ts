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

import { api } from '@/lib/api-client'
import { bulkInitiateScan, initiateScan, quickScan } from '@/services/scan.service'

const validSubdomainConfiguration = {
  steps: {
    subdomain_discovery: {
      enabled: true,
      engineConfig: { recon: { enabled: true, timeout: 3600 } },
    },
  },
}

describe('scan.service scanWorkflow contract', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('normal scan 请求通过 batchCreate 发送 canonical scanWorkflow 和对象配置', async () => {
    vi.mocked(api.post).mockResolvedValue({
      data: {
        count: 1,
        createdCount: 1,
        skipped: [],
        failed: [],
        scans: [
          {
            id: 1,
            targetId: 7,
            scanWorkflow: 'scanWorkflows/subdomain_discovery',
            plannedEngineIds: [],
            triggerType: 'manual',
            inputSource: 'scanSnapshot',
            status: 'pending',
            createdAt: '2026-01-01T00:00:00Z',
          },
        ],
      },
    } as never)

    await initiateScan({
      targetId: 7,
      configuration: validSubdomainConfiguration,
      scanWorkflow: 'subdomain_discovery',
      inputSource: 'scanSnapshot',
    })

    expect(api.post).toHaveBeenCalledWith('/scans:batchCreate', {
      requests: [{ target: 'targets/7' }],
      scanWorkflow: 'scanWorkflows/subdomain_discovery',
      inputSource: 'scanSnapshot',
      configuration: validSubdomainConfiguration,
    })
    expect(vi.mocked(api.post).mock.calls[0]?.[1]).not.toHaveProperty('workflowIds')
    expect(vi.mocked(api.post).mock.calls[0]?.[1]).not.toHaveProperty('workflowNames')
  })

  it('quick scan 请求发送 canonical scanWorkflow 和对象配置', async () => {
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
      configuration: validSubdomainConfiguration,
      scanWorkflow: 'subdomain_discovery',
      inputSource: 'targetInventory',
    })

    expect(api.post).toHaveBeenCalledWith('/scans:quickCreate', {
      targets: ['example.com'],
      scanWorkflow: 'scanWorkflows/subdomain_discovery',
      inputSource: 'targetInventory',
      configuration: validSubdomainConfiguration,
    })
    expect(vi.mocked(api.post).mock.calls[0]?.[1]).not.toHaveProperty('workflowIds')
    expect(vi.mocked(api.post).mock.calls[0]?.[1]).not.toHaveProperty('workflowNames')
  })

  it('指定节点时，所有 Scan 创建入口发送 canonical Agent resource name', async () => {
    vi.mocked(api.post).mockResolvedValue({
      data: { count: 0, createdCount: 0, targetStats: { created: 0, skipped: 0, failed: 0 }, assetStats: { websites: 0, endpoints: 0 }, errors: [], skipped: [], failed: [], scans: [] },
    } as never)

    await bulkInitiateScan({ targetIds: [7], agentId: 42, configuration: validSubdomainConfiguration, scanWorkflow: 'subdomain_discovery', inputSource: 'scanSnapshot' })
    expect(vi.mocked(api.post).mock.calls[0]?.[1]).toMatchObject({ agent: 'agents/42' })

    await quickScan({ targets: [{ name: 'example.com' }], agentId: 42, configuration: validSubdomainConfiguration, scanWorkflow: 'subdomain_discovery', inputSource: 'targetInventory' })
    expect(vi.mocked(api.post).mock.calls[1]?.[1]).toMatchObject({ agent: 'agents/42' })
  })

  it('bulk scan 请求通过 batchCreate 发送 canonical scanWorkflow 和对象配置', async () => {
    vi.mocked(api.post).mockResolvedValue({
      data: {
        count: 2,
        createdCount: 2,
        skipped: [],
        failed: [],
        scans: [
          {
            id: 1,
            targetId: 7,
            scanWorkflow: 'scanWorkflows/subdomain_discovery',
            plannedEngineIds: [],
            triggerType: 'manual',
            inputSource: 'scanSnapshot',
            status: 'pending',
            createdAt: '2026-01-01T00:00:00Z',
          },
          {
            id: 2,
            targetId: 8,
            scanWorkflow: 'scanWorkflows/subdomain_discovery',
            plannedEngineIds: [],
            triggerType: 'manual',
            inputSource: 'scanSnapshot',
            status: 'pending',
            createdAt: '2026-01-01T00:00:00Z',
          },
        ],
      },
    } as never)

    await bulkInitiateScan({
      targetIds: [7, 8],
      organizationIds: [3],
      configuration: validSubdomainConfiguration,
      scanWorkflow: 'subdomain_discovery',
      inputSource: 'scanSnapshot',
    })

    expect(api.post).toHaveBeenCalledTimes(1)
    expect(api.post).toHaveBeenCalledWith('/scans:batchCreate', {
      requests: [{ target: 'targets/7' }, { target: 'targets/8' }, { organization: 'organizations/3' }],
      scanWorkflow: 'scanWorkflows/subdomain_discovery',
      inputSource: 'scanSnapshot',
      configuration: validSubdomainConfiguration,
    })
    expect(vi.mocked(api.post).mock.calls[0]?.[1]).not.toHaveProperty('workflowIds')
    expect(vi.mocked(api.post).mock.calls[0]?.[1]).not.toHaveProperty('workflowNames')
  })

  it('organization bulk scan 请求使用 organization resource name', async () => {
    vi.mocked(api.post).mockResolvedValue({
      data: { count: 0, createdCount: 0, skipped: [], failed: [], scans: [] },
    } as never)

    await bulkInitiateScan({
      organizationIds: [3],
      configuration: validSubdomainConfiguration,
      scanWorkflow: 'subdomain_discovery',
      inputSource: 'scanSnapshot',
    })

    expect(api.post).toHaveBeenCalledWith('/scans:batchCreate', {
      requests: [{ organization: 'organizations/3' }],
      scanWorkflow: 'scanWorkflows/subdomain_discovery',
      inputSource: 'scanSnapshot',
      configuration: validSubdomainConfiguration,
    })
  })
})
