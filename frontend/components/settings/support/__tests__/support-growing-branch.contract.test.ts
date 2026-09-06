import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(
  path.resolve(process.cwd(), "components/settings/support/support-flying-birds.tsx"),
  "utf8",
)

describe("support growing branch contract", () => {
  it("loads the installed browser runtime instead of an unmanaged public vendor copy", () => {
    expect(source).toContain('import("p5/lib/p5.js")')
    expect(source).not.toContain('"/vendor/p5.min.js"')
  })

  it("adds a terminal flower after a branch completes its reveal", () => {
    expect(source).toContain("const terminalFlowerProbability = 0.22")
    expect(source).toContain("branch.segments.length === maxSegments + 1")
    expect(source).toContain("p.random() < terminalFlowerProbability")
    expect(source).toContain("revealStart + branch.revealDuration")
  })
})
