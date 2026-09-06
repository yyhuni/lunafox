import { describe, expect, it } from "vitest"
import { readdirSync, readFileSync, statSync } from "node:fs"
import path from "node:path"

const readSource = (filePath: string) => readFileSync(path.resolve(process.cwd(), filePath), "utf8")

function collectFiles(dir: string): string[] {
  return readdirSync(path.resolve(process.cwd(), dir)).flatMap((entry) => {
    const fullPath = path.resolve(process.cwd(), dir, entry)
    const stat = statSync(fullPath)

    if (stat.isDirectory()) {
      if (entry === "__tests__") return []
      return collectFiles(path.relative(process.cwd(), fullPath))
    }

    return entry.endsWith(".tsx") ? [path.relative(process.cwd(), fullPath)] : []
  })
}

const foundationConsumers = [
  "components/ui/table.tsx",
  "components/ui/badge.tsx",
  "components/ui/tabs.tsx",
  "components/ui/card.tsx",
  "components/ui/field.tsx",
  "components/ui/dialog.tsx",
  "components/ui/sheet.tsx",
  "components/ui/alert-dialog.tsx",
  "components/ui/sidebar.tsx",
  "components/ui/command.tsx",
]

describe("typography standardization contract", () => {
  it("requires high-propagation foundation components to consume shared typography roles", () => {
    for (const filePath of foundationConsumers) {
      const source = readSource(filePath)
      expect(source, filePath).toContain('from "@/lib/typography"')
      expect(source, filePath).toContain("textRole")
    }
  })

  it("moves readable table and filter typography away from English-centric defaults", () => {
    const tableSource = readSource("components/ui/table.tsx")
    const tabsSource = readSource("components/ui/tabs.tsx")
    const badgeSource = readSource("components/ui/badge.tsx")

    expect(tableSource).not.toContain("tracking-[0.14em] uppercase")
    expect(tabsSource).not.toContain("tracking-[0.12em] uppercase")
    expect(badgeSource).not.toContain("tracking-[0.08em] uppercase")
  })

  it("guards the vulnerabilities page against known ad hoc text hierarchy patterns", () => {
    const sources = [
      readSource("components/vulnerabilities/vulnerabilities-vertical-table.tsx"),
      readSource("components/vulnerabilities/vulnerability-vertical-detail.tsx"),
      readSource("components/vulnerabilities/vulnerabilities-vertical-header.tsx"),
    ].join("\n")

    expect(sources).toContain('from "@/lib/typography"')
    expect(sources).not.toContain("text-foreground/80")
    expect(sources).not.toContain("tracking-[0.14em]")
    expect(sources).not.toContain("tracking-[0.12em]")
  })

  it("requires the page header and sidebar shell to consume typography v2 roles", () => {
    const pageHeaderSource = readSource("components/common/page-header.tsx")
    const appSidebarSource = readSource("components/app-sidebar.tsx")
    const uiSidebarSource = readSource("components/ui/sidebar.tsx")
    const navUserSource = readSource("components/nav-user.tsx")
    const dropdownMenuOwnersSource = readSource("components/shared/dropdown-menu-owners.tsx")

    expect(pageHeaderSource).toContain("textRole.pageTitle")
    expect(pageHeaderSource).toContain("textRole.pageDescription")
    expect(appSidebarSource).toContain("textRole")
    expect(appSidebarSource).toContain("textRole.helperText")
    expect(uiSidebarSource).toContain("textRole.navLabel")
    expect(uiSidebarSource).toContain("textRole.helperText")
    expect(navUserSource).toContain("SidebarUserMenu")
    expect(dropdownMenuOwnersSource).toContain("textRole.navLabel")
    expect(dropdownMenuOwnersSource).toContain("textRole.helperText")
  })

  it("documents long-form and responsive typography policy in the UI foundation guide", () => {
    const readmeSource = readSource("components/ui/README.md")

    expect(readmeSource).toContain("bodyLarge")
    expect(readmeSource).toContain("Responsive Typography Rules")
    expect(readmeSource).toContain("PageHeader")
    expect(readmeSource).toContain("Dense data surfaces")
  })

  it("requires the vulnerabilities detail and toolbar to consume v2 hierarchy roles", () => {
    const detailSource = readSource("components/vulnerabilities/vulnerability-vertical-detail.tsx")
    const headerSource = readSource("components/vulnerabilities/vulnerabilities-vertical-header.tsx")

    expect(detailSource).toContain("textRole.panelTitle")
    expect(detailSource).toContain("textRole.helperText")
    expect(headerSource).toContain("textRole.tab")
    expect(headerSource).toContain("SharedCompactPagination")
  })

  it("keeps business table cell text on table-specific typography roles", () => {
    const tableColumnFiles = collectFiles("components").filter(
      (filePath) => filePath.endsWith("columns.tsx") || filePath.endsWith("results-table.tsx")
    )

    const offenders = tableColumnFiles.flatMap((filePath) => {
      const source = readSource(filePath)
      const forbiddenMatches = source.match(/textRole\.(?:body|bodySubtle|code)\b/g)
      if (!forbiddenMatches) return []
      return forbiddenMatches.map((match) => `${filePath}: ${match}`)
    })

    expect(offenders).toEqual([])
  })
})
