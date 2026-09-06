import { describe, expect, it } from "vitest"
import { existsSync, readFileSync } from "node:fs"
import path from "node:path"

const viewPath = path.resolve(process.cwd(), "components/settings/database-health/database-health-view.tsx")
const loadingStatePath = path.resolve(process.cwd(), "components/settings/database-health/database-health-loading-state.tsx")
const layoutPath = path.resolve(process.cwd(), "components/settings/database-health/database-health-layout.ts")
const legacySkeletonPath = path.resolve(process.cwd(), "components/settings/database-health/database-health-page-skeleton.tsx")

const viewSource = readFileSync(viewPath, "utf8")
const loadingStateSource = readFileSync(loadingStatePath, "utf8")
const layoutSource = readFileSync(layoutPath, "utf8")

describe("database-health-loading-state contract", () => {
  it("keeps the visual loading state in a lightweight module reused by the resolved view", () => {
    expect(existsSync(legacySkeletonPath)).toBe(false)
    expect(viewSource).toContain('from "@/components/settings/database-health/database-health-loading-state"')
    expect(viewSource).not.toContain("export function DatabaseHealthLoadingState")
    expect(loadingStateSource).toContain("export function DatabaseHealthLoadingState")
    expect(loadingStateSource).toContain("DatabaseHealthLoadingState requires a non-empty owner.")
    expect(loadingStateSource).toContain("getLoadingOwnerAttributes")
    expect(loadingStateSource).toContain('owner ? getLoadingOwnerAttributes({ owner, layer: "route", intent: "route" }) : {}')
    expect(loadingStateSource).toContain('data-slot="database-health-loading-state"')
    expect(loadingStateSource).toContain("PageHeader")
    expect(loadingStateSource).toContain("CardHeader")
    expect(loadingStateSource).toContain("TableHeader")
    expect(loadingStateSource).toContain("TABLE_DENSE_ROW_CLASS")
    expect(loadingStateSource).not.toContain("useDatabaseHealth")
    expect(loadingStateSource).not.toContain("database-health-view")
  })

  it("derives resolved and loading geometry from the shared layout contract", () => {
    const sharedLayoutConstants = [
      "DATABASE_HEALTH_PAGE_SHELL_CLASS",
      "DATABASE_HEALTH_CONTENT_SHELL_CLASS",
      "DATABASE_HEALTH_SECTION_PANEL_CLASS",
      "DATABASE_HEALTH_SNAPSHOT_HEADER_CLASS",
      "DATABASE_HEALTH_SNAPSHOT_STATUS_GROUP_CLASS",
      "DATABASE_HEALTH_SNAPSHOT_AUTO_CHECK_CLASS",
      "DATABASE_HEALTH_SECTION_HEADER_CLASS",
      "DATABASE_HEALTH_SNAPSHOT_GRID_CLASS",
      "DATABASE_HEALTH_CONTEXT_STAT_CLASS",
      "DATABASE_HEALTH_CORE_METRIC_GRID_CLASS",
      "DATABASE_HEALTH_CORE_METRIC_CLASS",
      "DATABASE_HEALTH_CORE_METRIC_DETAIL_ROW_CLASS",
      "DATABASE_HEALTH_FINDING_ROW_CLASS",
      "DATABASE_HEALTH_FINDING_HEADER_CLASS",
      "DATABASE_HEALTH_FINDING_TITLE_GROUP_CLASS",
      "DATABASE_HEALTH_FINDING_DETAIL_GRID_CLASS",
      "DATABASE_HEALTH_SECONDARY_GRID_CLASS",
      "DATABASE_HEALTH_SECTION_BODY_FLUSH_CLASS",
      "DATABASE_HEALTH_TABLE_CONTENT_CLASS",
      "DATABASE_HEALTH_OPTIONAL_SIGNAL_TABLE_COLUMN_COUNT",
      "DATABASE_HEALTH_ALERT_TABLE_COLUMN_COUNT",
    ]

    expect(viewSource).toContain('from "@/components/settings/database-health/database-health-layout"')
    expect(loadingStateSource).toContain('from "./database-health-layout"')
    for (const constant of sharedLayoutConstants) {
      expect(layoutSource).toContain(`export const ${constant}`)
      expect(loadingStateSource).toContain(constant)
      if (constant !== "DATABASE_HEALTH_SECTION_PANEL_CLASS") {
        expect(`${viewSource}\n${loadingStateSource}`).toContain(constant)
      }
    }
    expect(viewSource).toContain("DatabaseHealthSectionPanel as SectionPanel")
    expect(layoutSource).toContain("grid-cols-2")
    expect(layoutSource).toContain("md:grid-cols-3")
    expect(layoutSource).toContain("xl:grid-cols-6")
    expect(layoutSource).toContain("gap-px bg-border/60")
    expect(loadingStateSource).toContain("const CONTEXT_STAT_LOADING_COUNT = 6")
    expect(loadingStateSource).toContain('DatabaseHealthTextPlaceholder roleClassName={textRole.sectionTitle} className="w-20"')
    expect(loadingStateSource).toContain("roleClassName={textRole.metricValueDisplay}")
    expect(loadingStateSource).toContain('DatabaseHealthTextPlaceholder roleClassName={textRole.caption} className="w-20"')
    expect(loadingStateSource).toContain('DatabaseHealthTextPlaceholder roleClassName={textRole.caption} className="w-24"')
    expect(loadingStateSource).toContain("lines = 1")
    expect(loadingStateSource).toContain("lines={2}")
    expect(loadingStateSource).toContain('collapseExtraLinesAt="lg"')
    expect(loadingStateSource).toContain('DatabaseHealthBadgePlaceholder className="w-40"')
    expect(loadingStateSource).toContain('DatabaseHealthBadgePlaceholder className="w-36"')

    expect(viewSource).not.toContain('className={cn("flex flex-col gap-4 py-4 md:gap-6 md:py-6", className)}')
    expect(viewSource).not.toContain('className="flex flex-col gap-4 py-4 md:gap-6 md:py-6"')
    expect(viewSource).not.toContain('className="space-y-4 px-4 lg:px-6"')
    expect(viewSource).not.toContain('className="p-0"')
    expect(layoutSource).toContain("export const DATABASE_HEALTH_SECTION_HEADER_WITH_DESCRIPTION_CLASS")
    expect(layoutSource).toContain('DATABASE_HEALTH_CONTENT_SHELL_CLASS =\n  "min-h-0 flex-1 space-y-4 overflow-y-auto px-4 [scrollbar-gutter:stable] lg:px-6"')
    expect(viewSource).toContain("DATABASE_HEALTH_SECTION_HEADER_WITH_DESCRIPTION_CLASS")
    expect(viewSource).not.toContain('className="border-b px-6 py-4"')
    expect(viewSource).not.toContain("<LoadingTable rows={OPTIONAL_SIGNAL_COUNT} columns={4}")
    expect(viewSource).not.toContain("<LoadingTable rows={ALERT_ROW_COUNT} columns={4}")
  })

  it("pairs first-screen database health regions across loading and content", () => {
    for (const slot of [
      "database-health-header",
      "database-health-snapshot",
      "database-health-metrics",
      "database-health-findings",
    ]) {
      expect(loadingStateSource).toContain(`getLoadingStructureSlotAttributes(\"${slot}\")`)
      expect(viewSource).toContain(`getLoadingStructureSlotAttributes(\"${slot}\")`)
    }
  })

  it("mirrors auto-check cadence without a redundant overall-status badge", () => {
    const snapshotHeaderStart = loadingStateSource.indexOf("<CardHeader className={DATABASE_HEALTH_SNAPSHOT_HEADER_CLASS}>")
    const snapshotHeaderSource = loadingStateSource.slice(
      snapshotHeaderStart,
      loadingStateSource.indexOf("DATABASE_HEALTH_SNAPSHOT_GRID_CLASS", snapshotHeaderStart)
    )

    expect(snapshotHeaderSource).toContain("DATABASE_HEALTH_SNAPSHOT_AUTO_CHECK_CLASS")
    expect(snapshotHeaderSource).toContain('DatabaseHealthTextPlaceholder roleClassName={textRole.sectionTitle} className="w-24"')
    expect(snapshotHeaderSource).toContain('DatabaseHealthTextPlaceholder roleClassName={textRole.metadataLabel} className="w-24"')
    expect(snapshotHeaderSource).not.toContain('Skeleton className="h-5 w-24"')
    expect(snapshotHeaderSource).not.toContain('Skeleton className="h-4 w-24"')
    expect(snapshotHeaderSource).not.toContain('Skeleton className="h-6 w-16 rounded-full"')
    expect(snapshotHeaderSource).not.toContain("<Badge")
    expect(snapshotHeaderSource).not.toContain('Skeleton className="size-9"')
    expect(snapshotHeaderSource).not.toContain("ActionSkeleton")
  })
})
