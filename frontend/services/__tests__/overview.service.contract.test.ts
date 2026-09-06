import { beforeEach, describe, expect, it, vi } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "services/overview.service.ts"), "utf8")

const apiMocks = vi.hoisted(() => ({
  get: vi.fn(),
}))

vi.mock("@/lib/api-client", () => ({
  api: apiMocks,
}))

import { api } from "@/lib/api-client"
import { getAssetStatistics, getStatisticsHistory } from "@/services/overview.service"

describe("overview.service contract", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })
  it("preserves current source markers", () => {
    expect(source).toContain("from '@/lib/api-client'")
  })

  it("fetches server runtime metrics through the system runtime endpoint", () => {
    expect(source).toContain("getServerRuntimeMetrics")
    expect(source).toContain("/admin/system/runtimeMetrics/current")
    expect(source).toContain("ServerRuntimeMetrics")
  })

  it("uses the versioned asset statistics report views without a history fallback", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [] } as never)

    await getAssetStatistics()
    await getStatisticsHistory(7)

    expect(api.get).toHaveBeenNthCalledWith(1, "/assetStatistics")
    expect(api.get).toHaveBeenNthCalledWith(2, "/assetStatistics/history", { params: { days: 7 } })
    expect(source).not.toContain("/assets/statistics")
    expect(source).toContain("getStatisticsHistory = cache(async (days: number)")
  })
})
