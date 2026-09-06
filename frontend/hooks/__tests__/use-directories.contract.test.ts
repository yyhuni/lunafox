import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "hooks/use-directories.ts"), "utf8")
const typeSource = readFileSync(path.resolve(process.cwd(), "types/directory.types.ts"), "utf8")

describe("use-directories contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useTargetDirectories")
    expect(source).toContain("from '@tanstack/react-query'")
  })

  it("keys target and scan directory queries by canonical collection controls", () => {
    expect(source).toContain("DirectoryListQueryParams")
    expect(source).toContain("DirectoryFilterOptionField")
    expect(typeSource).toContain("pageSize?: number")
    expect(typeSource).toContain("pageToken?: string")
    expect(typeSource).toContain("filter?: string")
    expect(typeSource).toContain("orderBy?: string")
    expect(typeSource).toContain('export type DirectoryFilterOptionField = "status" | "contentType"')
    expect(source).toContain("pageToken: params?.pageToken")
    expect(source).toContain("orderBy: params?.orderBy")
    expect(source).not.toContain("params: { page: number; pageSize: number; filter?: string }")
  })

  it("exposes scoped filter option hooks for migrated directory facets", () => {
    expect(source).toContain("filterOptions: (scope: \"target\" | \"scan\", id: number, field: DirectoryFilterOptionField)")
    expect(source).toContain("export function useTargetDirectoryFilterOptions")
    expect(source).toContain("DirectoryService.getTargetDirectoryFilterOptions")
    expect(source).toContain("export function useScanDirectoryFilterOptions")
    expect(source).toContain("DirectoryService.getScanDirectoryFilterOptions")
  })

  it("does not preserve legacy page as a directory query fact", () => {
    expect(source).not.toContain("page: params?.page")
    expect(source).not.toContain("page: number")
  })
})
