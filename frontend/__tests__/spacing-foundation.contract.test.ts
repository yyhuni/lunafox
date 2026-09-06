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
    for (const { filePath, source } of productionOuterRhythmSources) {
      expect(source).toContain("gap-4")
      expect(source).toContain("py-4")

      // The command-only overview header has a reviewed tighter desktop rhythm.
      if (filePath === "app/overview/page.tsx") {
        expect(source).toContain("md:gap-5")
        expect(source).toContain("md:pt-5")
        expect(source).toContain("md:pb-6")
        continue
      }

      expect(source).toContain("md:gap-6")
      expect(source).toContain("md:py-6")
    }
  })

  it("keeps core production page shells on the shared gutter contract", () => {
    for (const { source } of productionGutterSources) {
      expect(source).toContain("px-4")
      expect(source).toContain("lg:px-6")
    }
  })

  it("does not allow route-local full-page padding in the vulnerabilities production shell", () => {
    const vulnerabilitiesPage = productionOuterRhythmSources.find(({ filePath }) =>
      filePath.includes("app/vulnerabilities/page.tsx")
    )

    expect(vulnerabilitiesPage?.source).not.toContain("p-4 md:p-6")
  })
})
