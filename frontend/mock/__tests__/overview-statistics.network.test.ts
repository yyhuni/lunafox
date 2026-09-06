import { describe, expect, it } from "vitest"

const API_BASE = "http://localhost/v1"

describe("Overview asset statistics mock network boundary", () => {
  it("serves the same versioned report views used by production Overview", async () => {
    const [statisticsResponse, historyResponse] = await Promise.all([
      fetch(`${API_BASE}/assetStatistics`),
      fetch(`${API_BASE}/assetStatistics/history?days=7`),
    ])

    expect(statisticsResponse.status).toBe(200)
    expect(historyResponse.status).toBe(200)

    const statistics = await statisticsResponse.json() as Record<string, unknown>
    const history = await historyResponse.json() as Array<Record<string, unknown>>
    expect(statistics).toMatchObject({
      totalAssets: expect.any(Number),
      vulnBySeverity: {
        critical: expect.any(Number),
        high: expect.any(Number),
        medium: expect.any(Number),
        low: expect.any(Number),
        info: expect.any(Number),
      },
    })
    expect(history).toHaveLength(7)
    expect(history[0]).toMatchObject({ date: expect.any(String), totalAssets: expect.any(Number) })
  })

  it("does not preserve the retired mock-only statistics paths", async () => {
    const response = await fetch(`${API_BASE}/assets/statistics`)

    expect(response.status).toBe(501)
  })
})
