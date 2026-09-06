import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/loading/card-grid-skeleton.tsx"), "utf8")

describe("card-grid-skeleton contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function CardGridSkeleton")
    expect(source).toContain("className")
    expect(source).toContain("from \"@/lib/utils\"")
    expect(source).toContain("getLoadingOwnerAttributes")
    expect(source).toContain("layer?: LoadingLayer")
    expect(source).toContain('layer = "workspace"')
    expect(source).toContain("owner?: string")
    expect(source).toContain("owner !== undefined && !owner.trim()")
    expect(source).toContain('owner ? getLoadingOwnerAttributes({ owner, layer, intent: "data" }) : {}')
  })

  it("derives toolbar search geometry from the shared search toolbar skeleton", () => {
    expect(source).toContain('from "@/components/shared/loading/search-toolbar-skeleton"')
    expect(source).toContain("<SearchToolbarSkeleton")
    expect(source).toContain('inputWidthMode="fill"')
    expect(source).toContain('toolbarDensity="standard"')
    expect(source).not.toContain('Skeleton className="h-9 w-full rounded-lg sm:max-w-sm"')
  })
})
