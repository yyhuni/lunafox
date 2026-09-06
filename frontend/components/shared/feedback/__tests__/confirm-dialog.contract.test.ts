import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/feedback/confirm-dialog.tsx"), "utf8")

describe("confirm-dialog contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function ConfirmDialog")
    expect(source).toContain("from \"@/components/shared/feedback/confirm-dialog-sections\"")
    expect(source).toContain("processingText?: string")
    expect(source).toContain("processingText,")
  })
})
