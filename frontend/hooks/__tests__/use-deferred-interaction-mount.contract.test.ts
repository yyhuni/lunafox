import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "hooks/use-deferred-interaction-mount.ts"), "utf8")

describe("use-deferred-interaction-mount contract", () => {
  it("keeps sidebar lifecycle controls centralized in the shared hook", () => {
    expect(source).toContain("export const deferredInteractionUnmountDelayMs")
    expect(source).toContain("preloadWhen")
    expect(source).toContain("unmountDelayMs")
    expect(source).toContain("window.setTimeout")
    expect(source).toContain("window.clearTimeout")
  })
})
