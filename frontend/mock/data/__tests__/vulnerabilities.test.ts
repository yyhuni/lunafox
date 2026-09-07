import { describe, expect, it } from "vitest"

import { getMockVulnerabilities, mockVulnerabilities } from "../vulnerabilities"

describe("getMockVulnerabilities", () => {
  it("支持按 isReviewed=true 过滤", () => {
    const result = getMockVulnerabilities({
      page: 1,
      pageSize: 200,
      isReviewed: true,
    })

    expect(result.total).toBe(mockVulnerabilities.filter((item) => item.isReviewed).length)
    expect(result.results.every((item) => item.isReviewed)).toBe(true)
  })

  it("支持按 isReviewed=false 过滤", () => {
    const result = getMockVulnerabilities({
      page: 1,
      pageSize: 200,
      isReviewed: false,
    })

    expect(result.total).toBe(mockVulnerabilities.filter((item) => !item.isReviewed).length)
    expect(result.results.every((item) => !item.isReviewed)).toBe(true)
  })

  it("uses exact raw URL equality for the ordinary URL filter", () => {
    const url = mockVulnerabilities[0]?.url
    const result = getMockVulnerabilities({
      page: 1,
      pageSize: 200,
      url,
    })

    expect(result.total).toBe(1)
    expect(result.results).toEqual([expect.objectContaining({ url })])
    expect(getMockVulnerabilities({ page: 1, pageSize: 200, url: `${url} ` }).total).toBe(0)
  })
})
