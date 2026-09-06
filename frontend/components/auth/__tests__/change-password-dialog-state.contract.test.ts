import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/auth/change-password-dialog-state.ts"), "utf8")

describe("change-password-dialog-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useChangePasswordDialogState")
    expect(source).toContain("from \"react\"")
  })
})
