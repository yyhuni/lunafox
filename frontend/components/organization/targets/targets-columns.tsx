"use client"

import React from "react"
import { ColumnDef } from "@tanstack/react-table"
import Link from "next/link"
import { Checkbox } from "@/components/ui/checkbox"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { Eye, Trash2, Copy, Check } from "@/components/icons"
import { DataTableColumnHeader } from "@/components/ui/data-table/column-header"
import type { Target } from "@/types/target.types"
import { toast } from "sonner"

// Translation type definition
export interface OrgTargetsTranslations {
  columns: {
    targetName: string
    type: string
  }
  actions: {
    selectAll: string
    selectRow: string
  }
  tooltips: {
    viewDetails: string
    unlinkTarget: string
    clickToCopy: string
    copied: string
  }
  types: {
    domain: string
    ip: string
    cidr: string
  }
}

interface CreateColumnsProps {
  formatDate: (dateString: string) => string
  navigate: (path: string) => void
  handleDelete: (target: Target) => void
  t: OrgTargetsTranslations
}

/**
 * target row operation component
 */
function TargetRowActions({
  onView,
  onDelete,
  t,
}: {
  onView: () => void
  onDelete: () => void
  t: OrgTargetsTranslations
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
              aria-label={t.tooltips.viewDetails}
            >
              <Eye className="h-4 w-4" />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="top">
            <p className="text-xs">{t.tooltips.viewDetails}</p>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>

      <TooltipProvider delayDuration={300}>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8 text-destructive hover:text-destructive"
              onClick={onDelete}
              aria-label={t.tooltips.unlinkTarget}
            >
              <Trash2 className="h-4 w-4" />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="top">
            <p className="text-xs">{t.tooltips.unlinkTarget}</p>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    </div>
  )
}

/**
 * target name cell component
 */
function TargetNameCell({ 
  name, 
  targetId, 
  t,
}: { 
  name: string
  targetId: number
  t: OrgTargetsTranslations
}) {
  const [copied, setCopied] = React.useState(false)
  
  const handleCopy = async (e: React.MouseEvent) => {
    e.stopPropagation()
    try {
      await navigator.clipboard.writeText(name)
      setCopied(true)
      toast.success(t.tooltips.copied)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      // Fallback
    }
  }
  
  return (
    <div className="group flex items-start gap-1 flex-1 min-w-0">
      <Link
        href={`/target/${targetId}/overview/`}
        className="text-sm font-medium hover:text-primary hover:underline underline-offset-2 transition-colors text-left break-all leading-relaxed whitespace-normal"
      >
        {name}
      </Link>
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
}

/**
 * Create target table column definitions
 */
export const createTargetColumns = ({
  formatDate,
  navigate,
  handleDelete,
  t,
}: CreateColumnsProps): ColumnDef<Target>[] => {
  void formatDate
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
    accessorKey: "name",
    size: 350,
    minSize: 250,
    meta: { title: t.columns.targetName },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.targetName} />
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
    accessorKey: "type",
    size: 100,
    minSize: 80,
    meta: { title: t.columns.type },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.type} />
    ),
    cell: ({ row }) => {
      const type = row.getValue("type") as string | null
      if (!type) {
        return <span className="text-sm text-muted-foreground">-</span>
      }
      const typeMap: Record<string, { label: string; variant: "default" | "secondary" | "outline" }> = {
        domain: { label: t.types.domain, variant: "default" },
        ip: { label: t.types.ip, variant: "secondary" },
        cidr: { label: t.types.cidr, variant: "outline" },
      }
      const typeInfo = typeMap[type] || { label: type, variant: "secondary" as const }
      return (
        <Badge variant={typeInfo.variant}>
          {typeInfo.label}
        </Badge>
      )
    },
  },
  {
    id: "actions",
    size: 80,
    minSize: 80,
    maxSize: 80,
    enableResizing: false,
    cell: ({ row }) => (
      <TargetRowActions
        onView={() => navigate(`/target/${row.original.id}/overview/`)}
        onDelete={() => handleDelete(row.original)}
        t={t}
      />
    ),
    enableSorting: false,
    enableHiding: false,
  },
  ]
}
