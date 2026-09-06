import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/auth/change-password-dialog.tsx"), "utf8")

describe("change-password-dialog contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function ChangePasswordDialog")
    expect(source).toContain("className")
    expect(source).toContain("from \"next-intl\"")
  })
})
