import { beforeEach, describe, expect, it, vi } from "vitest"

const apiMocks = vi.hoisted(() => ({
  get: vi.fn(),
}))

vi.mock("@/lib/api-client", () => ({
  api: apiMocks,
  default: apiMocks,
}))

import { api } from "@/lib/api-client"
import { SystemLogQueryError, systemLogService } from "@/services/system-log.service"

describe("systemLogService.fetchServerLogs", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("请求 Server system logEntries 并解析 viewer metadata", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        results: [
          {
            id: "srv:lunafox-server:1740381601000000000:stdout:abc:000000",
            ts: "2026-02-24T10:00:01Z",
            tsNs: "1740381601000000000",
            stream: "stdout",
            line: "{\"level\":\"info\",\"msg\":\"server started\"}",
            truncated: false,
          },
        ],
        nextPageToken: "follow-token",
        previousPageToken: "older-token",
        hasOlder: true,
        hasNewer: false,
        caughtUp: true,
        gap: false,
        gapReason: "",
      },
    } as never)

    const result = await systemLogService.fetchServerLogs({
      limit: 50,
      cursor: "cursor-0",
      direction: "newer",
    })

    expect(result.logs).toHaveLength(1)
    expect(result.nextCursor).toBe("follow-token")
    expect(result.previousCursor).toBe("older-token")
    expect(result.hasOlder).toBe(true)
    expect(result.caughtUp).toBe(true)

    expect(api.get).toHaveBeenCalledWith("/admin/system/logEntries", {
      params: {
        pageSize: "50",
        pageToken: "cursor-0",
        direction: "newer",
      },
      signal: undefined,
    })
  })

  it("保留取消请求错误，避免日志轮询把正常取消误报成获取失败", async () => {
    const canceledError = new Error("canceled") as Error & { code: string }
    canceledError.name = "CanceledError"
    canceledError.code = "ERR_CANCELED"
    vi.mocked(api.get).mockRejectedValue(canceledError)

    await expect(systemLogService.fetchServerLogs()).rejects.toBe(canceledError)
    await expect(systemLogService.fetchServerLogs()).rejects.not.toBeInstanceOf(SystemLogQueryError)
  })

  it("拒绝超过单页上限或无效的 pageSize，且不发起请求", async () => {
    await expect(systemLogService.fetchServerLogs({ limit: 501 })).rejects.toMatchObject({
      code: "bad_request",
    } satisfies Partial<SystemLogQueryError>)
    await expect(systemLogService.fetchServerLogs({ limit: -1 })).rejects.toMatchObject({
      code: "bad_request",
    } satisfies Partial<SystemLogQueryError>)
    expect(api.get).not.toHaveBeenCalled()
  })
})
