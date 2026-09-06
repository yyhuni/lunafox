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
import { AgentLogQueryError, agentService } from '@/services/agent.service'

describe('agentService.fetchDistributedLogs', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('请求普通 JSON 接口并解析 viewer metadata', async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        results: [
          {
            id: 'agt_1:lunafox-agent:1740381601000000000:abc:000001',
            ts: '2026-02-24T10:00:01Z',
            tsNs: '1740381601000000000',
            stream: 'stdout',
            line: 'hello',
            truncated: false,
          },
        ],
        nextPageToken: 'cursor-1',
        previousPageToken: 'cursor-prev',
        hasOlder: true,
        hasNewer: false,
        caughtUp: true,
        gap: false,
        gapReason: '',
      },
    } as never)

    const result = await agentService.fetchDistributedLogs({
      agentNodeId: 1,
      container: 'lunafox-agent',
      limit: 50,
      cursor: 'cursor-0',
    })

    expect(result.logs).toHaveLength(1)
    expect(result.nextCursor).toBe('cursor-1')
    expect(result.previousCursor).toBe('cursor-prev')
    expect(result.hasOlder).toBe(true)
    expect(result.hasNewer).toBe(false)
    expect(result.caughtUp).toBe(true)
    expect(result.gap).toBe(false)
    expect(result.gapReason).toBe('')

    expect(api.get).toHaveBeenCalledTimes(1)
    expect(api.get).toHaveBeenCalledWith('/admin/agents/1/logEntries', {
      params: {
        container: 'lunafox-agent',
        pageSize: '50',
        pageToken: 'cursor-0',
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
      agentService.fetchDistributedLogs({
        agentNodeId: 1,
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
      agentService.fetchDistributedLogs({
        agentNodeId: 1,
        container: 'lunafox-agent',
      })
    ).rejects.toMatchObject({
      code: 'bad_request',
      status: 400,
    } satisfies Partial<AgentLogQueryError>)
  })

  it('保留取消请求错误，避免日志状态机将正常取消当成网络失败', async () => {
    const canceledError = new Error('canceled') as Error & { code: string }
    canceledError.name = 'CanceledError'
    canceledError.code = 'ERR_CANCELED'
    vi.mocked(api.get).mockRejectedValue(canceledError)

    await expect(agentService.fetchDistributedLogs({ agentNodeId: 1, container: 'lunafox-agent' })).rejects.toBe(canceledError)
    await expect(agentService.fetchDistributedLogs({ agentNodeId: 1, container: 'lunafox-agent' })).rejects.not.toBeInstanceOf(AgentLogQueryError)
  })

  it('会过滤掉缺失 id 的非法日志项', async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        results: [
          {
            ts: '2026-02-24T10:00:01Z',
            tsNs: '1740381601000000000',
            stream: 'stdout',
            line: 'invalid-without-id',
            truncated: false,
          },
        ],
        nextPageToken: '',
        previousPageToken: '',
        hasOlder: false,
        hasNewer: false,
        caughtUp: true,
        gap: false,
      },
    } as never)

    const result = await agentService.fetchDistributedLogs({
      agentNodeId: 1,
      container: 'lunafox-agent',
    })

    expect(result.logs).toHaveLength(0)
    expect(result.nextCursor).toBe('')
    expect(result.previousCursor).toBe('')
    expect(result.hasOlder).toBe(false)
    expect(result.hasNewer).toBe(false)
    expect(result.caughtUp).toBe(true)
    expect(result.gap).toBe(false)
    expect(result.gapReason).toBe('')
  })

  it('older 查询携带 direction 参数', async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        results: [],
        nextPageToken: 'follow-1',
        previousPageToken: 'older-1',
        hasOlder: true,
        hasNewer: false,
        caughtUp: true,
        gap: false,
        gapReason: '',
      },
    } as never)

    await agentService.fetchDistributedLogs({
      agentNodeId: 1,
      container: 'lunafox-agent',
      cursor: 'older-0',
      direction: 'older',
    })

    expect(api.get).toHaveBeenCalledWith('/admin/agents/1/logEntries', {
      params: {
        container: 'lunafox-agent',
        pageSize: '200',
        pageToken: 'older-0',
        direction: 'older',
      },
      signal: undefined,
    })
  })

  it('拒绝超过单页上限或无效的 pageSize，且不发起请求', async () => {
    await expect(
      agentService.fetchDistributedLogs({ agentNodeId: 1, container: 'lunafox-agent', limit: 501 }),
    ).rejects.toMatchObject({ code: 'bad_request' } satisfies Partial<AgentLogQueryError>)
    await expect(
      agentService.fetchDistributedLogs({ agentNodeId: 1, container: 'lunafox-agent', limit: -1 }),
    ).rejects.toMatchObject({ code: 'bad_request' } satisfies Partial<AgentLogQueryError>)
    expect(api.get).not.toHaveBeenCalled()
  })

  it('缺失 metadata 时使用保守默认值', async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        results: [],
        nextPageToken: 'cursor-1',
      },
    } as never)

    const result = await agentService.fetchDistributedLogs({
      agentNodeId: 1,
      container: 'lunafox-agent',
    })

    expect(result.previousCursor).toBe('')
    expect(result.hasOlder).toBe(false)
    expect(result.hasNewer).toBe(false)
    expect(result.caughtUp).toBe(false)
    expect(result.gap).toBe(false)
    expect(result.gapReason).toBe('')
  })
})
