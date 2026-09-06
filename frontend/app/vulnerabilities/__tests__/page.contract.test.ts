import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "app/vulnerabilities/page.tsx"), "utf8")
const workspaceSource = readFileSync(
  path.resolve(process.cwd(), "app/vulnerabilities/vulnerabilities-workspace.tsx"),
  "utf8"
)

describe("page contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export default async function VulnerabilitiesPage")
    expect(source).toContain("className")
    expect(source).toContain("from \"next-intl/server\"")
    expect(source).toContain("from \"./vulnerabilities-workspace\"")
    expect(source).not.toContain('"use client"')
  })

  it("directly imports the route-critical vulnerabilities workspace", () => {
    expect(workspaceSource).toContain(
      'import { VulnerabilitiesVerticalView } from "@/components/vulnerabilities/vulnerabilities-vertical-view"'
    )
    expect(workspaceSource).toContain("VulnerabilitiesVerticalView")
    expect(workspaceSource).not.toContain("dynamic(")
    expect(workspaceSource).not.toContain("next/dynamic")
    expect(workspaceSource).not.toContain("ssr: false")
    expect(workspaceSource).not.toContain('"use client"')
  })

  it("keeps the workspace handoff as the only visible loading state", () => {
    expect(workspaceSource).not.toContain("loading: () => null")
    expect(source).not.toContain('from "@/components/shared/loading/data-table-skeleton"')
    expect(source).not.toContain("loading: () => <DataTableSkeleton")
  })

  it("uses the shared production page shell rhythm instead of route-local full-page padding", () => {
    expect(source).toContain("gap-4 py-4 md:gap-6 md:py-6")
    expect(source).toContain("px-4 lg:px-6")
    expect(source).toContain("min-h-0")
    expect(source).not.toContain("p-4 md:p-6")
  })

  it("provides the primary vulnerabilities workspace with a bounded flex height chain", () => {
    expect(source).toContain('className="@container/main flex min-h-0 flex-1 flex-col gap-4 py-4 md:gap-6 md:py-6"')
    expect(source).toContain('className="flex min-h-0 flex-1 flex-col px-4 lg:px-6"')
  })

  it("renders a page-level header above the vulnerabilities workspace", () => {
    expect(source).toContain('from "@/components/common/page-header"')
    expect(source).toContain("<PageHeader")
    expect(source).toContain("description=")
    expect(source).not.toContain("<h1")
  })
})
