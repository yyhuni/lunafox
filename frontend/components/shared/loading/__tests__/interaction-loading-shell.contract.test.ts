import { describe, expect, it } from "vitest"
import { readdirSync, readFileSync, statSync } from "node:fs"
import path from "node:path"

function readFrontendSource(relativePath: string) {
  return readFileSync(path.resolve(process.cwd(), relativePath), "utf8")
}

function listSourceFiles(relativeDir: string): string[] {
  const absoluteDir = path.resolve(process.cwd(), relativeDir)
  const entries = readdirSync(absoluteDir)
  const files: string[] = []

  for (const entry of entries) {
    const absolutePath = path.join(absoluteDir, entry)
    const relativePath = path.relative(process.cwd(), absolutePath)
    const stat = statSync(absolutePath)

    if (stat.isDirectory()) {
      if (entry === "__tests__" || entry === "node_modules") continue
      files.push(...listSourceFiles(relativePath))
      continue
    }

    if (/\.(ts|tsx)$/.test(entry)) {
      files.push(relativePath)
    }
  }

  return files
}

const drawerMarkers = [
  "FormDrawer",
  "SheetContent",
  "DetailDrawer",
  "EditScheduledScanDialog",
  "CreateScheduledScanDialog",
  "CreateScheduledScanSheet",
]

describe("interaction loading shell contract", () => {
  it("does not open a dialog-shaped shell before a drawer or sheet", () => {
    const offenders = listSourceFiles("components").flatMap((file) => {
      if (file.endsWith("interaction-loading-dialog.tsx")) return []
      const source = readFrontendSource(file)
      const usesDialogShell = source.includes("<InteractionLoadingDialog")
      const opensDrawer = drawerMarkers.some((marker) => source.includes(marker))
      const preloadsEditDrawer = source.includes('openAfterLoad("edit-dialog"')
      if ((usesDialogShell && opensDrawer) || preloadsEditDrawer) return [file]
      return []
    })

    expect(offenders).toEqual([])
  })

  it("opens every scheduled scan edit sidebar directly on the requested tab", () => {
    const surfaces = [
      {
        file: "components/scan/scheduled/scheduled-scan-page.tsx",
        initialTab: "initialTab={editDialogInitialTab}",
      },
      {
        file: "components/organization/organization-detail-view-sections.tsx",
        initialTab: "initialTab={state.scheduledScanEditInitialTab}",
      },
      {
        file: "components/target/target-settings-sections.tsx",
        initialTab: "initialTab={state.editDialogInitialTab}",
      },
    ]

    for (const surface of surfaces) {
      const source = readFrontendSource(surface.file)
      expect(source).toContain(surface.initialTab)
      expect(source).not.toContain("InteractionLoadingDialog")
      expect(source).not.toContain('openAfterLoad("edit-dialog"')
    }
  })
})
