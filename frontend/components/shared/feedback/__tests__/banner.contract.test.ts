import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/feedback/banner.tsx"), "utf8")

describe("index contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("function useControllableBannerState")
    expect(source).not.toContain("@radix-ui/react-use-controllable-state")
  })
})
