import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/loading/detail-page-shell-skeleton.tsx"), "utf8")

describe("detail-page-shell-skeleton contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function DetailPageShellSkeleton")
    expect(source).toContain("getLoadingOwnerAttributes")
    expect(source).toContain("layer?: LoadingLayer")
    expect(source).toContain("intent?: LoadingIntent")
    expect(source).toContain('layer = "route"')
    expect(source).toContain('intent = "route"')
    expect(source).toContain("owner?: string")
    expect(source).toContain("primaryTabLabels?: readonly string[]")
    expect(source).toContain("activePrimaryTabIndex?: number")
    expect(source).toContain("owner ? getLoadingOwnerAttributes({ owner, layer, intent }) : {}")
    expect(source).toContain("TabsList")
    expect(source).toContain("TabsTrigger")
    expect(source).toContain("<Tabs value={`tab-${activePrimaryTabIndex}`}>")
    expect(source).toContain("primaryTabLabels[index]")
  })

  it("fast-fails blank owners while supporting metadata-free nested use", () => {
    expect(source).toContain('if (owner !== undefined && !owner.trim())')
    expect(source).toContain('throw new Error("DetailPageShellSkeleton owner must be non-empty when provided.")')
    expect(source).toContain('throw new Error("DetailPageShellSkeleton primaryTabLabels must not contain blank labels.")')
    expect(source).toContain('throw new Error("DetailPageShellSkeleton primaryTabLabels must match primaryTabCount.")')
    expect(source).toContain('throw new Error("DetailPageShellSkeleton activePrimaryTabIndex must reference a rendered tab.")')
    expect(source).toContain('variant="content"')
    expect(source).toContain("flex items-center gap-2 text-sm px-4 lg:px-6")
    expect(source).not.toContain("<button")
  })
})
