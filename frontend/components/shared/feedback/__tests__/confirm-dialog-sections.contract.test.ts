import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/feedback/confirm-dialog-sections.tsx"), "utf8")

describe("confirm-dialog-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function ConfirmDialogLayout")
    expect(source).toContain("from \"@/components/ui/alert-dialog\"")
    expect(source).toContain('variant === "destructive" ? "dark:bg-destructive" : undefined')
    expect(source).toContain("loadingLabel={state.processingLabel}")
  })
})
