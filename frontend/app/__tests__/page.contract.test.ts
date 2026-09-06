import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "app/page.tsx"), "utf8")

describe("page contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export default function Home")
    expect(source).toContain('from "next/navigation"')
  })
})
