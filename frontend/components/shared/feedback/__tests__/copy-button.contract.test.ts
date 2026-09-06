import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/feedback/copy-button.tsx"), "utf8")

describe("copy-button contract", () => {
  it("owns the shared copy affordance style and clipboard fallback", () => {
    expect(source).toContain("export function CopyButton")
    expect(source).toContain("export async function copyTextToClipboard")
    expect(source).toContain("COPY_BUTTON_BASE_CLASSNAME")
    expect(source).toContain("COPY_BUTTON_HOVER_REVEAL_CLASSNAME")
    expect(source).toContain('variant="ghost"')
    expect(source).toContain('size={size}')
    expect(source).toContain("window.isSecureContext")
    expect(source).toContain("document.execCommand(\"copy\")")
    expect(source).toContain("toastFeedback.success")
    expect(source).toContain("toastFeedback.error")
  })
})
