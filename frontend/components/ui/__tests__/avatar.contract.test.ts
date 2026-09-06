import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ui/avatar.tsx"), "utf8")

describe("avatar contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
    expect(source).toContain("radius-round")
  })

  it("uses Base UI Avatar as the shared primitive baseline", () => {
    expect(source).toContain('from "@base-ui/react/avatar"')
    expect(source).toContain("AvatarPrimitive.Root")
    expect(source).toContain("AvatarPrimitive.Image")
    expect(source).toContain("AvatarPrimitive.Fallback")
    expect(source).not.toContain("@radix-ui/react-avatar")
  })
})
