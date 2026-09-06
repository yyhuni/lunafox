import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "hooks/use-ip-addresses.ts"), "utf8")

describe("use-ip-addresses contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useTargetIPAddresses")
    expect(source).toContain("from \"@tanstack/react-query\"")
  })

  it("normalizes canonical list query params without legacy page", () => {
    expect(source).toContain("pageSize: params?.pageSize ?? 10")
    expect(source).toContain("pageToken: params?.pageToken")
    expect(source).toContain("orderBy: params?.orderBy ?? \"\"")
    expect(source).not.toContain("page: params?.page")
    expect(source).not.toContain("Required<GetIPAddressesParams>")
  })

  it("exposes parent-scoped port option queries", () => {
    expect(source).toContain("portOptions: (scope: \"target\" | \"scan\", id: number)")
    expect(source).toContain("export function useTargetPortOptions")
    expect(source).toContain("export function useScanPortOptions")
    expect(source).toContain("IPAddressService.getTargetPortOptions")
    expect(source).toContain("IPAddressService.getScanPortOptions")
  })
})
