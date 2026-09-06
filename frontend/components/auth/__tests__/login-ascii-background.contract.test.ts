import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/auth/login-ascii-background.tsx"), "utf8")

describe("login-ascii-background contract", () => {
  it("keeps the animated login pixel field sparse enough to avoid dense blocks", () => {
    expect(source).toContain("const PARTICLE_ALPHA = 0.32")
    expect(source).toContain("const PARTICLE_SIZE_RATIO = 0.48")
    expect(source).toContain("const cell = width < 720 ? 10 : 14")
    expect(source).toContain("threshold: 0.44 + hash2(x + 7, y - 5) * 0.22")
  })
})
