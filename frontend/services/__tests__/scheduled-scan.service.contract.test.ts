import { beforeEach, describe, expect, it, vi } from "vitest"

const apiMocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  patch: vi.fn(),
  delete: vi.fn(),
}))

vi.mock("@/lib/api-client", () => ({
  api: apiMocks,
}))

vi.mock("@/mock", () => ({
  USE_MOCK: false,
  mockDelay: vi.fn(),
  getMockScheduledScans: vi.fn(),
  getMockScheduledScanById: vi.fn(),
}))

import { api } from "@/lib/api-client"
import { getScheduledScans } from "@/services/scheduled-scan.service"

describe("scheduled-scan.service contract", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("列表查询参数只发送 camelCase", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        results: [],
        total: 0,
        page: 1,
        pageSize: 25,
        totalPages: 0,
      },
    } as never)

    await getScheduledScans({
      page: 1,
      pageSize: 25,
      search: "demo",
      targetId: 7,
      organizationId: 9,
    })

    expect(api.get).toHaveBeenCalledWith("/scheduled-scans/", {
      params: {
        page: 1,
        pageSize: 25,
        search: "demo",
        targetId: 7,
        organizationId: 9,
      },
    })
    expect(vi.mocked(api.get).mock.calls[0]?.[1]?.params).not.toHaveProperty("target_id")
    expect(vi.mocked(api.get).mock.calls[0]?.[1]?.params).not.toHaveProperty("organization_id")
  })
})
