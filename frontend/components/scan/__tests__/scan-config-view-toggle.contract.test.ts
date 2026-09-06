import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(
  path.resolve(process.cwd(), "components/scan/scan-config-view-toggle.tsx"),
  "utf8"
)

describe("scan-config-view-toggle contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export const ScanConfigViewToggle = React.forwardRef")
    expect(source).toContain("<ScanConfigEditor")
  })

  it("owns one complete Catalog state and passes it to every resource field", () => {
    expect(source).toContain("const wordlistCatalog = useCompleteWordlistCatalogState()")
    expect(source).toContain("wordlistCatalog={wordlistCatalog}")
  })

  it("keeps advanced controls and both modes inside one shared content scroll region", () => {
    expect(source).toContain('<ScrollArea className={cn("min-h-0 flex-1", className)} contentClassName="!min-w-0 w-full">')
    expect(source).toContain('className="flex min-h-full flex-col gap-3"')
    expect(source).toContain('className="radius-surface flex flex-wrap items-center justify-between gap-3 border bg-muted/20 px-4 py-3"')
    expect(source).toContain('className="min-h-0 flex-1"')
    expect(source).toContain("showLabel={false}")
    expect(source).not.toContain('<ScrollArea className="h-full" contentClassName="!min-w-0 w-full">')
    expect(source).not.toContain('radius-surface min-h-0 flex-1 overflow-hidden border border-border/60 bg-card')
  })

  it("serializes current form values before switching into yaml mode", () => {
    expect(source).toContain("const handleViewModeChange = useCallback")
    expect(source).toContain("const yamlStr = serializeFormToYaml(formValues)")
    expect(source).toContain("if (yamlStr !== configuration)")
    expect(source).toContain("onSync?: (value: string) => void")
    expect(source).toContain("const applySyncedConfig = onSync ?? onChange")
    expect(source).toContain("onCheckedChange={handleViewModeChange}")
    expect(source).not.toContain("lastValid")
  })

  it("allows reset to restore the parent workflow baseline", () => {
    expect(source).toContain("onReset?: () => void")
    expect(source).toContain("if (onReset)")
    expect(source).toContain("onReset()")
  })

  it("shows edited state beside the engine configuration heading", () => {
    expect(source).toContain('className="flex items-center justify-between gap-3"')
    expect(source).toContain('t("steps.engineConfig")')
    expect(source).toContain('t("configEdited")')
    expect(source).toContain('inline-flex shrink-0 items-center gap-1 text-[11px] text-warning')
  })
})
