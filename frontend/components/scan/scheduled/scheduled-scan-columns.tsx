"use client"

import React from "react"
import { ColumnDef } from "@tanstack/react-table"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Switch } from "@/components/ui/switch"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  MoreHorizontal,
  Trash2,
  Edit,
  Building2,
  Target,
} from "@/components/icons"
import { DataTableColumnHeader } from "@/components/ui/data-table/column-header"
import type { ScheduledScan } from "@/types/scheduled-scan.types"

// Translation type definitions
export interface ScheduledScanTranslations {
  columns: {
    taskName: string
    scanWorkflow: string
    cronExpression: string
    scope: string
    status: string
    nextRun: string
    runCount: string
    lastRun: string
  }
  actions: {
    editTask: string
    delete: string
    openMenu: string
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
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          className="flex h-8 w-8 p-0 data-[state=open]:bg-muted"
        >
          <MoreHorizontal />
          <span className="sr-only">{t.actions.openMenu}</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onClick={onEdit}>
          <Edit />
          {t.actions.editTask}
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem
          onClick={onDelete}
          className="text-destructive focus:text-destructive"
        >
          <Trash2 />
          {t.actions.delete}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
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
    accessorKey: "name",
    size: 350,
    minSize: 250,
    meta: { title: t.columns.taskName },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.taskName} />
    ),
    cell: ({ row }) => {
      const name = row.getValue("name") as string
      if (!name) return <span className="text-muted-foreground text-sm">-</span>

      return (
        <div className="flex-1 min-w-0">
          <span className="text-sm font-medium break-all leading-relaxed whitespace-normal">
            {name}
          </span>
        </div>
      )
    },
  },
  {
    accessorKey: "workflowNames",
    size: 150,
    minSize: 100,
    meta: { title: t.columns.scanWorkflow },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.scanWorkflow} />
    ),
    cell: ({ row }) => {
      const workflowNames = row.original.workflowNames || []
      if (workflowNames.length === 0) {
        return <span className="text-muted-foreground text-sm">-</span>
      }
      return (
        <div className="flex flex-wrap gap-1">
          {workflowNames.map((name, index) => (
            <Badge key={index} variant="secondary">
              {name}
            </Badge>
          ))}
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
    size: 150,
    minSize: 100,
    cell: ({ row }) => {
      const cron = row.original.cronExpression
      return (
        <div className="flex flex-col gap-1">
          <span className="text-sm">
            {parseCronExpression(cron, t)}
          </span>
          <code className="text-xs text-muted-foreground font-mono">
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
    size: 200,
    minSize: 150,
    cell: ({ row }) => {
      const scanMode = row.original.scanMode
      const organizationName = row.original.organizationName
      const targetName = row.original.targetName
      
      if (scanMode === 'organization' && organizationName) {
        return (
          <div className="flex items-center gap-2">
            <Building2 className="h-4 w-4 text-muted-foreground shrink-0" />
            <span className="text-sm truncate">{organizationName}</span>
          </div>
        )
      }
      
      if (targetName) {
        return (
          <div className="flex items-center gap-2">
            <Target className="h-4 w-4 text-muted-foreground shrink-0" />
            <span className="text-sm font-mono truncate">{targetName}</span>
          </div>
        )
      }
      
      return <span className="text-sm text-muted-foreground">-</span>
    },
    enableSorting: false,
  },
  {
    accessorKey: "isEnabled",
    size: 120,
    minSize: 100,
    meta: { title: t.columns.status },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.status} />
    ),
    cell: ({ row }) => {
      const isEnabled = row.getValue("isEnabled") as boolean
      const scan = row.original
      return (
        <div className="flex items-center gap-2">
          <Switch
            checked={isEnabled}
            onCheckedChange={(checked: boolean) =>
              handleToggleStatus(scan, checked)
            }
          />
          <span className="text-sm text-muted-foreground">
            {isEnabled ? t.status.enabled : t.status.disabled}
          </span>
        </div>
      )
    },
  },
  {
    accessorKey: "nextRunTime",
    size: 150,
    minSize: 120,
    meta: { title: t.columns.nextRun },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.nextRun} />
    ),
    cell: ({ row }) => {
      const nextRunTime = row.getValue("nextRunTime") as string | undefined
      return (
        <div className="text-sm text-muted-foreground">
          {nextRunTime ? formatDate(nextRunTime) : "-"}
        </div>
      )
    },
  },
  {
    accessorKey: "runCount",
    size: 80,
    minSize: 60,
    meta: { title: t.columns.runCount },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.runCount} />
    ),
    cell: ({ row }) => {
      const count = row.getValue("runCount") as number
      return (
        <div className="text-sm text-muted-foreground font-mono">{count}</div>
      )
    },
  },
  {
    accessorKey: "lastRunTime",
    size: 150,
    minSize: 120,
    meta: { title: t.columns.lastRun },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.lastRun} />
    ),
    cell: ({ row }) => {
      const lastRunTime = row.getValue("lastRunTime") as string | undefined
      return (
        <div className="text-sm text-muted-foreground">
          {lastRunTime ? formatDate(lastRunTime) : "-"}
        </div>
      )
    },
  },
  {
    id: "actions",
    size: 60,
    minSize: 60,
    maxSize: 60,
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
