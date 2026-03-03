"use client"

import { ColumnDef } from "@tanstack/react-table"
import { Eye, Circle, CheckCircle2 } from "@/components/icons"
import { Checkbox } from "@/components/ui/checkbox"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { ExpandableUrlCell } from "@/components/ui/data-table/expandable-cell"
import { DataTableColumnHeader } from "@/components/ui/data-table/column-header"
import { SEVERITY_STYLES } from "@/lib/severity-config"

import type { Vulnerability, VulnerabilitySeverity } from "@/types/vulnerability.types"

// Translation type definitions
export interface VulnerabilityTranslations {
  columns: {
    status?: string
    severity: string
    source: string
    vulnType: string
    url: string
    createdAt: string
  }
  actions: {
    details: string
    selectAll: string
    selectRow: string
  }
  tooltips: {
    vulnDetails: string
    reviewed: string
    pending: string
  }
  severity: {
    critical: string
    high: string
    medium: string
    low: string
    info: string
  }
}

interface ColumnActions {
  formatDate: (date: string) => string
  handleViewDetail: (vulnerability: Vulnerability) => void
  onToggleReview?: (vulnerability: Vulnerability) => void
  t: VulnerabilityTranslations
}

export function createVulnerabilityColumns({
  formatDate,
  handleViewDetail,
  onToggleReview,
  t,
}: ColumnActions): ColumnDef<Vulnerability>[] {
  // Unified vulnerability severity color configuration
  // Color progression: cool (info) → warm (low/medium) → hot (high/critical)
  const severityConfig: Record<VulnerabilitySeverity, { className: string }> = {
    critical: { className: SEVERITY_STYLES.critical.className },
    high: { className: SEVERITY_STYLES.high.className },
    medium: { className: SEVERITY_STYLES.medium.className },
    low: { className: SEVERITY_STYLES.low.className },
    info: { className: SEVERITY_STYLES.info.className },
  }

  return [
    {
      id: "select",
      size: 40,
      minSize: 40,
      maxSize: 40,
      enableResizing: false,
      header: ({ table }) => (
        <Checkbox
          checked={
            table.getIsAllPageRowsSelected() ||
            (table.getIsSomePageRowsSelected() && "indeterminate")
          }
          onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
          aria-label={t.actions.selectAll}
        />
      ),
      cell: ({ row }) => (
        <Checkbox
          checked={row.getIsSelected()}
          onCheckedChange={(value) => row.toggleSelected(!!value)}
          aria-label={t.actions.selectRow}
        />
      ),
      enableSorting: false,
      enableHiding: false,
    },
    {
      id: "reviewStatus",
      meta: { title: t.columns.status || "状态" },
      size: 100,
      minSize: 90,
      maxSize: 110,
      enableResizing: false,
      header: t.columns.status || "状态",
      cell: ({ row }) => {
        const isReviewed = row.original.isReviewed
        const isPending = !isReviewed

        return (
          onToggleReview ? (
            <Badge
              asChild
              variant="outline"
              className={`transition-[background-color,border-color,color,box-shadow] gap-1.5 cursor-pointer hover:ring-2 hover:ring-offset-1 ${isPending
                ? "bg-blue-500/10 text-blue-600 border-blue-500/30 hover:ring-blue-500/30 dark:text-blue-400 dark:border-blue-400/30"
                : "bg-muted/50 text-muted-foreground border-muted-foreground/20 hover:ring-muted-foreground/30"
                }`}
            >
              <button
                type="button"
                onClick={() => onToggleReview(row.original)}
                aria-label={isPending ? t.tooltips.pending : t.tooltips.reviewed}
              >
                {isPending ? (
                  <Circle className="h-3 w-3" />
                ) : (
                  <CheckCircle2 className="h-3 w-3" />
                )}
                {isPending ? t.tooltips.pending : t.tooltips.reviewed}
              </button>
            </Badge>
          ) : (
            <Badge
              variant="outline"
              className={`transition-[background-color,border-color,color,box-shadow] gap-1.5 cursor-default ${isPending
                ? "bg-blue-500/10 text-blue-600 border-blue-500/30 dark:text-blue-400 dark:border-blue-400/30"
                : "bg-muted/50 text-muted-foreground border-muted-foreground/20"
                }`}
            >
              {isPending ? (
                <Circle className="h-3 w-3" />
              ) : (
                <CheckCircle2 className="h-3 w-3" />
              )}
              {isPending ? t.tooltips.pending : t.tooltips.reviewed}
            </Badge>
          )
        )
      },
      enableSorting: false,
      enableHiding: false,
    },
    {
      accessorKey: "severity",
      meta: { title: t.columns.severity },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.severity} />
      ),
      size: 100,
      minSize: 80,
      maxSize: 120,
      enableResizing: false,
      cell: ({ row }) => {
        const severity = row.getValue("severity") as VulnerabilitySeverity
        const config = severityConfig[severity]
        return (
          <Badge className={config.className}>
            {severity}
          </Badge>
        )
      },
    },
    {
      accessorKey: "source",
      meta: { title: t.columns.source },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.source} />
      ),
      size: 100,
      minSize: 80,
      maxSize: 150,
      enableResizing: false,
      cell: ({ row }) => {
        const source = row.getValue("source") as string
        return (
          <Badge variant="outline">
            {source}
          </Badge>
        )
      },
    },
    {
      accessorKey: "vulnType",
      meta: { title: t.columns.vulnType },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.vulnType} />
      ),
      size: 150,
      minSize: 100,
      maxSize: 250,
      cell: ({ row }) => {
        const vulnType = row.getValue("vulnType") as string
        const vulnerability = row.original
        return (
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                type="button"
                className="font-medium text-left hover:text-primary hover:underline underline-offset-2 transition-colors"
                onClick={() => handleViewDetail(vulnerability)}
                aria-label={t.tooltips.vulnDetails}
              >
                {vulnType}
              </button>
            </TooltipTrigger>
            <TooltipContent>{t.tooltips.vulnDetails}</TooltipContent>
          </Tooltip>
        )
      },
    },
    {
      accessorKey: "url",
      meta: { title: t.columns.url },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.url} />
      ),
      size: 500,
      minSize: 300,
      maxSize: 700,
      cell: ({ row }) => (
        <ExpandableUrlCell value={row.original.url} />
      ),
    },
    {
      accessorKey: "createdAt",
      meta: { title: t.columns.createdAt },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.createdAt} />
      ),
      size: 150,
      minSize: 120,
      maxSize: 200,
      enableResizing: false,
      cell: ({ row }) => {
        const createdAt = row.getValue("createdAt") as string
        return (
          <span className="text-sm text-muted-foreground">
            {formatDate(createdAt)}
          </span>
        )
      },
    },
    {
      id: "actions",
      header: "",
      size: 80,
      minSize: 80,
      maxSize: 80,
      enableResizing: false,
      cell: ({ row }) => {
        const vulnerability = row.original

        return (
          <div className="text-right">
            <Button
              variant="ghost"
              size="sm"
              className="h-8 px-2"
              onClick={() => handleViewDetail(vulnerability)}
            >
              <Eye className="h-4 w-4 mr-1" />
              {t.actions.details}
            </Button>
          </div>
        )
      },
    },
  ]
}
