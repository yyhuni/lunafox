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
  getMockAgents: vi.fn(),
  getMockAgentById: vi.fn(),
  getMockRegistrationToken: vi.fn(),
}))

import { api } from '@/lib/api-client'
import { AgentLogQueryError, agentService } from '@/services/agent.service'

describe('agentService.fetchAgentLogs', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('请求普通 JSON 接口并解析 logs/nextCursor/hasMore', async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        logs: [
          {
            id: 'agt_1:lunafox-agent:1740381601000000000:abc:000001',
            ts: '2026-02-24T10:00:01Z',
            tsNs: '1740381601000000000',
            stream: 'stdout',
            line: 'hello',
            truncated: false,
          },
        ],
        nextCursor: 'cursor-1',
        hasMore: true,
      },
    } as never)

    const result = await agentService.fetchAgentLogs({
      agentId: 1,
      container: 'lunafox-agent',
      limit: 50,
      cursor: 'cursor-0',
    })

    expect(result.logs).toHaveLength(1)
    expect(result.nextCursor).toBe('cursor-1')
    expect(result.hasMore).toBe(true)

    expect(api.get).toHaveBeenCalledTimes(1)
    expect(api.get).toHaveBeenCalledWith('/admin/agents/1/logs', {
      params: {
        container: 'lunafox-agent',
        limit: '50',
        cursor: 'cursor-0',
      },
      signal: undefined,
    })
  })

  it('HTTP 错误会抛出带 code/status 的 AgentLogQueryError（503）', async () => {
    vi.mocked(api.get).mockRejectedValue({
      response: {
        status: 503,
        data: {
          error: {
            code: 'loki_unavailable',
            message: 'Loki is unavailable',
          },
        },
      },
      message: 'Request failed with status code 503',
    } as never)

    await expect(
      agentService.fetchAgentLogs({
        agentId: 1,
        container: 'lunafox-agent',
      })
    ).rejects.toMatchObject({
      code: 'loki_unavailable',
      status: 503,
    } satisfies Partial<AgentLogQueryError>)
  })

  it('HTTP 错误会抛出带 code/status 的 AgentLogQueryError（400）', async () => {
    vi.mocked(api.get).mockRejectedValue({
      response: {
        status: 400,
        data: {
          error: {
            code: 'bad_request',
            message: 'direction is deprecated, please remove it',
          },
        },
      },
      message: 'Request failed with status code 400',
    } as never)

    await expect(
      agentService.fetchAgentLogs({
        agentId: 1,
        container: 'lunafox-agent',
      })
    ).rejects.toMatchObject({
      code: 'bad_request',
      status: 400,
    } satisfies Partial<AgentLogQueryError>)
  })

  it('会过滤掉缺失 id 的非法日志项', async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        logs: [
          {
            ts: '2026-02-24T10:00:01Z',
            tsNs: '1740381601000000000',
            stream: 'stdout',
            line: 'invalid-without-id',
            truncated: false,
          },
        ],
        nextCursor: '',
        hasMore: false,
      },
    } as never)

    const result = await agentService.fetchAgentLogs({
      agentId: 1,
      container: 'lunafox-agent',
    })

    expect(result.logs).toHaveLength(0)
    expect(result.nextCursor).toBe('')
    expect(result.hasMore).toBe(false)
  })
})
