import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/feedback/confirm-dialog-state.ts"), "utf8")

describe("confirm-dialog-state contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useConfirmDialogState")
    expect(source).toContain("from \"next-intl\"")
    expect(source).toContain("processingText?: string")
    expect(source).toContain('processingText || t("processing")')
  })
})
