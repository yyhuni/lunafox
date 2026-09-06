import { afterEach, describe, expect, it } from "vitest"

import { resetMockNucleiPocs } from "@/mock/data/nuclei-pocs"
import { resetMockScenario, setMockScenario } from "@/mock/scenarios"

const API_BASE = "http://localhost/v1"

describe("Nuclei POC filter-options mock network boundary", () => {
  afterEach(() => {
    resetMockNucleiPocs()
    resetMockScenario()
  })

  it("serves the complete-catalog tag projection and rejects malformed queries", async () => {
    const response = await fetch(`${API_BASE}/nucleiPocs/filterOptions?field=tags`)
    expect(response.status).toBe(200)
    await expect(response.json()).resolves.toEqual({
      results: expect.arrayContaining([
        { value: "actuator", label: "actuator", count: 1 },
        { value: "rce", label: "rce", count: 1 },
      ]),
    })

    const invalidResponses = await Promise.all([
      fetch(`${API_BASE}/nucleiPocs/filterOptions`),
      fetch(`${API_BASE}/nucleiPocs/filterOptions?field=severity`),
      fetch(`${API_BASE}/nucleiPocs/filterOptions?field=tags&filter=x`),
      fetch(`${API_BASE}/nucleiPocs/filterOptions?field=tags&field=tags`),
    ])
    expect(invalidResponses.every((candidate) => candidate.status === 400)).toBe(true)
  })

  it("returns an empty options collection for the empty catalog scenario", async () => {
    setMockScenario("empty")
    const response = await fetch(`${API_BASE}/nucleiPocs/filterOptions?field=tags`)
    expect(response.status).toBe(200)
    await expect(response.json()).resolves.toEqual({ results: [] })
  })
})
