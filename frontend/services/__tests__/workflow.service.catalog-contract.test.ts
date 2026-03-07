import { beforeEach, describe, expect, it, vi } from 'vitest'

const apiClientMocks = vi.hoisted(() => ({
  get: vi.fn(),
}))

vi.mock('@/lib/api-client', () => ({
  default: apiClientMocks,
}))

vi.mock('@/mock', () => ({
  USE_MOCK: false,
  mockDelay: vi.fn(),
  getMockWorkflows: vi.fn(),
  getMockWorkflowProfiles: vi.fn(),
  getMockWorkflowProfileById: vi.fn(),
}))

import { getWorkflowProfiles, getWorkflows } from '@/services/workflow.service'

describe('workflow.service catalog contract', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('workflows 元数据会被归一化为只读目录项和对象配置', async () => {
    apiClientMocks.get.mockResolvedValue({
      data: [
        {
          workflowId: 'subdomain_discovery',
          displayName: 'Subdomain Discovery',
          description: 'Discover subdomains',
          version: '1.0.0',
          configuration: 'subdomain_discovery:\n  recon:\n    enabled: true',
        },
      ],
    })

    const workflows = await getWorkflows()

    expect(apiClientMocks.get).toHaveBeenCalledWith('/workflows/')
    expect(workflows).toEqual([
      {
        id: 1,
        name: 'subdomain_discovery',
        title: 'Subdomain Discovery',
        description: 'Discover subdomains',
        version: '1.0.0',
        configuration: {
          subdomain_discovery: {
            recon: {
              enabled: true,
            },
          },
        },
        isValid: undefined,
        createdAt: undefined,
        updatedAt: undefined,
      },
    ])
  })

  it('preset 列表保留 workflowIds 并把配置归一化为对象', async () => {
    apiClientMocks.get.mockResolvedValue({
      data: [
        {
          id: 'subdomain_discovery.fast',
          name: 'Fast Profile',
          workflowIds: ['subdomain_discovery'],
          configuration: 'subdomain_discovery: {}',
        },
      ],
    })

    const presets = await getWorkflowProfiles()

    expect(apiClientMocks.get).toHaveBeenCalledWith('/workflows/profiles/')
    expect(presets[0]?.workflowIds).toEqual(['subdomain_discovery'])
    expect(presets[0]?.configuration).toEqual({ subdomain_discovery: {} })
  })
})
