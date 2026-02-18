"use client"

import React from "react"
import { ColumnDef } from "@tanstack/react-table"
import Link from "next/link"
import { Checkbox } from "@/components/ui/checkbox"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import type { ScanRecord, ScanStatus } from "@/types/scan.types"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { Eye, Trash2, MoreHorizontal } from "@/components/icons"
import { DataTableColumnHeader } from "@/components/ui/data-table/column-header"
import { IconCircleX } from "@/components/icons"
import { ScanStatusBadge } from "@/components/scan/scan-status-badge"
import { cn } from "@/lib/utils"

// Translation type definitions
export interface ScanHistoryTranslations {
  columns: {
    target: string
    summary: string
    engineName: string
    workerName: string
    createdAt: string
    status: string
    progress: string
  }
  actions: {
    snapshot: string
    stop: string
    stopScanPending: string
    delete: string
    selectAll: string
    selectRow: string
  }
  tooltips: {
    targetDetails: string
    viewProgress: string
  }
  status: {
    cancelled: string
    completed: string
    failed: string
    pending: string
    running: string
  }
  summary: {
    subdomains: string
    websites: string
    ipAddresses: string
    endpoints: string
    vulnerabilities: string
  }
}

// StatusBadge component removed in favor of ScanStatusBadge in separate file
// function StatusBadge({ ... }) { ... }

// Column creation function parameter types
interface CreateColumnsProps {
  formatDate: (dateString: string) => string
  handleDelete: (scan: ScanRecord) => void
  handleStop: (scan: ScanRecord) => void
  handleViewProgress?: (scan: ScanRecord) => void
  statusClickable?: boolean
  t: ScanHistoryTranslations
  hideTargetColumn?: boolean
}

/**
 * Create scan history table column definitions
 */
