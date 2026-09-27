import { describe, expect, it } from "vitest"
import { compactSurfaceClassNames, compactSingleLineValueHeight } from "@/lib/ui/compact-surface-contract"

describe("compact surface contract", () => {
  it("keeps ordinary value surfaces on the shared 32px geometry", () => {
    expect(compactSurfaceClassNames.value).toContain("h-8")
    expect(compactSurfaceClassNames.value).toContain("min-h-8")
    expect(compactSurfaceClassNames.value).toContain("radius-control")
    expect(compactSurfaceClassNames.value).toContain("border")
    expect(compactSingleLineValueHeight).toBe("h-8")
  })

  it("keeps ordinary info surfaces on the shared border and inset contract", () => {
    for (const surface of [compactSurfaceClassNames.info, compactSurfaceClassNames.mutedInfo]) {
      expect(surface).toContain("radius-control")
      expect(surface).toContain("border")
      expect(surface).toContain("px-3")
      expect(surface).toContain("py-2")
    }
  })
})
