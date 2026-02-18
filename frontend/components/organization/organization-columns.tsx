"use client"

import { ColumnDef } from "@tanstack/react-table"
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
import { MoreHorizontal, Play, Calendar, Edit, Trash2, Eye } from "@/components/icons"
import { DataTableColumnHeader } from "@/components/ui/data-table/column-header"
import { ExpandableCell } from "@/components/ui/data-table/expandable-cell"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import Link from "next/link"

import type { Organization } from "@/types/organization.types"

// Translation type definitions
export interface OrganizationTranslations {
  columns: {
    organization: string
    description: string
    totalTargets: string
    added: string
  }
  actions: {
    scheduleScan: string
    editOrganization: string
    delete: string
    openMenu: string
    selectAll: string
    selectRow: string
  }
  tooltips: {
    organizationDetails: string
    initiateScan: string
  }
}

// Column creation function parameter types
interface CreateColumnsProps {
  formatDate: (dateString: string) => string
  handleEdit: (org: Organization) => void
  handleDelete: (org: Organization) => void
  handleInitiateScan: (org: Organization) => void
  handleScheduleScan: (org: Organization) => void
  t: OrganizationTranslations
}

/**
 * Organization row actions component
 */
function OrganizationRowActions({ 
  onScheduleScan,
  onEdit, 
  onDelete,
  t,
}: {
  onScheduleScan: () => void
  onEdit: () => void
  onDelete: () => void
  t: OrganizationTranslations
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
        <DropdownMenuItem onClick={onScheduleScan}>
          <Calendar />
          {t.actions.scheduleScan}
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem onClick={onEdit}>
          <Edit />
          {t.actions.editOrganization}
        </DropdownMenuItem>
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
 * Create organization table column definitions
 */
export const createOrganizationColumns = ({
  formatDate,
  handleEdit,
  handleDelete,
  handleInitiateScan,
  handleScheduleScan,
  t,
}: CreateColumnsProps): ColumnDef<Organization>[] => [
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
    size: 200,
    minSize: 150,
    meta: { title: t.columns.organization },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.organization} />
    ),
    cell: ({ row }) => {
      const organization = row.original
      return (
        <div className="flex-1 min-w-0">
          <Link 
            href={`/organization/${organization.id}`}
            className="text-sm font-medium hover:text-primary hover:underline underline-offset-2 transition-colors break-all leading-relaxed whitespace-normal"
          >
            {row.getValue("name")}
          </Link>
        </div>
      )
    },
  },
  {
    accessorKey: "description",
    size: 300,
    minSize: 200,
    meta: { title: t.columns.description },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.description} />
    ),
    cell: ({ row }) => (
      <ExpandableCell value={row.getValue("description")} variant="muted" />
    ),
  },
  {
    accessorKey: "targetCount",
    size: 120,
    minSize: 80,
    meta: { title: t.columns.totalTargets },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.totalTargets} />
    ),
    cell: ({ row }) => {
      const targetCount = row.original.targetCount ?? 0
      return (
        <div className="text-sm">
          <Badge variant="secondary" className="text-xs">
            {targetCount}
          </Badge>
        </div>
      )
    },
  },
  {
    accessorKey: "createdAt",
    size: 150,
    minSize: 120,
    meta: { title: t.columns.added },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.added} />
    ),
    cell: ({ row }) => {
      const createdAt = row.getValue("createdAt") as string | undefined
      const isZeroTime = createdAt && (
        createdAt === "0001-01-01T00:00:00Z" ||
        createdAt.startsWith("0001-01-01")
      )

      return (
        <div className="text-sm text-muted-foreground">
          {createdAt && !isZeroTime ? formatDate(createdAt) : (
            <span className="text-muted-foreground">-</span>
          )}
        </div>
      )
    },
  },
  {
    id: "actions",
    size: 120,
    minSize: 120,
    maxSize: 120,
    enableResizing: false,
    cell: ({ row }) => (
      <div className="flex items-center gap-1">
        <TooltipProvider delayDuration={300}>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                asChild
                variant="ghost"
                size="icon"
                className="h-8 w-8"
                aria-label={t.tooltips.organizationDetails}
              >
                <Link href={`/organization/${row.original.id}`}>
                  <Eye className="h-4 w-4" />
                </Link>
              </Button>
            </TooltipTrigger>
            <TooltipContent side="top">
              <p className="text-xs">{t.tooltips.organizationDetails}</p>
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
                onClick={() => handleInitiateScan(row.original)}
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

        <OrganizationRowActions
          onScheduleScan={() => handleScheduleScan(row.original)}
          onEdit={() => handleEdit(row.original)}
          onDelete={() => handleDelete(row.original)}
          t={t}
        />
      </div>
    ),
    enableSorting: false,
    enableHiding: false,
  },
]
