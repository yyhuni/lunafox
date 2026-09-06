import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(
  path.resolve(process.cwd(), "components/scan/initiate-scan-dialog-sections.tsx"),
  "utf8",
)

describe("initiate-scan-dialog-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function InitiateScanStepHeader")
    expect(source).toContain("export function getScanStepProgress")
    expect(source).toContain("className")
    expect(source).toContain('from "react"')
    expect(source).toContain('role="progressbar"')
    expect(source).toContain('bg-interaction-accent transition-[width]')
    expect(source).toContain('size-3 -translate-x-1/2 -translate-y-1/2 rounded-full bg-interaction-accent')
  })

  it("keeps scan drawer footer actions from submitting the surrounding form", () => {
    // All footer buttons must declare type="button" so they never trigger
    // the implicit form submit inside <FormDrawer>'s <form> wrapper.
    expect(source).toContain("currentStep === 1")
    const footerSource = source.slice(
      source.indexOf("interface InitiateScanFooterProps"),
      source.indexOf("interface InitiateScanOverwriteDialogProps"),
    )
    expect(footerSource).not.toContain("onCancel: () => void")
    expect(footerSource).not.toContain("onClick={onCancel}")
    expect(footerSource).toMatch(/<Button\s+type="button"[^>]*onClick=\{onBack\}/)
    expect(footerSource).toMatch(/<Button\s+type="button"[^>]*onClick=\{onNext\}/)
    expect(footerSource).toMatch(/type="button"[\s\S]*?onClick=\{onStart\}/)
  })

  it("allows callers with an outer workflow to show a previous-step action", () => {
    const footerSource = source.slice(
      source.indexOf("interface InitiateScanFooterProps"),
      source.indexOf("interface InitiateScanOverwriteDialogProps"),
    )
    expect(footerSource).toContain("showBackButton?: boolean")
    expect(footerSource).toContain("showBackButton ?? currentStep > 1")
    expect(footerSource).toContain("{shouldShowBackButton && (")
  })

  it("keeps footer navigation arrows consistent across scan steps", () => {
    const footerSource = source.slice(
      source.indexOf("interface InitiateScanFooterProps"),
      source.indexOf("interface InitiateScanOverwriteDialogProps"),
    )
    expect(footerSource).toContain("ChevronLeft")
    expect(footerSource).toContain("ChevronRight")
    expect(footerSource).toContain('<ChevronLeft className="size-4" />')
    expect(footerSource).toContain('<ChevronRight className="size-4" />')
  })

  it("keeps the first step as a searchable single-workflow selector", () => {
    expect(source).toContain("workflowListTitle")
    expect(source).toContain("<ScanSearchablePicker")
    expect(source).toContain("workflowSearchPlaceholder")
    expect(source).toContain("workflowSearchEmpty")
    expect(source).toContain("onWorkflowNamesChange([workflow.name])")
    expect(source).not.toContain("TabsTrigger")
    expect(source).not.toContain("presetWorkflows")
  })

  it("keeps non-default launch controls behind a collapsed execution-options disclosure", () => {
    const executionOptionsSource = source.slice(
      source.indexOf("interface InitiateScanExecutionOptionsProps"),
      source.indexOf("interface InitiateScanConfigStepProps"),
    )

    expect(source).toContain("export function InitiateScanExecutionOptions")
    expect(executionOptionsSource).toContain('defaultOpen={false}')
    expect(executionOptionsSource).toContain('t("executionOptions.title")')
    expect(executionOptionsSource).toContain("<ScanInputSourceSelector")
    expect(executionOptionsSource).toContain("<ScanAgentSelector")
    expect(executionOptionsSource).toContain("hover:bg-transparent")
    expect(executionOptionsSource).toContain("text-muted-foreground transition-colors group-hover:text-foreground")
    expect(executionOptionsSource).toContain("group-hover:text-foreground")
    expect(executionOptionsSource).toContain('data-[panel-open]:[&>svg]:rotate-90')
    expect(executionOptionsSource.indexOf("<ChevronRight")).toBeLessThan(
      executionOptionsSource.indexOf('t("executionOptions.title")'),
    )
  })

  it("shows workflow identity instead of configuration status badges", () => {
    expect(source).toContain("workflow.displayName || workflow.name")
    expect(source).toContain("workflow.description")
    expect(source).toContain("workflow.isExecutable")
    expect(source).toContain("workflowUnavailable")
    expect(source).not.toContain("workflowConfiguredHint")
    expect(source).not.toContain("workflowUnconfiguredHint")
    expect(source).not.toContain("workflowConfigured")
    expect(source).not.toContain("workflowUnconfigured")
  })

  it("keeps long workflow labels inside a bounded overlay", () => {
    expect(source).toContain('from "./scan-searchable-picker"')
    expect(source).toContain('className="min-h-12 gap-3 py-2"')
    expect(source).toContain('className="flex min-w-0 flex-1 items-center gap-2 overflow-hidden"')
    expect(source).toContain('className="min-w-0 flex-1 overflow-hidden"')
    expect(source).toContain('"block truncate", textRole.navLabel')
    expect(source).toContain('"block truncate", textRole.helperText')
  })

  it("keeps workflow choice icons neutral without a background container", () => {
    expect(source).toContain('className="flex size-8 shrink-0 items-center justify-center text-muted-foreground"')
    expect(source).not.toContain('bg-muted text-muted-foreground')
    expect(source).toContain('<Zap className="size-6" />')
    expect(source).not.toContain("size-7 shrink-0 items-center justify-center border bg-muted/30")
  })

  it("keeps edited state out of the config step summary row", () => {
    expect(source).not.toContain('t("configEdited")')
  })

  it("shows a focused message when every workflow Step is disabled", () => {
    const footerSource = source.slice(
      source.indexOf("interface InitiateScanFooterProps"),
      source.indexOf("interface InitiateScanOverwriteDialogProps"),
    )
    expect(footerSource).toContain("hasNoEnabledSteps: boolean")
    expect(footerSource).toContain('t("validation.noEnabledSteps")')
  })

  it("uses a workflow display name instead of exposing the resource value in the first-step footer", () => {
    const footerSource = source.slice(
      source.indexOf("interface InitiateScanFooterProps"),
      source.indexOf("interface InitiateScanOverwriteDialogProps"),
    )
    expect(footerSource).toContain("selectedWorkflowDisplayName?: string")
    expect(footerSource).toContain("selectedWorkflowDisplayName || selectedWorkflowNames[0]")
  })

  it("builds the engine config model from backend workflow steps", () => {
    expect(source).toContain('from "@/lib/engine-catalog"')
    expect(source).toContain("buildWorkflowWithEngineCatalog(selectedWorkflow, engineCatalogDetails, locale)")
    expect(source).toContain("isEngineCatalogError")
    expect(source).toContain("locale?: Locale")
    expect(source).toContain("selectedWorkflow?.steps?.length")
  })
})
