import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "hooks/use-websites.ts"), "utf8")
const typeSource = readFileSync(path.resolve(process.cwd(), "types/website.types.ts"), "utf8")

describe("use-websites contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useTargetWebSites")
    expect(source).toContain("from '@tanstack/react-query'")
  })

  it("keys target and scan website queries by canonical collection controls", () => {
    expect(source).toContain("WebsiteListQueryParams")
    expect(typeSource).toContain("pageSize?: number")
    expect(typeSource).toContain("pageToken?: string")
    expect(typeSource).toContain("filter?: string")
    expect(typeSource).toContain("orderBy?: string")
    expect(source).toContain("pageToken: params?.pageToken")
    expect(source).toContain("orderBy: params?.orderBy")
    expect(source).not.toContain("params: { page: number; pageSize: number; filter?: string }")
  })

  it("exposes parent-scoped website filter option queries", () => {
    expect(typeSource).toContain("WebsiteFilterOptionField")
    expect(source).toContain("filterOptions: (scope: \"target\" | \"scan\", id: number, field: WebsiteFilterOptionField)")
    expect(source).toContain("export function useTargetWebsiteFilterOptions")
    expect(source).toContain("export function useScanWebsiteFilterOptions")
    expect(source).toContain("WebsiteService.getTargetWebsiteFilterOptions")
    expect(source).toContain("WebsiteService.getScanWebsiteFilterOptions")
  })

  it("keys Website detail separately from collections and does not retain the pseudo relation resource", () => {
    expect(source).toContain('detail: (websiteId: number)')
    expect(source).toContain('"detail", websiteId')
    expect(source).toContain("export function useWebsite")
    expect(source).toContain("WebsiteService.getWebsite")
    expect(source).not.toContain("websiteRelations")
    expect(typeSource).not.toContain("WebsiteRelationEvidence")
  })

  it("does not preserve legacy page as a website query fact", () => {
    expect(source).not.toContain("page: params?.page")
    expect(source).not.toContain("page: number")
  })
})
