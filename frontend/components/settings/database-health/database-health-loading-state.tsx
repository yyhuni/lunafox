import type { ComponentProps, ReactNode } from "react"

import { PageHeader } from "@/components/common/page-header"
import {
  getLoadingOwnerAttributes,
  getLoadingStructureSlotAttributes,
} from "@/components/shared/loading/loading-owner"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import {
  TABLE_DENSE_CELL_RHYTHM_CLASS,
  TABLE_DENSE_ROW_CLASS,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

import {
  DATABASE_HEALTH_ALERT_TABLE_COLUMN_COUNT,
  DATABASE_HEALTH_CONTENT_SHELL_CLASS,
  DATABASE_HEALTH_CONTEXT_STAT_CLASS,
  DATABASE_HEALTH_CORE_METRIC_CLASS,
  DATABASE_HEALTH_CORE_METRIC_DETAIL_ROW_CLASS,
  DATABASE_HEALTH_CORE_METRIC_GRID_CLASS,
  DATABASE_HEALTH_FINDING_DETAIL_GRID_CLASS,
  DATABASE_HEALTH_FINDING_HEADER_CLASS,
  DATABASE_HEALTH_FINDING_ROW_CLASS,
  DATABASE_HEALTH_FINDING_TITLE_GROUP_CLASS,
  DATABASE_HEALTH_OPTIONAL_SIGNAL_TABLE_COLUMN_COUNT,
  DATABASE_HEALTH_PAGE_SHELL_CLASS,
  DATABASE_HEALTH_SECONDARY_GRID_CLASS,
  DATABASE_HEALTH_SECTION_BODY_FLUSH_CLASS,
  DATABASE_HEALTH_SECTION_HEADER_CLASS,
  DATABASE_HEALTH_SECTION_PANEL_CLASS,
  DATABASE_HEALTH_SNAPSHOT_AUTO_CHECK_CLASS,
  DATABASE_HEALTH_SNAPSHOT_GRID_CLASS,
  DATABASE_HEALTH_SNAPSHOT_HEADER_CLASS,
  DATABASE_HEALTH_SNAPSHOT_STATUS_GROUP_CLASS,
  DATABASE_HEALTH_TABLE_CONTENT_CLASS,
} from "./database-health-layout"

const CONTEXT_STAT_LOADING_COUNT = 6
const CORE_METRIC_LOADING_COUNT = 6
const FINDING_ROW_LOADING_COUNT = 1
const OPTIONAL_SIGNAL_LOADING_COUNT = 3
const ALERT_ROW_LOADING_COUNT = 1

function DatabaseHealthTextPlaceholder({
  roleClassName,
  className,
  lines = 1,
  collapseExtraLinesAt,
}: {
  roleClassName: string
  className?: string
  lines?: number
  collapseExtraLinesAt?: "lg"
}) {
  // Preserve the resolved text role's line box so loading cannot shift panel rhythm.
  return (
    <div aria-hidden="true" className={cn("relative", roleClassName, className)}>
      {Array.from({ length: lines }).map((_, index) => (
        <span key={index} className={cn("block invisible", index > 0 && collapseExtraLinesAt === "lg" && "lg:hidden")}>
          {"\u00a0"}
        </span>
      ))}
      <Skeleton className="absolute inset-0" />
    </div>
  )
}

function DatabaseHealthBadgePlaceholder({ className }: { className?: string }) {
  return (
    <span aria-hidden="true" className={cn("relative inline-flex", className)}>
      <Badge variant="outline" className="invisible">{"\u00a0"}</Badge>
      <Skeleton className="absolute inset-0 rounded-full" />
    </span>
  )
}

export function DatabaseHealthSectionPanel({
  children,
  className,
  ...props
}: { children: ReactNode; className?: string } & ComponentProps<typeof Card>) {
  return (
    <Card className={cn(DATABASE_HEALTH_SECTION_PANEL_CLASS, className)} {...props}>
      {children}
    </Card>
  )
}

function LoadingTable({ rows, columns }: { rows: number; columns: number }) {
  return (
    <Table>
      <TableHeader>
        <TableRow>
          {Array.from({ length: columns }).map((_, index) => (
            <TableHead key={index}>
              <Skeleton className="h-4 w-16" />
            </TableHead>
          ))}
        </TableRow>
      </TableHeader>
      <TableBody>
        {Array.from({ length: rows }).map((_, rowIndex) => (
          <TableRow key={rowIndex} className={TABLE_DENSE_ROW_CLASS}>
            {Array.from({ length: columns }).map((_, cellIndex) => (
              <TableCell key={cellIndex} className={TABLE_DENSE_CELL_RHYTHM_CLASS}>
                <Skeleton className={cn("h-5", cellIndex === 1 ? "w-32" : "w-16")} />
              </TableCell>
            ))}
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}

export function DatabaseHealthLoadingState({
  owner,
  pageTitle,
  pageDescription,
  className,
}: {
  owner?: string
  pageTitle: string
  pageDescription: string
  className?: string
}) {
  if (owner !== undefined && !owner.trim()) {
    throw new Error("DatabaseHealthLoadingState requires a non-empty owner.")
  }

  return (
    <div
      {...(owner ? getLoadingOwnerAttributes({ owner, layer: "route", intent: "route" }) : {})}
      data-slot="database-health-loading-state"
      className={cn(DATABASE_HEALTH_PAGE_SHELL_CLASS, className)}
    >
      <div {...getLoadingStructureSlotAttributes("database-health-header")}>
        <PageHeader code="DBH-01" title={pageTitle} description={pageDescription} />
      </div>

      <div className={DATABASE_HEALTH_CONTENT_SHELL_CLASS}>
        <DatabaseHealthSectionPanel {...getLoadingStructureSlotAttributes("database-health-snapshot")}>
          <CardHeader className={DATABASE_HEALTH_SNAPSHOT_HEADER_CLASS}>
            <div className="min-w-0">
              <CardTitle>
                <DatabaseHealthTextPlaceholder roleClassName={textRole.sectionTitle} className="w-24" />
              </CardTitle>
              <CardDescription>
                <DatabaseHealthTextPlaceholder
                  roleClassName={textRole.bodySubtle}
                  className="mt-1 w-48 max-w-full"
                />
              </CardDescription>
            </div>
            <div className={DATABASE_HEALTH_SNAPSHOT_STATUS_GROUP_CLASS}>
              <div className={DATABASE_HEALTH_SNAPSHOT_AUTO_CHECK_CLASS}>
                <Skeleton className="size-2 rounded-full" />
                <DatabaseHealthTextPlaceholder roleClassName={textRole.metadataLabel} className="w-24" />
              </div>
            </div>
          </CardHeader>
          <CardContent className={DATABASE_HEALTH_SNAPSHOT_GRID_CLASS}>
            {Array.from({ length: CONTEXT_STAT_LOADING_COUNT }).map((_, index) => (
              <div key={index} className={DATABASE_HEALTH_CONTEXT_STAT_CLASS}>
                <Skeleton className="h-5 w-24 max-w-full" />
                <DatabaseHealthTextPlaceholder
                  roleClassName={textRole.caption}
                  className="mt-1 w-20 max-w-full"
                />
              </div>
            ))}
          </CardContent>
        </DatabaseHealthSectionPanel>

        <DatabaseHealthSectionPanel {...getLoadingStructureSlotAttributes("database-health-metrics")}>
          <CardHeader className={DATABASE_HEALTH_SECTION_HEADER_CLASS}>
            <CardTitle>
              <DatabaseHealthTextPlaceholder roleClassName={textRole.sectionTitle} className="w-20" />
            </CardTitle>
          </CardHeader>
          <CardContent className={DATABASE_HEALTH_SECTION_BODY_FLUSH_CLASS}>
            <div className={DATABASE_HEALTH_CORE_METRIC_GRID_CLASS}>
              {Array.from({ length: CORE_METRIC_LOADING_COUNT }).map((_, index) => (
                <div key={index} className={DATABASE_HEALTH_CORE_METRIC_CLASS}>
                  <Skeleton className="h-5 w-28 max-w-full" />
                  <DatabaseHealthTextPlaceholder
                    roleClassName={textRole.metricValueDisplay}
                    className="mt-2 w-20"
                  />
                  {index === 1 ? (
                    <div className="mt-2 max-w-44">
                      <Skeleton className="h-2 w-full" />
                    </div>
                  ) : null}
                  <div className={DATABASE_HEALTH_CORE_METRIC_DETAIL_ROW_CLASS}>
                    <DatabaseHealthTextPlaceholder roleClassName={textRole.caption} className="w-20" />
                    <DatabaseHealthTextPlaceholder roleClassName={textRole.caption} className="w-24" />
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </DatabaseHealthSectionPanel>

        <DatabaseHealthSectionPanel {...getLoadingStructureSlotAttributes("database-health-findings")}>
          <CardHeader className={DATABASE_HEALTH_SECTION_HEADER_CLASS}>
            <CardTitle>
              <DatabaseHealthTextPlaceholder roleClassName={textRole.sectionTitle} className="w-24" />
            </CardTitle>
          </CardHeader>
          <CardContent className={DATABASE_HEALTH_SECTION_BODY_FLUSH_CLASS}>
            {Array.from({ length: FINDING_ROW_LOADING_COUNT }).map((_, index) => (
              <div key={index} className={DATABASE_HEALTH_FINDING_ROW_CLASS}>
                <div className={DATABASE_HEALTH_FINDING_HEADER_CLASS}>
                  <div className="min-w-0">
                    <div className={DATABASE_HEALTH_FINDING_TITLE_GROUP_CLASS}>
                      <DatabaseHealthBadgePlaceholder className="w-16" />
                      <Skeleton className="h-5 w-48 max-w-full" />
                    </div>
                    <DatabaseHealthTextPlaceholder
                      roleClassName={textRole.bodySubtle}
                      className="mt-2 w-80 max-w-full"
                      lines={2}
                      collapseExtraLinesAt="lg"
                    />
                  </div>
                  <DatabaseHealthBadgePlaceholder className="w-24" />
                </div>
                <div className={DATABASE_HEALTH_FINDING_DETAIL_GRID_CLASS}>
                  <div>
                    <DatabaseHealthTextPlaceholder roleClassName={textRole.caption} className="w-16" />
                    <div className="mt-1 flex flex-wrap gap-2">
                      <DatabaseHealthBadgePlaceholder className="w-40" />
                      <DatabaseHealthBadgePlaceholder className="w-24" />
                      <DatabaseHealthBadgePlaceholder className="w-36" />
                    </div>
                  </div>
                  <div>
                    <DatabaseHealthTextPlaceholder roleClassName={textRole.caption} className="w-24" />
                    <DatabaseHealthTextPlaceholder
                      roleClassName={textRole.body}
                      className="mt-1 w-72 max-w-full"
                      lines={2}
                    />
                  </div>
                </div>
              </div>
            ))}
          </CardContent>
        </DatabaseHealthSectionPanel>

        <div className={DATABASE_HEALTH_SECONDARY_GRID_CLASS}>
          <DatabaseHealthSectionPanel id="database-health-optional-signals-loading">
            <CardHeader className={DATABASE_HEALTH_SECTION_HEADER_CLASS}>
              <CardTitle>
                <Skeleton className="h-5 w-24" />
              </CardTitle>
            </CardHeader>
            <CardContent className={DATABASE_HEALTH_TABLE_CONTENT_CLASS}>
              <LoadingTable
                rows={OPTIONAL_SIGNAL_LOADING_COUNT}
                columns={DATABASE_HEALTH_OPTIONAL_SIGNAL_TABLE_COLUMN_COUNT}
              />
            </CardContent>
          </DatabaseHealthSectionPanel>

          <DatabaseHealthSectionPanel id="database-health-alerts-loading">
            <CardHeader className={DATABASE_HEALTH_SECTION_HEADER_CLASS}>
              <CardTitle>
                <Skeleton className="h-5 w-20" />
              </CardTitle>
            </CardHeader>
            <CardContent className={DATABASE_HEALTH_TABLE_CONTENT_CLASS}>
              <LoadingTable
                rows={ALERT_ROW_LOADING_COUNT}
                columns={DATABASE_HEALTH_ALERT_TABLE_COLUMN_COUNT}
              />
            </CardContent>
          </DatabaseHealthSectionPanel>
        </div>
      </div>
    </div>
  )
}
