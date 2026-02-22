import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { AgentLogStreamEvent } from '@/types/agent-log.types'

const tokenManagerMocks = vi.hoisted(() => ({
  getAccessToken: vi.fn(),
}))

vi.mock('@/lib/api-client', () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
  },
  tokenManager: tokenManagerMocks,
}))

vi.mock('@/lib/env', () => ({
  getBackendBaseUrl: () => 'https://example.test',
}))

vi.mock('@/mock', () => ({
  USE_MOCK: false,
  mockDelay: vi.fn(),
  getMockAgents: vi.fn(),
  getMockAgentById: vi.fn(),
  getMockRegistrationToken: vi.fn(),
}))

import { AgentLogStreamError, agentService } from '@/services/agent.service'

describe('agentService.streamAgentLogs', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    tokenManagerMocks.getAccessToken.mockReturnValue('test-token')
  })

  it('解析 SSE 事件并按顺序回调', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        createSSEStream([
          'event: status\ndata: {"requestId":"req-1","status":"started"}\n\n',
          'event: log\ndata: {"requestId":"req-1","ts":"2026-02-22T12:00:00Z","stream":"stdout","line":"hello","truncated":false}\n\n',
          'event: ping\ndata: {"requestId":"req-1","ts":"2026-02-22T12:00:01Z"}\n\n',
          'event: done\ndata: {"requestId":"req-1","reason":"eof"}\n\n',
        ]),
        { status: 200, headers: { 'Content-Type': 'text/event-stream' } }
      )
    )
    vi.stubGlobal('fetch', fetchMock)

    const events: AgentLogStreamEvent[] = []
    await agentService.streamAgentLogs({
      agentId: 1,
      container: 'lunafox-agent',
      onEvent: (event) => events.push(event),
    })

    expect(events.map((event) => event.type)).toEqual(['status', 'log', 'ping', 'done'])
    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toContain('/api/admin/agents/1/logs/stream')
    expect(url).toContain('container=lunafox-agent')
    expect(url).toContain('tail=200')
    expect(url).toContain('follow=true')
    expect(url).toContain('timestamps=true')
    expect((init.headers as Record<string, string>).Authorization).toBe('Bearer test-token')
  })

  it('遇到 done 事件后立刻结束读取', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        createSSEStream([
          'event: status\ndata: {"requestId":"req-2","status":"started"}\n\n',
          'event: done\ndata: {"requestId":"req-2","reason":"eof"}\n\n',
          'event: log\ndata: {"requestId":"req-2","ts":"2026-02-22T12:00:02Z","stream":"stdout","line":"should-not-appear","truncated":false}\n\n',
        ]),
        { status: 200, headers: { 'Content-Type': 'text/event-stream' } }
      )
    )
    vi.stubGlobal('fetch', fetchMock)

    const events: AgentLogStreamEvent[] = []
    await agentService.streamAgentLogs({
      agentId: 2,
      container: 'lunafox-agent',
      onEvent: (event) => events.push(event),
    })

    expect(events.map((event) => event.type)).toEqual(['status', 'done'])
  })

  it('HTTP 错误响应会抛出带 code/status 的 AgentLogStreamError', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          error: {
            code: 'AGENT_OFFLINE',
            message: 'Agent is offline',
          },
        }),
        { status: 404, headers: { 'Content-Type': 'application/json' } }
      )
    )
    vi.stubGlobal('fetch', fetchMock)

    await expect(
      agentService.streamAgentLogs({
        agentId: 3,
        container: 'lunafox-agent',
        onEvent: () => {},
      })
    ).rejects.toMatchObject({
      code: 'AGENT_OFFLINE',
      status: 404,
    } satisfies Partial<AgentLogStreamError>)
  })

  it('当响应无 body 时抛出 STREAM_UNAVAILABLE', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    await expect(
      agentService.streamAgentLogs({
        agentId: 4,
        container: 'lunafox-agent',
        onEvent: () => {},
      })
    ).rejects.toMatchObject({
      code: 'STREAM_UNAVAILABLE',
    } satisfies Partial<AgentLogStreamError>)
  })
})

function createSSEStream(frames: string[]): ReadableStream<Uint8Array> {
  const encoder = new TextEncoder()
  return new ReadableStream<Uint8Array>({
    start(controller) {
      for (const frame of frames) {
        controller.enqueue(encoder.encode(frame))
      }
      controller.close()
    },
  })
}
