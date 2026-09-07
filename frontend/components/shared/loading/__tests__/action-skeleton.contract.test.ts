import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/loading/action-skeleton.tsx"), "utf8")

describe("action-skeleton contract", () => {
  it("exports a shared structural action placeholder primitive", () => {
    expect(source).toContain("export function ActionSkeleton")
    expect(source).toContain('data-slot="action-skeleton"')
    expect(source).toContain('aria-hidden="true"')
    expect(source).toContain('type ActionSkeletonSize = ButtonStructuralSize')
    expect(source).toContain('type ActionSkeletonEmphasis = "outline" | "primary"')
    expect(source).toContain('from "@/lib/ui/button-size-contract"')
    expect(source).toContain("radius-control")
  })

  it("stays structural instead of nesting skeleton detail children", () => {
    expect(source).not.toContain('from "@/components/ui/skeleton"')
    expect(source).not.toContain("<Skeleton")
    expect(source).toContain("loading-skeleton")
    expect(source).toContain('data-action-emphasis={emphasis}')
    expect(source).toContain("buttonStructuralSizeClassNames[size]")
    expect(source).toContain("buttonIconOnlySizes.has(size)")
    expect(source).not.toContain("bg-primary")
    expect(source).not.toContain("text-primary-foreground")
  })
})
