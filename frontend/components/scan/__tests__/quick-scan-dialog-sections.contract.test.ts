import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/quick-scan-dialog-sections.tsx"), "utf8")

describe("quick-scan-dialog-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function QuickScanHeader")
    expect(source).toContain("export function QuickScanTargetStep")
    expect(source).toContain("export function QuickScanFooter")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("does not use important layout utilities in the quick scan footer", () => {
    expect(source).not.toContain("!flex")
    expect(source).not.toContain("!items-center")
    expect(source).not.toContain("!justify-between")
  })

  it("uses drawer title semantics for the quick scan workbench shell", () => {
    expect(source).toContain("DrawerHeader")
    expect(source).toContain("DrawerTitle")
    expect(source).toContain("DrawerDescription")
    expect(source).toContain("EdgePanelHeader")
    expect(source).toContain('variant="workbench"')
    expect(source).toContain('<Zap className="size-7" />')
    expect(source).toContain('<DrawerTitle className={textRole.sectionTitle}>')
    expect(source).toContain('cn("max-w-2xl", textRole.helperText)')
    expect(source).toContain("stepHeader: React.ReactNode")
    expect(source).toContain("progress={stepHeader}")
    expect(source).not.toContain('from "@/components/ui/dialog"')
    expect(source).not.toContain('from "@/components/ui/alert-dialog"')
    expect(source).not.toContain("<DialogHeader")
    expect(source).not.toContain("<DialogTitle")
    expect(source).not.toContain("<DialogDescription")
    expect(source).not.toContain("QuickScanOverwriteDialog")
    expect(source).not.toContain("QuickScanTrigger")
  })

  it("reuses the add-target bulk input shell for the quick scan target step", () => {
    expect(source).toContain("AddTargetInputSection")
    expect(source).toContain("tTargetDialog")
    expect(source).toContain("formTargets={targetInput}")
    expect(source).toContain("targetCount={validCount}")
    expect(source).toContain("textareaRef={textareaRef}")
    expect(source).not.toContain('from "@/components/ui/textarea"')
    expect(source).not.toContain("<Textarea")
  })

  it("keeps quick scan footer actions from submitting the surrounding drawer form", () => {
    expect(source).toMatch(/<Button\s+type="button"[^>]*variant="outline"[^>]*onClick=\{onBack\}/)
    expect(source).toMatch(/<Button\s+type="button"[\s\S]*?onClick=\{onNext\}/)
    expect(source).toMatch(/<Button\s+type="button"[\s\S]*?onClick=\{onSubmit\}/)
  })

  it("uses the same compact footer action density as later scan steps", () => {
    expect(source).toMatch(/<Button\s+type="button"\s+variant="outline"\s+size="sm"[^>]*onClick=\{onBack\}/)
    expect(source).toMatch(/<Button\s+type="button"\s+size="sm"[\s\S]*?onClick=\{onNext\}/)
    expect(source).toMatch(/<Button\s+type="button"\s+size="sm"[\s\S]*?onClick=\{onSubmit\}/)
  })

  it("uses the canonical local-dismiss icon for the drawer close control", () => {
    expect(source).toContain("semanticIcons.action.cancel")
    expect(source).not.toContain("XIcon")
  })

  it("avoids layout-shifting trigger hover motion", () => {
    expect(source).not.toContain("group-hover:translate-x")
  })
})
