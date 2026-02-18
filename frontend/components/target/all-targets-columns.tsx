"use client"

import React from "react"
import { ColumnDef } from "@tanstack/react-table"
import Link from "next/link"
import { Checkbox } from "@/components/ui/checkbox"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { MoreHorizontal, Eye, Trash2, Play, Calendar, Copy, Check } from "@/components/icons"
import { toast } from "sonner"
import { DataTableColumnHeader } from "@/components/ui/data-table/column-header"
import { ExpandableBadgeList } from "@/components/ui/data-table/expandable-cell"
import type { Target } from "@/types/target.types"

// Translation type definitions
export interface AllTargetsTranslations {
  columns: {
    target: string
    organization: string
    addedOn: string
    lastScanned: string
  }
  actions: {
    scheduleScan: string
    delete: string
    selectAll: string
    selectRow: string
    openMenu: string
  }
  tooltips: {
    targetDetails: string
    targetSummary: string
    initiateScan: string
    clickToCopy: string
    copied: string
  }
}

/**
 * Copy to clipboard (compatible with HTTP environment)
 */
async function copyToClipboard(text: string): Promise<boolean> {
  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text)
    } else {
      const textArea = document.createElement('textarea')
      textArea.value = text
      textArea.style.position = 'fixed'
      textArea.style.left = '-9999px'
      textArea.style.top = '-9999px'
      document.body.appendChild(textArea)
      textArea.focus()
      textArea.select()
      document.execCommand('copy')
      textArea.remove()
    }
    return true
  } catch {
    return false
  }
}

interface CreateColumnsProps {
  formatDate: (dateString: string) => string
  navigate: (path: string) => void
  handleDelete: (target: Target) => void
  handleInitiateScan: (target: Target) => void
  handleScheduleScan: (target: Target) => void
  t: AllTargetsTranslations
}

/**
 * Target name cell component
 */
const TargetNameCell = React.memo(function TargetNameCell({
  name,
  targetId,
  t,
}: {
  name: string
  targetId: number
  t: AllTargetsTranslations
}) {
  const [copied, setCopied] = React.useState(false)

  const handleCopy = React.useCallback(async (e: React.MouseEvent) => {
    e.stopPropagation()
    const success = await copyToClipboard(name)
    if (success) {
      setCopied(true)
      toast.success(t.tooltips.copied)
      setTimeout(() => setCopied(false), 2000)
    }
  }, [name, t.tooltips.copied])

  return (
    <div className="group flex items-start gap-1 flex-1 min-w-0">
      <Tooltip>
        <TooltipTrigger asChild>
          <Link
            href={`/target/${targetId}/details`}
            className="text-sm font-medium hover:text-primary hover:underline underline-offset-2 transition-colors text-left break-all leading-relaxed whitespace-normal"
          >
            {name}
          </Link>
        </TooltipTrigger>
        <TooltipContent>{t.tooltips.targetDetails}</TooltipContent>
      </Tooltip>
      <TooltipProvider delayDuration={300}>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className={`h-6 w-6 flex-shrink-0 hover:bg-accent transition-opacity ${
                copied ? 'opacity-100' : 'opacity-0 group-hover:opacity-100'
              }`}
              onClick={handleCopy}
              aria-label={copied ? t.tooltips.copied : t.tooltips.clickToCopy}
            >
              {copied ? (
                <Check className="h-3.5 w-3.5 text-green-600 dark:text-green-400" />
              ) : (
                <Copy className="h-3.5 w-3.5 text-muted-foreground" />
              )}
            </Button>
          </TooltipTrigger>
          <TooltipContent side="top">
            <p className="text-xs">{copied ? t.tooltips.copied : t.tooltips.clickToCopy}</p>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    </div>
  )
})

/**
 * Target row actions component
 */
const TargetRowActions = React.memo(function TargetRowActions({
  onView,
  onInitiateScan,
  onScheduleScan,
  onDelete,
  t,
}: {
  onView: () => void
  onInitiateScan: () => void
  onScheduleScan: () => void
  onDelete: () => void
  t: AllTargetsTranslations
}) {
  return (
    <div className="flex items-center gap-1">
      <TooltipProvider delayDuration={300}>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8"
              onClick={onView}
              aria-label={t.tooltips.targetSummary}
            >
              <Eye className="h-4 w-4" />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="top">
            <p className="text-xs">{t.tooltips.targetSummary}</p>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>

      <TooltipProvider delayDuration={300}>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8"
              onClick={onInitiateScan}
              aria-label={t.tooltips.initiateScan}
            >
              <Play className="h-4 w-4" />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="top">
            <p className="text-xs">{t.tooltips.initiateScan}</p>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>

      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            className="h-8 w-8 p-0 data-[state=open]:bg-muted"
            aria-label={t.actions.openMenu}
          >
            <MoreHorizontal className="h-4 w-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-48">
          <DropdownMenuItem onClick={onScheduleScan}>
            <Calendar />
            {t.actions.scheduleScan}
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
    </div>
  )
})

/**
 * Create all targets table column definitions
 */
export const createAllTargetsColumns = ({
  formatDate,
  navigate,
  handleDelete,
  handleInitiateScan,
  handleScheduleScan,
  t,
}: CreateColumnsProps): ColumnDef<Target>[] => [
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
    accessorKey: "name",
    size: 350,
    minSize: 250,
    meta: { title: t.columns.target },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.target} />
    ),
    cell: ({ row }) => (
      <TargetNameCell
        name={row.getValue("name") as string}
        targetId={row.original.id}
        t={t}
      />
    ),
  },
  {
    accessorKey: "organizations",
    size: 200,
    minSize: 150,
    maxSize: 350,
    meta: { title: t.columns.organization },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.organization} />
    ),
    cell: ({ row }) => {
      const organizations = row.getValue("organizations") as Array<{ id: number; name: string }> | undefined
      return (
        <ExpandableBadgeList
          items={organizations}
          maxVisible={2}
          variant="secondary"
        />
      )
    },
    enableSorting: false,
  },
  {
    accessorKey: "createdAt",
    size: 150,
    minSize: 120,
    maxSize: 200,
    meta: { title: t.columns.addedOn },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.addedOn} />
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
    accessorKey: "lastScannedAt",
    size: 150,
    minSize: 120,
    maxSize: 200,
    meta: { title: t.columns.lastScanned },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.lastScanned} />
    ),
    cell: ({ row }) => {
      const lastScannedAt = row.original.lastScannedAt
      if (!lastScannedAt) {
        return <span className="text-sm text-muted-foreground">-</span>
      }
      return (
        <div className="text-sm text-muted-foreground">
          {formatDate(lastScannedAt)}
        </div>
      )
    },
  },
  {
    id: "actions",
    size: 120,
    minSize: 100,
    maxSize: 150,
    enableResizing: false,
    cell: ({ row }) => (
      <TargetRowActions
        onView={() => navigate(`/target/${row.original.id}/details`)}
        onInitiateScan={() => handleInitiateScan(row.original)}
        onScheduleScan={() => handleScheduleScan(row.original)}
        onDelete={() => handleDelete(row.original)}
        t={t}
      />
    ),
    enableSorting: false,
    enableHiding: false,
  },
]
