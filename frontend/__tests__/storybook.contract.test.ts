import { describe, expect, it } from "vitest"
import { existsSync, readFileSync } from "node:fs"
import path from "node:path"

const mainSource = readFileSync(path.resolve(process.cwd(), ".storybook/main.ts"), "utf8")
const previewSource = readFileSync(path.resolve(process.cwd(), ".storybook/preview.tsx"), "utf8")
const packageJson = JSON.parse(readFileSync(path.resolve(process.cwd(), "package.json"), "utf8")) as {
  scripts: Record<string, string>
}

describe("storybook contract", () => {
  it("uses the Next.js Vite framework and a11y addon", () => {
    expect(mainSource).toContain("@storybook/nextjs-vite")
    expect(mainSource).toContain("@storybook/addon-a11y")
  })

  it("exposes locale and mock scenario globals through preview", () => {
    expect(previewSource).toContain("mockScenario")
    expect(previewSource).toContain("setMockScenario")
    expect(previewSource).toContain("mswLoader")
  })

  it("registers Storybook scripts", () => {
    expect(packageJson.scripts.storybook).toBe("storybook dev -p 6006")
    expect(packageJson.scripts["build-storybook"]).toBe("storybook build")
  })

  it("keeps every core mock resource on a scenario-backed Storybook surface", () => {
    const requiredStories = [
      "components/organization/organization-list.stories.tsx",
      "components/target/all-targets-detail-view.stories.tsx",
      "components/scan/history/scan-history-list.stories.tsx",
      "components/subdomains/subdomains-detail-view.stories.tsx",
    ]

    for (const storyPath of requiredStories) {
      expect(existsSync(path.resolve(process.cwd(), storyPath)), storyPath).toBe(true)
    }
  })
})
