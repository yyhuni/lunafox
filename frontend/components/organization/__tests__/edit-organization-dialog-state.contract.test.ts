import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/organization/edit-organization-dialog-state.ts"), "utf8")

describe("edit-organization-dialog-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useEditOrganizationDialogState")
    expect(source).toContain("from \"react\"")
  })
})
