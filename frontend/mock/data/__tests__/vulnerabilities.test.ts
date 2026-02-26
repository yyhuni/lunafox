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
    expect(result.vulnerabilities.every((item) => item.isReviewed)).toBe(true)
  })

  it("支持按 isReviewed=false 过滤", () => {
    const result = getMockVulnerabilities({
      page: 1,
      pageSize: 200,
      isReviewed: false,
    })

    expect(result.total).toBe(mockVulnerabilities.filter((item) => !item.isReviewed).length)
    expect(result.vulnerabilities.every((item) => !item.isReviewed)).toBe(true)
  })

  it("支持按 filter 文本过滤", () => {
    const result = getMockVulnerabilities({
      page: 1,
      pageSize: 200,
      filter: "xss",
    })

    expect(result.total).toBeGreaterThan(0)
    expect(
      result.vulnerabilities.every((item) => {
        const haystack = `${item.url} ${item.vulnType} ${item.description ?? ""}`.toLowerCase()
        return haystack.includes("xss")
      })
    ).toBe(true)
  })
})
