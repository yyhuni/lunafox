import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"
import { getMockWebsiteFilterOptions } from "../websites"

const source = readFileSync(path.resolve(process.cwd(), "mock/data/websites.ts"), "utf8")

describe("websites contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function getMockWebsites")
    expect(source).toContain("export function getMockWebsiteFilterOptions")
    expect(source).toContain("from '@/types/website.types'")
  })

  it("returns parent-scoped website filter options for all migrated facets", () => {
    expect(getMockWebsiteFilterOptions({ targetId: 1, field: "statusCode" }).results).toEqual([
      { value: "200", label: "200", count: 3 },
      { value: "301", label: "301", count: 1 },
    ])
    expect(getMockWebsiteFilterOptions({ targetId: 1, field: "tech" }).results).toEqual(
      expect.arrayContaining([
        { value: "React", label: "React", count: 2 },
        { value: "Node.js", label: "Node.js", count: 1 },
      ])
    )
    expect(getMockWebsiteFilterOptions({ targetId: 1, field: "webserver" }).results).toEqual([
      { value: "nginx/1.24.0", label: "nginx/1.24.0", count: 4 },
    ])
    expect(getMockWebsiteFilterOptions({ targetId: 1, field: "contentType" }).results).toEqual([
      { value: "application/json", label: "application/json", count: 1 },
      { value: "text/html; charset=utf-8", label: "text/html; charset=utf-8", count: 3 },
    ])
    expect(getMockWebsiteFilterOptions({ targetId: 1, field: "vhost" }).results).toEqual([
      { value: "false", label: "false", count: 4 },
    ])
  })
})
