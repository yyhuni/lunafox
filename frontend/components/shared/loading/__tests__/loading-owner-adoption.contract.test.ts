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
      if (entry === "__tests__") continue
      files.push(...listSourceFiles(relativePath))
      continue
    }

    if (/\.(ts|tsx)$/.test(entry)) {
      files.push(relativePath)
    }
  }

  return files
}

describe("shared loading owner adoption contract", () => {
  it("marks route-level shared skeleton owners with the route layer", () => {
    for (const file of [
      "components/shared/loading/page-section-skeleton.tsx",
      "components/shared/loading/settings-page-skeleton.tsx",
    ]) {
      const source = readFrontendSource(file)

      expect(source).toContain("getLoadingOwnerAttributes")
      expect(source).toContain("layer?: LoadingLayer")
      expect(source).toContain('layer = "route"')
      expect(source).toContain("owner?: string")
      expect(source).toContain("owner !== undefined && !owner.trim()")
      expect(source).toContain('owner ? getLoadingOwnerAttributes({ owner, layer, intent: "route" }) : {}')
    }
  })

  it("marks interaction loading surfaces with the interaction layer", () => {
    for (const file of [
      "components/shared/loading/interaction-loading-dialog.tsx",
      "components/shared/loading/interaction-loading-drawer.tsx",
    ]) {
      const source = readFrontendSource(file)

      expect(source).toContain("getLoadingOwnerAttributes")
      expect(source).toContain('layer: "interaction"')
      expect(source).toContain('intent: "interaction"')
      expect(source).toContain('getLoadingOwnerAttributes({ owner, layer: "interaction", intent: "interaction" })')
    }
  })

  it("marks the server boot owner as the boot layer", () => {
    const source = readFrontendSource("app/layout.tsx")

    expect(source).toContain('data-loading-owner="initial-boot"')
    expect(source).toContain('data-loading-layer="boot"')
    expect(source).toContain('data-loading-intent="boot"')
  })

  it("keeps production loading owners behind the shared helper", () => {
    const allowedDirectOwnerFiles = new Set([
      "app/layout.tsx",
      "components/boot-layer-controller.tsx",
      "components/route-progress.tsx",
      "components/shared/loading/loading-owner.ts",
    ])
    const directOwnerFiles = [
      ...listSourceFiles("app"),
      ...listSourceFiles("components"),
    ].filter((file) => (
      !allowedDirectOwnerFiles.has(file) &&
      readFrontendSource(file).includes("data-loading-owner")
    ))

    expect(directOwnerFiles).toEqual([])
  })
})
