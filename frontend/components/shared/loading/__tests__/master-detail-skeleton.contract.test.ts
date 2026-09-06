import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/loading/master-detail-skeleton.tsx"), "utf8")

describe("master-detail-skeleton contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function MasterDetailSkeleton")
    expect(source).toContain("layer?: LoadingLayer")
    expect(source).toContain('layer = "workspace"')
    expect(source).toContain("getLoadingOwnerAttributes")
    expect(source).toContain("owner?: string")
    expect(source).toContain("owner !== undefined && !owner.trim()")
    expect(source).toContain('owner ? getLoadingOwnerAttributes({ owner, layer, intent: "data" }) : {}')
    expect(source).toContain("from \"@/components/ui/skeleton\"")
  })

  it("derives header search geometry from the shared search toolbar skeleton", () => {
    expect(source).toContain('from "@/components/shared/loading/search-toolbar-skeleton"')
    expect(source).toContain("<SearchToolbarSkeleton")
    expect(source).toContain('inputWidthMode="fill"')
    expect(source).toContain('toolbarDensity="standard"')
    expect(source).not.toContain('Skeleton className="h-9 w-full rounded-lg"')
  })

  it("top-aligns header actions with search controls in loading state", () => {
    expect(source).toContain('className="flex gap-4 items-start justify-between lg:px-6 px-4 py-4"')
    expect(source).not.toContain('className="flex gap-4 items-center justify-between lg:px-6 px-4 py-4"')
  })
})
