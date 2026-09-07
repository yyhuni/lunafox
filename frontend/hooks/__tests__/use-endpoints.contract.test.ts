import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "hooks/use-endpoints.ts"), "utf8")
const typeSource = readFileSync(path.resolve(process.cwd(), "types/endpoint.types.ts"), "utf8")

describe("use-endpoints contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useEndpoint")
    expect(source).toContain("from \"@tanstack/react-query\"")
  })

  it("keys target and scan endpoint queries by canonical collection controls", () => {
    expect(source).toContain("EndpointListQueryParams")
    expect(typeSource).toContain("pageSize?: number")
    expect(typeSource).toContain("pageToken?: string")
    expect(typeSource).toContain("filter?: string")
    expect(typeSource).toContain("orderBy?: string")
    expect(source).toContain("pageToken: params?.pageToken")
    expect(source).toContain("orderBy: params?.orderBy")
    expect(source).not.toContain("page: params?.page")
    expect(source).not.toContain("defaultParams.page")
  })

  it("exposes parent-scoped endpoint filter option queries", () => {
    expect(typeSource).toContain("EndpointFilterOptionField")
    expect(source).toContain("filterOptions: (scope: \"target\" | \"scan\", id: number, field: EndpointFilterOptionField)")
    expect(source).toContain("export function useTargetEndpointFilterOptions")
    expect(source).toContain("export function useScanEndpointFilterOptions")
    expect(source).toContain("EndpointService.getTargetEndpointFilterOptions")
    expect(source).toContain("EndpointService.getScanEndpointFilterOptions")
  })

  it("does not preserve legacy page as an endpoint query fact", () => {
    expect(source).not.toContain("page: 1")
    expect(source).not.toContain("normalizePagination")
  })
})
