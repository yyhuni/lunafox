import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/editors/datetime-picker.tsx"), "utf8")

describe("datetime-picker contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function DateTimePicker")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })
})
