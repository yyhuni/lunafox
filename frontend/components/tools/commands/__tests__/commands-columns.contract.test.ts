import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/tools/commands/commands-columns.tsx"), "utf8")

describe("commands-columns contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function createCommandColumns")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("routes the commands row action menu through the shared dense row owner", () => {
    expect(source).toContain("DenseRowActionMenu")
    expect(source).toContain("ariaLabel={t.actions.openMenu}")
    expect(source).toContain('from "@/components/shared/feedback/copy-button"')
    expect(source).toContain("copyTextToClipboard(command.commandTemplate)")
    expect(source).not.toContain("navigator.clipboard.writeText(command.commandTemplate)")
  })
})