export const createScanHistoryColumns = ({
  formatDate,
  handleDelete,
  handleStop,
  handleViewProgress,
  statusClickable = true,
  t,
  hideTargetColumn = false,
}: CreateColumnsProps): ColumnDef<ScanRecord>[] => {
  const columns: ColumnDef<ScanRecord>[] = [
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
    accessorKey: "target",
    accessorFn: (row) => row.target?.name,
    size: 350,
    minSize: 100,
    meta: { title: t.columns.target },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.target} />
    ),
    cell: ({ row }) => {
      const targetName = row.original.target?.name
      const targetId = row.original.targetId
      
      return (
        <div className="flex-1 min-w-0">
          {targetId ? (
            <Tooltip>
              <TooltipTrigger asChild>
                <Link
                  href={`/target/${targetId}/details`}
                  className="text-sm font-medium hover:text-primary hover:underline underline-offset-2 transition-colors text-left break-all leading-relaxed whitespace-normal"
                >
                  {targetName}
                </Link>
              </TooltipTrigger>
              <TooltipContent>{t.tooltips.targetDetails}</TooltipContent>
            </Tooltip>
          ) : (
            <span className="text-sm font-medium break-all leading-relaxed whitespace-normal">
              {targetName}
            </span>
          )}
        </div>
      )
    },
  },
  {
    accessorKey: "cachedStats",
    accessorFn: (row) => row.cachedStats,
    meta: { title: t.columns.summary },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.summary} />
    ),
    size: 290,
    minSize: 150,
    cell: ({ row }) => {
      const subdomains = row.original.cachedStats?.subdomainsCount ?? 0
      const websites = row.original.cachedStats?.websitesCount ?? 0
      const endpoints = row.original.cachedStats?.endpointsCount ?? 0
      const ips = row.original.cachedStats?.ipsCount ?? 0
      const vulns = row.original.cachedStats?.vulnsTotal ?? 0

      const badges: React.ReactNode[] = []

      if (subdomains > 0) {
        badges.push(
          <Badge 
            key="subdomains"
            variant="outline"
            data-badge-type="subdomain"
          >
            {subdomains} SUB
          </Badge>
        )
      }

      if (websites > 0) {
        badges.push(
          <Badge 
            key="websites"
            variant="outline"
            data-badge-type="website"
          >
            {websites} WEB
          </Badge>
        )
      }

      if (ips > 0) {
        badges.push(
          <Badge 
            key="ips"
            variant="outline"
            data-badge-type="ip"
          >
            {ips} IP
          </Badge>
        )
      }

      if (endpoints > 0) {
        badges.push(
          <Badge 
            key="endpoints"
            variant="outline"
            data-badge-type="endpoint"
          >
            {endpoints} URL
          </Badge>
        )
      }

      if (vulns > 0) {
        badges.push(
          <Badge 
            key="vulnerabilities"
            variant="outline"
            data-badge-type="vulnerability"
          >
            {vulns} VULN
          </Badge>
        )
      }

      return (
        <div className="flex flex-wrap items-center gap-1.5">
          {badges.length > 0 ? (
            badges
          ) : (
            <Badge
              variant="outline"
              className="gap-0 bg-muted/70 text-muted-foreground/80 border-border/40 px-1.5 py-0.5 rounded-full justify-center"
            >
              <span className="text-[11px] font-medium leading-none">-</span>
              <span className="sr-only">No summary</span>
            </Badge>
          )}
        </div>
      )
    },
    enableSorting: false,
  },
  {
    accessorKey: "engineNames",
    size: 150,
    minSize: 100,
    maxSize: 200,
    enableResizing: false,
    meta: { title: t.columns.engineName },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.engineName} />
    ),
    cell: ({ row }) => {
      const engineNames = row.getValue("engineNames") as string[] | undefined
      if (!engineNames || engineNames.length === 0) {
        return <span className="text-muted-foreground text-sm">-</span>
      }
      return (
        <div className="flex flex-wrap gap-1">
          {engineNames.map((name, index) => (
            <Badge key={index} variant="secondary" data-badge-type="engine">
              {name}
            </Badge>
          ))}
        </div>
      )
    },
  },
  {
    accessorKey: "workerName",
    size: 120,
    minSize: 80,
    maxSize: 180,
    enableResizing: false,
    meta: { title: t.columns.workerName },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.workerName} />
    ),
    cell: ({ row }) => {
      const workerName = row.getValue("workerName") as string | null | undefined
      return (
        <Badge variant="outline" data-badge-type="worker">
          {workerName || "-"}
        </Badge>
      )
    },
  },
  {
    accessorKey: "createdAt",
    size: 150,
    minSize: 120,
    maxSize: 200,
    enableResizing: false,
    meta: { title: t.columns.createdAt },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.createdAt} />
    ),
    cell: ({ row }) => {
      const createdAt = row.getValue("createdAt") as string
      return (
        <div className="text-sm text-muted-foreground">
          {formatDate(createdAt)}
        </div>
      )
    },
  },
  {
    accessorKey: "status",
    size: 140, // Increased size for the dual-line badge
    minSize: 130,
    maxSize: 160,
    enableResizing: false,
    meta: { title: t.columns.status },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.status} />
    ),
    cell: ({ row }) => {
      const status = row.getValue("status") as ScanStatus
      const progress = row.original.progress
      const isClickable = Boolean(handleViewProgress) && statusClickable

      if (isClickable) {
        return (
          <button
            type="button"
            onClick={() => handleViewProgress?.(row.original)}
            className={cn("w-full cursor-pointer text-left")}
            aria-label={t.tooltips.viewProgress}
          >
            <ScanStatusBadge
              status={status}
              progress={progress}
              labels={t.status}
              variant="inline" // Using F2 variant (Inline Block)
            />
          </button>
        )
      }

      return (
        <div className={cn("cursor-default")}>
          <ScanStatusBadge
            status={status}
            progress={progress}
            labels={t.status}
            variant="inline" // Using F2 variant (Inline Block)
          />
        </div>
      )
    },
  },
  // Progress column removed as it's integrated into status
  // {
  //   accessorKey: "progress", 
  //   ... 
  // },
  {
    id: "actions",
    size: 120,
    minSize: 100,
    maxSize: 150,
    enableResizing: false,
    cell: ({ row }) => {
      const scan = row.original
      const canStop = scan.status === 'running' || scan.status === 'pending'
      
      return (
        <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity duration-200">
          <TooltipProvider delayDuration={300}>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  asChild
                  variant="ghost"
                  size="icon"
                  className="h-8 w-8 text-primary hover:bg-primary/10"
                  aria-label={t.actions.snapshot}
                >
                  <Link href={`/scan/history/${scan.id}/`}>
                    <Eye className="h-4 w-4" />
                  </Link>
                </Button>
              </TooltipTrigger>
              <TooltipContent>{t.actions.snapshot}</TooltipContent>
            </Tooltip>
          </TooltipProvider>

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" className="h-8 w-8 p-0">
                <span className="sr-only">Open menu</span>
                <MoreHorizontal className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              {canStop && (
                <>
                  <DropdownMenuItem onClick={() => handleStop(scan)}>
                    <IconCircleX className="mr-2 h-4 w-4 text-orange-500" />
                    {t.actions.stop}
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                </>
              )}
              
              <DropdownMenuItem 
                onClick={() => handleDelete(scan)}
                className="text-destructive focus:text-destructive"
              >
                <Trash2 className="mr-2 h-4 w-4" />
                {t.actions.delete}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      )
    },
    enableSorting: false,
    enableHiding: false,
  },
]

  // Filter out target column if hideTargetColumn is true
  if (hideTargetColumn) {
    return columns.filter((col) => (col as { accessorKey?: string }).accessorKey !== 'target')
  }

  return columns
}
