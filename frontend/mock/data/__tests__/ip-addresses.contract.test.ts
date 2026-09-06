import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"
import { getMockIPAddresses, getMockPortOptions } from "../ip-addresses"

const source = readFileSync(path.resolve(process.cwd(), "mock/data/ip-addresses.ts"), "utf8")

describe("ip-addresses contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function getMockIPAddresses")
    expect(source).toContain("export function getMockPortOptions")
    expect(source).toContain("from '@/types/ip-address.types'")
  })

  it("supports port smart-filter syntax in mock IP address queries", () => {
    const result = getMockIPAddresses({ filter: 'port="443"', page: 1, pageSize: 50 })

    expect(result.total).toBeGreaterThan(0)
    expect(result.results.every((item) => item.ports.includes(443))).toBe(true)
  })

  it("returns scoped port options with one count per matching IP", () => {
    expect(getMockPortOptions({ domain: "acme.com" }).results).toEqual(
      expect.arrayContaining([
        { value: "80", label: "80", count: 4 },
        { value: "443", label: "443", count: 5 },
        { value: "3306", label: "3306", count: 2 },
      ])
    )
  })
})
