import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/settings/database-health/database-health-view.tsx"), "utf8")
const loadingStateSource = readFileSync(
  path.resolve(process.cwd(), "components/settings/database-health/database-health-loading-state.tsx"),
  "utf8"
)

describe("database-health-view contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function DatabaseHealthView")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("owns the database health loading state for initial query loading", () => {
    expect(source).toContain("export interface DatabaseHealthViewProps")
    expect(source).toContain("DatabaseHealthLoadingState")
    expect(source).not.toContain("export function DatabaseHealthLoadingState")
    expect(loadingStateSource).toContain("export function DatabaseHealthLoadingState")
    expect(source).toContain("pageTitle: string")
    expect(source).toContain("pageDescription: string")
    expect(source).toContain("onReady?: () => void")
    expect(source).toContain("deferInitialSkeleton?: boolean")
    expect(source).toContain("onReady,")
    expect(source).toContain("deferInitialSkeleton = false")
    expect(source).toContain("onReady?.()")
    expect(source).toContain("DatabaseHealthLoadingState")
    expect(source).toContain('owner="database-health-page"')
    expect(source).toContain("if (loading && !deferInitialSkeleton)")
    expect(source).toContain("if (loading) return null")
    expect(loadingStateSource).toContain('data-slot="database-health-loading-state"')
    expect(source).toContain('<PageHeader code="DBH-01" title={pageTitle} description={pageDescription} />')
    expect(source).not.toContain('useTranslations("pages.databaseHealth")')
    expect(loadingStateSource).toContain("DatabaseHealthLoadingState requires a non-empty owner.")
    expect(source).not.toContain("DatabaseHealthPageSkeleton")
    expect(source).not.toContain("database-health-page-skeleton")
    expect(source).not.toContain('{loading ? (')
  })

  it("keeps the visual database health loading state owner optional", () => {
    expect(loadingStateSource).toContain("owner?: string")
    expect(loadingStateSource).toContain("if (owner !== undefined && !owner.trim())")
    expect(loadingStateSource).toContain('owner ? getLoadingOwnerAttributes({ owner, layer: "route", intent: "route" }) : {}')
    expect(source).toContain('owner="database-health-page"')
  })

  it("uses shared page and section shell primitives", () => {
    expect(source).toContain("PageHeader")
    expect(source).toContain("CardHeader")
    expect(source).toContain("CardTitle")
    expect(source).toContain("TableHeader")
    expect(source).not.toContain("bg-transparent")
  })

  it("uses shared status tone helpers instead of raw var utilities", () => {
    expect(source).toContain("getStatusToneBadgeClass")
    expect(source).toContain("getStatusToneTextClass")
    expect(source).not.toContain("text-[var(--success)]")
    expect(source).not.toContain("text-[var(--warning)]")
    expect(source).not.toContain("text-[var(--error)]")
    expect(source).not.toContain("bg-[var(--success)]")
    expect(source).not.toContain("bg-[var(--warning)]")
    expect(source).not.toContain("bg-[var(--error)]")
    expect(source).not.toContain("border-[var(--warning)]")
  })

  it("renders only backend-backed MVP database health sections", () => {
    expect(source).toContain('t("sections.snapshot")')
    expect(source).toContain('t("sections.core")')
    expect(source).toContain('t("sections.optional")')
    expect(source).toContain('t("findings.title")')
    expect(source).toContain('t("alerts.title")')
    expect(source).toContain("databaseSizeBytes")
    expect(source).toContain("formatBytes")
    expect(source).toContain("findings.map")
    expect(source).not.toContain('t("overview.region")')
    expect(source).not.toContain("data?.region")
    expect(source).not.toContain("buildDatabaseHealthFindings")
    expect(source).not.toContain("getHighestStatus")
    expect(source).not.toContain("statusFromThreshold(data")
    expect(source).not.toContain("databaseName")
    expect(source).not.toContain("storageUsedBytes")
    expect(source).not.toContain("storageTotalBytes")
    expect(source).not.toContain("historicalEvents")
    expect(source).not.toContain("activeEvents")
    expect(source).not.toContain("riskSummary")
    expect(source).not.toContain("diagnosis.")
    expect(source).not.toContain("formatStorageSummary")
  })

  it("localizes known backend database health diagnosis and alert copy", () => {
    expect(source).toContain("translateDatabaseHealthFinding")
    expect(source).toContain("translateDatabaseHealthAlert")
    expect(source).toContain('t("backendFindings.oldestPendingTaskAgeSec.title")')
    expect(source).toContain('t("backendFindings.connectionUsagePercent.title")')
    expect(source).toContain('t("backendAlerts.establishedConnectionCountHigh.title")')
    expect(source).toContain('t("backendAlerts.taskBacklogAgeHigh.title")')
    expect(source).toContain('t("backendAlerts.taskBacklogAgeCritical.description")')
  })

  it("formats duration values with locale-aware readable units", () => {
    expect(source).toContain("formatDurationSeconds")
    expect(source).toContain("formatDurationUnit")
    expect(source).toContain("unitDisplay: \"narrow\"")
    expect(source).not.toContain("return `${minutes}m`")
  })

  it("does not render secondary header descriptions for core metrics", () => {
    expect(source).not.toContain("{t(\"overview.lastCheck\")}: {formatDateTime(data?.observedAt)}")
    expect(source).not.toContain("t(\"sections.coreSignalDesc\")")
  })

  it("keeps auto-check cadence in the snapshot status header instead of the metadata grid", () => {
    const contextItemsStart = source.indexOf("const contextItems")
    const contextItemsEnd = source.indexOf("const translatedAlerts", contextItemsStart)
    const contextItemsSource = source.slice(contextItemsStart, contextItemsEnd)
    const snapshotHeaderStart = source.indexOf("<CardHeader className={DATABASE_HEALTH_SNAPSHOT_HEADER_CLASS}>")
    const snapshotHeaderEnd = source.indexOf("</CardHeader>", snapshotHeaderStart)
    const snapshotHeaderSource = source.slice(snapshotHeaderStart, snapshotHeaderEnd)

    expect(contextItemsSource).not.toContain('t("instance.autoCheck")')
    expect(contextItemsSource).not.toContain("AUTO_CHECK_INTERVAL_SECONDS")
    expect(snapshotHeaderSource).toContain('t("instance.autoCheck")')
    expect(snapshotHeaderSource).toContain("AUTO_CHECK_INTERVAL_SECONDS")
  })

  it("uses compact header rhythm for title-only MVP sections", () => {
    expect(source).toContain("DATABASE_HEALTH_SECTION_HEADER_CLASS")
    expect(source).toContain("DATABASE_HEALTH_SECTION_BODY_FLUSH_CLASS")
    expect(source).toContain("DATABASE_HEALTH_SECTION_HEADER_WITH_DESCRIPTION_CLASS")
    expect(source).not.toContain('className="p-0"')
    expect(source).not.toContain('className="border-b px-6 py-4"')
    expect(source).toMatch(
      /<CardHeader className=\{DATABASE_HEALTH_SECTION_HEADER_CLASS\}>\s*<CardTitle>{t\("sections\.core"\)}<\/CardTitle>\s*<\/CardHeader>/
    )
    expect(source).toMatch(
      /<CardHeader className=\{DATABASE_HEALTH_SECTION_HEADER_CLASS\}>\s*<CardTitle>{t\("sections\.optional"\)}<\/CardTitle>\s*<\/CardHeader>/
    )
  })
})
