import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const productionOuterRhythmSources = [
  "app/overview/page.tsx",
  "app/targets/page.tsx",
  "app/organizations/page.tsx",
  "app/vulnerabilities/page.tsx",
  "components/settings/notifications/notification-settings-page-content.tsx",
  "components/settings/system-logs/system-logs-view.tsx",
].map((filePath) => {
  const source = readFileSync(path.resolve(process.cwd(), filePath), "utf8")
  const companionSource = getCompanionLayoutSource(filePath)

  return {
    filePath,
    source: `${source}\n${companionSource}`,
  }
})

const pageShellDensitySource = readFileSync(
  path.resolve(process.cwd(), "components/shared/layout/page-shell-density.ts"),
  "utf8"
)

const productionGutterSources = [
  "app/targets/page.tsx",
  "app/organizations/page.tsx",
  "app/vulnerabilities/page.tsx",
  "components/overview/overview-section-layouts.tsx",
  "components/settings/notifications/notification-settings-page-content.tsx",
  "components/settings/system-logs/system-logs-view.tsx",
].map((filePath) => {
  const source = readFileSync(path.resolve(process.cwd(), filePath), "utf8")
  const companionSource = getCompanionLayoutSource(filePath)

  return {
    filePath,
    source: `${source}\n${companionSource}`,
  }
})

function getCompanionLayoutSource(filePath: string) {
  if (filePath.endsWith("notification-settings-page-content.tsx")) {
    return readFileSync(path.resolve(process.cwd(), "components/settings/notifications/notification-settings-layout.ts"), "utf8")
  }
  if (filePath.endsWith("system-logs-view.tsx")) {
    return readFileSync(path.resolve(process.cwd(), "components/settings/system-logs/system-logs-layout.ts"), "utf8")
  }
  return ""
}

describe("spacing-foundation contract", () => {
  it("keeps core production page shells on the shared outer rhythm", () => {
    expect(pageShellDensitySource).toContain('COMPACT_PAGE_RHYTHM_CLASS = "gap-3 py-3"')
    for (const { filePath, source } of productionOuterRhythmSources) {
      expect(source).toMatch(/COMPACT_[A-Z_]+_SHELL_CLASS|COMPACT_PAGE_RHYTHM_CLASS/)
      expect(filePath).toBeTruthy()
    }
  })

  it("keeps core production page shells on the shared gutter contract", () => {
    expect(pageShellDensitySource).toContain('COMPACT_CONTENT_GUTTER_CLASS = "px-3"')
    expect(pageShellDensitySource).toContain('COMPACT_FORM_OVERLAY_HORIZONTAL_INSET_CLASS = "px-4"')
    expect(pageShellDensitySource).toContain('COMPACT_FORM_OVERLAY_VERTICAL_INSET_CLASS = "py-3"')
    expect(pageShellDensitySource).toContain('COMPACT_FORM_OVERLAY_SECTION_GAP_CLASS = "gap-3"')
    expect(pageShellDensitySource).toContain('COMPACT_FORM_DIALOG_CONTENT_CLASS')
    expect(pageShellDensitySource).toContain('COMPACT_PAGE_SCROLL_AREA_CLASS = "min-h-0 min-w-0 flex-1"')
    expect(pageShellDensitySource).toContain('COMPACT_PAGE_SCROLL_AREA_CONTENT_CLASS = "!min-w-0 w-full"')
    expect(pageShellDensitySource).toContain('COMPACT_PAGE_SCROLL_AREA_VIEWPORT_CLASS = "!overflow-x-hidden"')
    for (const { source } of productionGutterSources) {
      expect(source).toContain("COMPACT_CONTENT_GUTTER_CLASS")
    }
  })

  it("does not allow route-local full-page padding in the vulnerabilities production shell", () => {
    const vulnerabilitiesPage = productionOuterRhythmSources.find(({ filePath }) =>
      filePath.includes("app/vulnerabilities/page.tsx")
    )

    expect(vulnerabilitiesPage?.source).not.toContain("p-4 md:p-6")
  })
})
