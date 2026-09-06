import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/target/target-settings-sections.tsx"), "utf8")
const layoutSource = readFileSync(path.resolve(process.cwd(), "components/target/target-settings-layout.ts"), "utf8")

describe("target-settings-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function TargetSettingsRouteFallback")
    expect(source).toContain("export function TargetSettingsLoadingState")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
    expect(source).toContain("rowCount")
  })

  it("uses the real scheduled scan table without a redundant settings card", () => {
    expect(source).toContain("BlacklistSettingsLoadingState")
    expect(source).toContain("section?: TargetSettingsSection")
    expect(source).toContain('<BlacklistSettingsLoadingState embedded scope="target" />')
    expect(source).toContain("state: TargetSettingsState")
    expect(source).toContain("<ScheduledScanDataTable")
    expect(source).toContain("loadingRowCount={rowCount}")
    expect(source).toContain("data={[]}")
    expect(source).toContain("const columns = createScheduledScanColumns")
    expect(source).toContain('accessorKey !== "scanMode"')
    expect(source).toContain("columns={columns}")
    expect(source).toContain("showQuickFilters={false}")
    expect(source).toContain("initialLoading")
    expect(source).not.toContain("TargetSettingsSkeleton")
    expect(source).not.toContain("target-settings-skeleton")
    expect(source).not.toMatch(/\bDataTableSkeleton\b/)
    expect(source).toContain('owner: "target-settings-inline-table"')
    expect(source).toContain("getDataTableSkeletonRowCount(state.pageSize)")
    expect(source).not.toContain('Skeleton className="h-48 w-full"')
    expect(source).not.toContain("TargetSettingsBlacklistCardLoadingState")
    expect(source).not.toContain("<Card")
    expect(source).not.toContain("<CardHeader")
    expect(source).not.toContain("<CardContent")
    expect(source).not.toContain("<CardTitle")
    expect(source).not.toContain("<CardDescription")
    expect(source).not.toContain("scheduledScans.title")
    expect(source).not.toContain("scheduledScans.description")
  })

  it("derives the resolved settings content and skeleton geometry from a shared layout contract", () => {
    const sharedLayoutConstants = [
      "TARGET_SETTINGS_SHELL_CLASS",
    ]

    expect(source).toContain('from "./target-settings-layout"')
    for (const constant of sharedLayoutConstants) {
      expect(layoutSource).toContain(`export const ${constant}`)
      expect(source).toContain(constant)
    }

    expect(source).not.toContain('className="space-y-6"')
    expect(source).not.toContain("TARGET_SETTINGS_SKELETON_TABLE_COLUMN_COUNT")
    expect(source).toContain('section = "blacklist"')
    expect(source).toContain('section === "blacklist"')
    expect(source).not.toContain("TARGET_SETTINGS_BLACKLIST_CONTENT_CLASS")
    expect(source).not.toContain("TargetSettingsBlacklistCardSkeleton")
  })

  it("delegates blacklist loading geometry to the shared Target-local workbench", () => {
    expect(source).toContain('from "@/components/settings/blacklist/blacklist-settings-loading-state"')
    expect(source).toContain("BlacklistSettingsLoadingState")
    expect(source).toContain('scope="target"')
    expect(source).not.toContain("useTargetBlacklist")
    expect(source).not.toContain("useUpdateTargetBlacklist")
  })

  it("does not retain fixed-column route fallback geometry beside the scheduled scan table", () => {
    expect(layoutSource).not.toContain("TARGET_SETTINGS_TABLE_COLUMN_COUNT")
    expect(layoutSource).not.toContain("TARGET_SETTINGS_TABLE_TOOLBAR_BUTTON_COUNT")
    expect(source).toContain("TARGET_SETTINGS_ROUTE_FALLBACK_PAGE_SIZE")
    expect(source).toContain("pageSize={TARGET_SETTINGS_ROUTE_FALLBACK_PAGE_SIZE}")
  })

  it("keeps scheduled scans data loading on the resolved scheduled scan table geometry", () => {
    expect(source).toContain('from "@/components/scan/scheduled/scheduled-scan-data-table"')
    expect(source).toContain("function TargetSettingsScheduledScansTableLoadingState")
    expect(source).toContain('data-slot="target-settings-scheduled-scans-table-loading-state"')
    expect(source).toContain('owner: "target-settings-inline-table"')
    expect(source).toContain("data={[]}")
    expect(source).toContain("columns={state.columns}")
    expect(source).toContain("loadingRowCount={getDataTableSkeletonRowCount(state.pageSize)}")
    expect(source).toContain("showQuickFilters={false}")
    expect(source).not.toContain("const ScheduledScanDataTable = dynamic")
    expect(source).not.toContain('owner="target-settings-scans"')
    expect(source).not.toContain("loading: () => <DataTableSkeleton")
    expect(source).not.toContain("function TargetSettingsScheduledScansTableSkeleton")
    expect(source).not.toContain('data-slot="target-settings-scheduled-scans-table-skeleton"')
    expect(source).not.toContain("TargetSettingsScheduledScansTableSkeleton rowCount")
  })

  it("keeps the state-less route fallback toolbar inert instead of wiring no-op actions", () => {
    const routeFallbackSource = source.slice(
      source.indexOf("export function TargetSettingsRouteFallback"),
      source.indexOf("export function TargetSettingsLoadingState")
    )

    expect(routeFallbackSource).toContain("initialLoading")
    expect(routeFallbackSource).not.toContain("onAddNew={() => {}}")
    expect(routeFallbackSource).not.toContain("onSearch={() => {}}")
    expect(routeFallbackSource).not.toContain("onPageChange={() => {}}")
    expect(routeFallbackSource).not.toContain("onPageSizeChange={() => {}}")
  })

  it("does not show a dialog-shaped loading state before opening the create scheduled scan sheet", () => {
    expect(source).not.toContain('owner="target-settings-create-dialog-loading"')
    expect(source).toContain('owner="target-settings-edit-dialog-loading"')
  })

  it("releases scheduled scan create and edit sidebars after close", () => {
    expect(source).toContain("deferredInteractionUnmountDelayMs")
    expect(source).toContain("useDeferredInteractionMount")
    expect(source).toContain("shouldMountCreateDialog")
    expect(source).toContain("shouldMountEditDialog")
  })
})
