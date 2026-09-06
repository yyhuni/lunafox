import { describe, expect, it } from "vitest"
import { existsSync, readFileSync } from "node:fs"
import path from "node:path"

const scriptNames = [
  "check-color-foundation.mjs",
  "check-spacing-foundation.mjs",
  "check-button-foundation.mjs",
  "check-component-foundation.mjs",
  "check-a11y-foundation.mjs",
  "check-motion-foundation.mjs",
  "check-loading-foundation.mjs",
  "check-ui-foundation.mjs",
] as const

function scriptSource(scriptName: string) {
  const scriptPath = path.resolve(process.cwd(), "scripts", scriptName)
  if (!existsSync(scriptPath)) {
    expect.fail(`scripts/${scriptName} is missing`)
  }
  return readFileSync(scriptPath, "utf8")
}

describe("UI foundation guardrail scripts", () => {
  it("provides dedicated scripts for every foundation guardrail family", () => {
    for (const scriptName of scriptNames) {
      expect(existsSync(path.resolve(process.cwd(), "scripts", scriptName))).toBe(true)
    }
  })

  it("supports inventory and verify modes with shared exception ledger input", () => {
    for (const scriptName of scriptNames) {
      const source = scriptSource(scriptName)

      expect(source).toContain("\"inventory\"")
      expect(source).toContain("\"verify\"")
      expect(source).toContain("foundation-exceptions.json")
      expect(source).toContain("components")
      expect(source).toContain("app")
    }
  })

  it("keeps color guardrail coverage broad enough for production color drift", () => {
    const source = scriptSource("check-color-foundation.mjs")

    expect(source).toContain('scanRoots: ["app", "components", "hooks", "lib"]')
    expect(source).toContain("lib/color-themes.ts")
    expect(source).toContain("(?:bg|text|border|from|via|to|ring|fill|stroke|decoration|shadow)")
    expect(source).toContain("red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue")
    expect(source).toContain("from|via|to|ring|fill|stroke|decoration|shadow")
    expect(source).toContain("(?:white|black)(?:")
    expect(source).not.toContain("components/pixel-blast.tsx")
    expect(source).not.toContain("components/shared/visualization/")
  })

  it("locks expected checks for each guardrail family", () => {
    expect(scriptSource("check-color-foundation.mjs")).toContain("raw-color-literal")
    expect(scriptSource("check-color-foundation.mjs")).toContain("raw-color-function")
    expect(scriptSource("check-color-foundation.mjs")).toContain("theme-locked-tailwind-color")
    expect(scriptSource("check-color-foundation.mjs")).toContain("components/overview/agent-globe-card.tsx")
    expect(scriptSource("check-spacing-foundation.mjs")).toContain("arbitrary-spacing-or-size")
    expect(scriptSource("check-spacing-foundation.mjs")).toContain("important-utility")
    expect(scriptSource("check-button-foundation.mjs")).toContain("native-button-with-classes")
    expect(scriptSource("check-button-foundation.mjs")).toContain("<button\\b(?=[^>]*\\bclassName=)[^>]*>")
    expect(scriptSource("check-button-foundation.mjs")).toContain("adhoc-button-size")
    expect(scriptSource("check-component-foundation.mjs")).toContain("icon-park\\/react")
    expect(scriptSource("check-component-foundation.mjs")).toContain("file.startsWith(\"components/icons/\")")
    expect(scriptSource("check-component-foundation.mjs")).toContain("components/icons")
    expect(scriptSource("check-component-foundation.mjs")).toContain("raw-stable-concept-icon-import")
    expect(scriptSource("check-component-foundation.mjs")).toContain("rawStableConceptIconImportPattern")
    expect(scriptSource("check-component-foundation.mjs")).toContain('scanRoots: ["app", "components", "hooks", "lib"]')
    expect(scriptSource("check-component-foundation.mjs")).toContain("excluded-route-import")
    expect(scriptSource("check-a11y-foundation.mjs")).toContain("icon-only-button-without-name")
    expect(scriptSource("check-a11y-foundation.mjs")).toContain("focus-outline-removal")
    expect(scriptSource("check-motion-foundation.mjs")).toContain("layout-shifting-hover-motion")
    expect(scriptSource("check-motion-foundation.mjs")).toContain("prefers-reduced-motion")
    expect(scriptSource("check-motion-foundation.mjs")).toContain("app/layout.tsx")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("direct-route-progress-engine-import")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("legacy-loading-wrapper-import")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("missing-loading-contract-prop")
    expect(scriptSource("check-loading-foundation.mjs")).toContain('allowOwnerlessProp: "nested"')
    expect(scriptSource("check-loading-foundation.mjs")).toContain("contract.allowOwnerlessProp")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("isContentHandoffSkeletonChild")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("duplicate-loading-owner")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("hard-replacement-loading-handoff")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("route-local-button-skeleton-helper")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("route-local-control-shell-lookalike")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("route-local-table-skeleton-replica")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("business-skeleton-declaration")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("legacy-skeleton-row-count-name")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("isApprovedSkeletonDeclaration")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("extractLoadingOwnerSourceRanges")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("collectMatchesInRanges")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("data-loading-controlled-table-skeleton-variant")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("component-skeleton-legacy-breadcrumb")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("lazy-page-visible-default-fallback")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("route-lazy-page-visible-fallback")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("route-dynamic-visible-fallback")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("reviewedVisibleDynamicFallbacks")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("unreviewed-visible-dynamic-fallback")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("HeaderIconPlaceholder")
    expect(scriptSource("check-loading-foundation.mjs")).not.toContain("NucleiRepoContentLoadingState")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("WorkflowConfigPreviewLoadingState")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("ArchitectureFlowCanvasLoadingState")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("content-handoff-missing-wrapper-geometry")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("loading-route-contract-missing")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("loading-geometry-contract-missing")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("loading-geometry-owner-missing")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("loading-geometry-slot-source-missing")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("loading-geometry-contract-invalid")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("loading-geometry-not-applicable-reason-missing")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("loading-geometry-owner-disposition-conflict")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("loading-geometry-owner-exclusion-reason-missing")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("loading-geometry-prepared-surface-removed")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("loading-geometry-${error}")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("loading-geometry-bounded-exception-owner-unpaired")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("getLoadingGeometryInventoryValidationErrors")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("approvedLoadingStructureSlotTemplates")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("extractContentHandoffSourceRanges")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("orphan-loading-structure-slot")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("data-loading-slot declared outside ContentHandoff")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("getLoadingStructureSlotAttributes\\s*\\(")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("routeLoadingGeometryContractFindings")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("splitTopLevelArguments")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("findBalancedGroupEnd")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("findJsxOpeningTagEnd")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("getJsxOpeningTagStaticProp")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("extractJsxOpeningTagStaticProps")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("braceDepth")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("const className = getJsxOpeningTagStaticProp")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("\"contentClassName\"")
    expect(scriptSource("check-loading-foundation.mjs")).toContain("unclassified loading snippets")
  })

  it("aggregates the individual guardrails and existing UI boundary checks", () => {
    const source = scriptSource("check-ui-foundation.mjs")

    expect(source).toContain("check-typography-foundation.mjs")
    expect(source).toContain("check-color-foundation.mjs")
    expect(source).toContain("check-spacing-foundation.mjs")
    expect(source).toContain("check-button-foundation.mjs")
    expect(source).toContain("check-component-foundation.mjs")
    expect(source).toContain("check-a11y-foundation.mjs")
    expect(source).toContain("check-motion-foundation.mjs")
    expect(source).toContain("check-loading-foundation.mjs")
    expect(source).toContain("check-ui-boundary.mjs")
  })

  it("keeps the UI README as an allowed near-code foundation entrypoint", () => {
    const source = scriptSource("check-ui-boundary.mjs")

    expect(source).toContain("README.md")
  })
})
