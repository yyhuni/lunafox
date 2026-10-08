import { afterEach, describe, expect, it } from "vitest"
import { buildScans } from "@/mock/builders/core"

describe("buildScans mock list size", () => {
  afterEach(() => {
    delete process.env.NEXT_PUBLIC_MOCK_SCAN_LIST_SIZE
  })

  it("keeps the fixture page when the list size override is unset", () => {
    const page = buildScans({ page: 1, pageSize: 10 })

    expect(page.total).toBeLessThanOrEqual(100)
    expect(page.results).toHaveLength(Math.min(10, page.total))
  })

  it("repeats wrapped scan rows past the virtual-scroll threshold", () => {
    process.env.NEXT_PUBLIC_MOCK_SCAN_LIST_SIZE = "160"
    const page = buildScans({ pageSize: 200 })

    expect(page.total).toBe(160)
    expect(page.results).toHaveLength(160)
    expect(page.results[0]?.plannedEngineIds.length).toBeGreaterThan(1)
    expect(page.results[0]?.target?.name).toContain("customer-api-shared-gateway")
    expect(new Set(page.results.map((scan) => scan.id)).size).toBe(160)
  })
})
