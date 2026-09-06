import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/organization/targets/add-target-dialog.tsx"), "utf8")

describe("add-target-dialog contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("from \"./link-target-dialog\"")
  })
})
