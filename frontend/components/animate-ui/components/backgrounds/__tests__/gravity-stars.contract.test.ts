import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/animate-ui/components/backgrounds/gravity-stars.tsx"), "utf8")

describe("gravity-stars contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from 'react'")
  })
})
