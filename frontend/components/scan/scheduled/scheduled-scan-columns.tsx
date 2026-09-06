"use client"

import React from "react"
import { ColumnDef } from "@tanstack/react-table"
import { Checkbox } from "@/components/ui/checkbox"
import { Switch } from "@/components/ui/switch"
import {
  DropdownMenuItem,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu"
import {
  semanticIcons,
} from "@/components/icons"
import { DataTableColumnHeader } from "@/components/shared/data-table/column-header"
import { DenseRowActionMenu } from "@/components/shared/data-table/menu-owners"
import { TimestampCell } from "@/components/shared/data-table/timestamp-cell"
import { getStatusToneTextClass } from "@/lib/status-config"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import type { ScheduledScan } from "@/types/scheduled-scan.types"
import { scheduledScanTableColumnLayout } from "./scheduled-scan-table-layout"

const EditIcon = semanticIcons.action.edit
const DeleteIcon = semanticIcons.action.delete
const OrganizationIcon = semanticIcons.concept.organization
const TargetIcon = semanticIcons.concept.target

// Translation type definitions
export interface ScheduledScanTranslations {
  columns: {
    taskName: string
    cronExpression: string
    scope: string
    status: string
    nextRun: string
    handoffResults: string
    trigger: string
    success: string
    failure: string
    lastRun: string
  }
  actions: {
    editTask: string
    delete: string
    openMenu: string
    selectAll: string
    selectRow: string
  }
  status: {
    enabled: string
    disabled: string
  }
  cron: {
    everyMinute: string
    everyNMinutes: string
    everyHour: string
    everyNHours: string
    everyDay: string
    everyWeek: string
    everyMonth: string
    weekdays: string[]
  }
}

interface CreateColumnsProps {
  formatDate: (dateString: string) => string
  handleEdit: (scan: ScheduledScan) => void
  handleDelete: (scan: ScheduledScan) => void
  handleToggleStatus: (scan: ScheduledScan, enabled: boolean) => void
  t: ScheduledScanTranslations
}

/**
 * Parse Cron expression to human-readable format
 */
function parseCronExpression(cron: string, t: ScheduledScanTranslations): string {
  if (!cron) return '-'
  
  const parts = cron.split(' ')
  if (parts.length !== 5) return cron
  
  const [minute, hour, day, month, weekday] = parts
  
  if (minute === '*' && hour === '*' && day === '*' && month === '*' && weekday === '*') {
    return t.cron.everyMinute
  }
  
  if (minute.startsWith('*/') && hour === '*') {
    return t.cron.everyNMinutes.replace('{n}', minute.slice(2))
  }
  
  if (minute !== '*' && hour === '*' && day === '*') {
    return t.cron.everyHour.replace('{minute}', minute)
  }
  
  if (hour.startsWith('*/')) {
    return t.cron.everyNHours.replace('{n}', hour.slice(2)).replace('{minute}', minute)
  }
  
  if (day === '*' && month === '*' && weekday === '*') {
    return t.cron.everyDay.replace('{time}', `${hour.padStart(2, '0')}:${minute.padStart(2, '0')}`)
  }
  
  if (day === '*' && month === '*' && weekday !== '*') {
    const dayName = t.cron.weekdays[parseInt(weekday)] || weekday
    return t.cron.everyWeek.replace('{day}', dayName).replace('{time}', `${hour.padStart(2, '0')}:${minute.padStart(2, '0')}`)
  }
  
  if (day !== '*' && month === '*' && weekday === '*') {
    return t.cron.everyMonth.replace('{day}', day).replace('{time}', `${hour.padStart(2, '0')}:${minute.padStart(2, '0')}`)
  }
  
  return cron
}

/**
 * Scheduled scan row actions component
 */
function ScheduledScanRowActions({
  onEdit,
  onDelete,
  t,
}: {
  onEdit: () => void
  onDelete: () => void
  t: ScheduledScanTranslations
}) {
  return (
    <DenseRowActionMenu ariaLabel={t.actions.openMenu}>
      <DropdownMenuItem onClick={onEdit}>
        <EditIcon />
        {t.actions.editTask}
      </DropdownMenuItem>
      <DropdownMenuSeparator />
      <DropdownMenuItem
        onClick={onDelete}
        className="focus:text-destructive text-destructive"
      >
        <DeleteIcon />
        {t.actions.delete}
      </DropdownMenuItem>
    </DenseRowActionMenu>
  )
}

/**
 * Create scheduled scan table column definitions
 */
export const createScheduledScanColumns = ({
  formatDate,
  handleEdit,
  handleDelete,
  handleToggleStatus,
  t,
}: CreateColumnsProps): ColumnDef<ScheduledScan>[] => [
  {
    id: "select",
    size: scheduledScanTableColumnLayout.select.size,
    minSize: scheduledScanTableColumnLayout.select.minSize,
    maxSize: scheduledScanTableColumnLayout.select.maxSize,
    enableResizing: false,
    header: ({ table }) => (
      <Checkbox
        checked={table.getIsAllPageRowsSelected()}
        indeterminate={
          !table.getIsAllPageRowsSelected() && table.getIsSomePageRowsSelected()
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
    accessorKey: "name",
    size: scheduledScanTableColumnLayout.name.size,
    minSize: scheduledScanTableColumnLayout.name.minSize,
    meta: { title: t.columns.taskName },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.taskName} />
    ),
    cell: ({ row }) => {
      const name = row.getValue("name") as string
      if (!name) return <span className={textRole.tableCellSecondary}>-</span>

      return (
        <div className="min-w-0">
          <span className={cn(textRole.tableCellPrimary, "break-all whitespace-normal")}>
            {name}
          </span>
        </div>
      )
    },
  },
  {
    accessorKey: "cronExpression",
    meta: { title: t.columns.cronExpression },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.cronExpression} />
    ),
    size: scheduledScanTableColumnLayout.cronExpression.size,
    minSize: scheduledScanTableColumnLayout.cronExpression.minSize,
    cell: ({ row }) => {
      const cron = row.original.cronExpression
      return (
        <div className="flex flex-col gap-1">
          <span className={textRole.tableCellPrimary}>
            {parseCronExpression(cron, t)}
          </span>
          <code className={cn(textRole.tableCellSecondary, "font-mono tabular-nums")}>
            {cron}
          </code>
        </div>
      )
    },
    enableSorting: false,
  },
  {
    accessorKey: "scanMode",
    meta: { title: t.columns.scope },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.scope} />
    ),
    size: scheduledScanTableColumnLayout.scanMode.size,
    minSize: scheduledScanTableColumnLayout.scanMode.minSize,
    cell: ({ row }) => {
      const scanMode = row.original.scanMode
      const organizationName = row.original.organizationName
      const targetName = row.original.targetName
      
      if (scanMode === 'organization' && organizationName) {
        return (
          <div className="flex gap-2 items-center">
            <OrganizationIcon className="h-4 shrink-0 text-muted-foreground w-4" />
            <span className={cn(textRole.tableCellPrimary, "truncate")}>{organizationName}</span>
          </div>
        )
      }
      
      if (targetName) {
        return (
          <div className="flex gap-2 items-center">
            <TargetIcon className="h-4 shrink-0 text-muted-foreground w-4" />
            <span className={cn(textRole.tableCellPrimary, "font-mono truncate")}>{targetName}</span>
          </div>
        )
      }
      
      return <span className={textRole.tableCellSecondary}>-</span>
    },
    enableSorting: false,
  },
  {
    accessorKey: "isEnabled",
    size: scheduledScanTableColumnLayout.isEnabled.size,
    minSize: scheduledScanTableColumnLayout.isEnabled.minSize,
    meta: { title: t.columns.status },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.status} />
    ),
    cell: ({ row }) => {
      const isEnabled = row.getValue("isEnabled") as boolean
      const scan = row.original
      return (
        <div className="flex gap-2 items-center">
          <Switch
            checked={isEnabled}
            onCheckedChange={(checked: boolean) =>
              handleToggleStatus(scan, checked)
            }
          />
          <span className={textRole.tableCellSecondary}>
            {isEnabled ? t.status.enabled : t.status.disabled}
          </span>
        </div>
      )
    },
  },
  {
    accessorKey: "nextRunTime",
    size: scheduledScanTableColumnLayout.nextRunTime.size,
    minSize: scheduledScanTableColumnLayout.nextRunTime.minSize,
    maxSize: scheduledScanTableColumnLayout.nextRunTime.maxSize,
    enableResizing: false,
    meta: { title: t.columns.nextRun },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.nextRun} />
    ),
    cell: ({ row }) => {
      const nextRunTime = row.getValue("nextRunTime") as string | undefined
      return <TimestampCell value={nextRunTime ? formatDate(nextRunTime) : "-"} />
    },
  },
  {
    id: "handoffResults",
    size: scheduledScanTableColumnLayout.handoffResults.size,
    minSize: scheduledScanTableColumnLayout.handoffResults.minSize,
    maxSize: scheduledScanTableColumnLayout.handoffResults.maxSize,
    meta: { title: t.columns.handoffResults },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.handoffResults} />
    ),
    cell: ({ row }) => {
      return (
        <span className={cn(textRole.tableCellSecondary, "font-mono tabular-nums whitespace-nowrap")}>
          <span>{t.columns.trigger} {row.original.runCount}</span>
          <span aria-hidden="true"> · </span>
          <span className={getStatusToneTextClass("success")}>
            {t.columns.success} {row.original.successfulHandoffCount}
          </span>
          <span aria-hidden="true"> · </span>
          <span className={getStatusToneTextClass("error")}>
            {t.columns.failure} {row.original.failedHandoffCount}
          </span>
        </span>
      )
    },
    enableSorting: false,
  },
  {
    accessorKey: "lastRunTime",
    size: scheduledScanTableColumnLayout.lastRunTime.size,
    minSize: scheduledScanTableColumnLayout.lastRunTime.minSize,
    maxSize: scheduledScanTableColumnLayout.lastRunTime.maxSize,
    enableResizing: false,
    meta: { title: t.columns.lastRun },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.lastRun} />
    ),
    cell: ({ row }) => {
      const lastRunTime = row.getValue("lastRunTime") as string | undefined
      return <TimestampCell value={lastRunTime ? formatDate(lastRunTime) : "-"} />
    },
  },
  {
    id: "actions",
    size: scheduledScanTableColumnLayout.actions.size,
    minSize: scheduledScanTableColumnLayout.actions.minSize,
    maxSize: scheduledScanTableColumnLayout.actions.maxSize,
    enableResizing: false,
    cell: ({ row }) => (
      <ScheduledScanRowActions
        onEdit={() => handleEdit(row.original)}
        onDelete={() => handleDelete(row.original)}
        t={t}
      />
    ),
    enableSorting: false,
    enableHiding: false,
  },
]
