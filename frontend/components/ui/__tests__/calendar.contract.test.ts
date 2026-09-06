import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ui/calendar.tsx"), "utf8")

describe("calendar contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses token-driven rounded range styling instead of hard square range segments", () => {
    expect(source).toContain("data-[range-middle=true]:rounded-[calc(var(--radius)*0.75)]")
    expect(source).not.toContain("data-[range-middle=true]:rounded-none")
  })
})
