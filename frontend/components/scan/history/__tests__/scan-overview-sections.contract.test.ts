import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/history/scan-overview-sections.tsx"), "utf8")

describe("scan-overview-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function ScanOverviewHeader")
    expect(source).toContain("export function ScanOverviewRuntimeDetails")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("keeps route fallback skeletons out of the runtime sections module", () => {
    expect(source).not.toContain("export function ScanOverviewLoadingState")
    expect(source).not.toContain('from "@/components/scan/history/scan-overview-loading-state"')
    expect(source).not.toContain('from "@/components/ui/skeleton"')
  })

  it("uses shared tabs and scan status helpers", () => {
    expect(source).toContain("getScanStatusBadgeVariant")
    expect(source).toContain("getScanStatusClasses")
    expect(source).toContain("getStatusToneBgClass")
    expect(source).toContain('variant="content"')
    expect(source).not.toContain('variant="minimal-tab"')
    expect(source).not.toContain("var(--warning)")
    expect(source).not.toContain("var(--error)")
    expect(source).not.toContain("var(--info)")
    expect(source).not.toContain("animate-pulse bg-success")
  })

  it("shows the persisted input source in the scan overview header", () => {
    expect(source).toContain('t("inputSource.label")')
    expect(source).toContain('t(`inputSource.${scan.inputSource}`)')
  })

  it("reuses runtime detail panels for the page-level runtime workspace", () => {
    expect(source).toContain('from "@/components/scan/history/scan-runtime-detail-drawer"')
    expect(source).toContain("RuntimeDetailTabsPanel")
    expect(source).toContain("Separator")
    expect(source).toContain("RuntimeConfigurationPanel")
    expect(source).not.toContain('value="assets"')
    expect(source).not.toContain("runtimeDrawer.details.assets")
    expect(source).not.toContain("showTitle={false}")
    expect(source).not.toContain('value="debug"')
    expect(source).not.toContain("runtimeDrawer.details.debug")
  })

  it("renders runtime details as one integrated tab panel", () => {
    expect(source).toContain("<RuntimeDetailTabsPanel")
    expect(source).toContain("headerTrailing={")
    expect(source).not.toContain('from "@/components/shared/detail-drawer"')
    expect(source).not.toContain('className={cn(textRole.sectionTitle, "shrink-0")}')
    expect(source).not.toContain('className="hidden h-4 sm:block"')
    expect(source).not.toContain('<h3 className={textRole.sectionTitle}>{t("runtimeDrawer.details.title")}</h3>')
    expect(source).not.toContain('overflow-hidden rounded-lg border border-border/60">\n          <DetailDrawerTabsContent')
  })

  it("renders scan configuration as a read-only yaml code block instead of an editor", () => {
    expect(source).toContain("RuntimeConfigurationPanel")
    expect(source).toContain("serializeWorkflowConfiguration")
    expect(source).not.toContain("normalizeWorkflowConfiguration")
    expect(source).not.toContain("Collapsible")
    expect(source).not.toContain("runtimeDrawer.config.rawTitle")
    expect(source).not.toContain("runtimeDrawer.config.rawHint")
    expect(source).not.toContain("YamlEditor")
    expect(source).not.toContain("ScanOverviewEditorLoadingState")
    expect(source).not.toContain("@/components/shared/editors/yaml-editor")
  })

  it("keeps copy-log actions out of the page-level runtime detail tab header", () => {
    expect(source).not.toContain('t("runtimeDrawer.details.copyLogs")')
    expect(source).not.toContain("onCopyLogs")
    expect(source).not.toContain('disabled={!logText}')
    expect(source).not.toContain('<Copy className="h-3.5 w-3.5" />')
  })

  it("avoids layout-shifting hover motion", () => {
    expect(source).not.toContain("group-hover:translate-x")
  })
})
