"use client"

import { ColumnDef } from "@tanstack/react-table"
import { Checkbox } from "@/components/ui/checkbox"
import { Badge } from "@/components/ui/badge"
import {
  DropdownMenuItem,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu"
import { semanticIcons } from "@/components/icons"
import { DataTableColumnHeader } from "@/components/shared/data-table/column-header"
import { ExpandableCell } from "@/components/shared/data-table/expandable-cell"
import { DenseRowActionMenu } from "@/components/shared/data-table/menu-owners"
import { DenseRowActionButton } from "@/components/shared/data-table/row-actions"
import { TimestampCell } from "@/components/shared/data-table/timestamp-cell"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

import type { Organization } from "@/types/organization.types"
import { getOrganizationInitial } from "./organization-initial"
import { organizationTableColumnLayout } from "./organization-table-layout"

const ViewIcon = semanticIcons.action.view
const RunIcon = semanticIcons.action.run
const ScheduleIcon = semanticIcons.action.schedule
const DeleteIcon = semanticIcons.action.delete

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
  handleViewDetail: (org: Organization) => void
  handleDelete: (org: Organization) => void
  handleInitiateScan: (org: Organization) => void
  handleScheduleScan: (org: Organization) => void
  t: OrganizationTranslations
}

/**
 * Organization row actions component
 */
function OrganizationRowActions({ 
  onViewDetail,
  onInitiateScan,
  onScheduleScan,
  onDelete,
  t,
}: {
  onViewDetail: () => void
  onInitiateScan: () => void
  onScheduleScan: () => void
  onDelete: () => void
  t: OrganizationTranslations
}) {
  return (
    <DenseRowActionMenu
      ariaLabel={t.actions.openMenu}
      leadingActions={<DenseRowActionButton icon={<RunIcon aria-hidden="true" />} label={t.tooltips.initiateScan} onClick={onInitiateScan} />}
    >
      <DropdownMenuItem onClick={onViewDetail}>
        <ViewIcon />
        {t.tooltips.organizationDetails}
      </DropdownMenuItem>
      <DropdownMenuSeparator />
      <DropdownMenuItem onClick={onScheduleScan}>
        <ScheduleIcon />
        {t.actions.scheduleScan}
      </DropdownMenuItem>
      <DropdownMenuSeparator />
      <DropdownMenuItem
        onClick={onDelete}
        variant="destructive"
      >
        <DeleteIcon />
        {t.actions.delete}
      </DropdownMenuItem>
    </DenseRowActionMenu>
  )
}

/**
 * Create organization table column definitions
 */
export const createOrganizationColumns = ({
  formatDate,
  handleViewDetail,
  handleDelete,
  handleInitiateScan,
  handleScheduleScan,
  t,
}: CreateColumnsProps): ColumnDef<Organization>[] => [
  {
    id: "select",
    size: organizationTableColumnLayout.select.size,
    minSize: organizationTableColumnLayout.select.minSize,
    maxSize: organizationTableColumnLayout.select.maxSize,
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
    size: organizationTableColumnLayout.name.size,
    minSize: organizationTableColumnLayout.name.minSize,
    meta: {
      title: t.columns.organization,
      orderBy: "displayName",
      firstSortDirection: "asc",
      serverSortPerformance: "covered by idx_org_name_id_active for active organizations",
    },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.organization} />
    ),
    cell: ({ row }) => {
      const organizationName = row.getValue("name") as string

      return (
        <div className="group flex min-w-0 items-center gap-3">
          <span
            aria-hidden="true"
            className={cn(
              "flex size-9 shrink-0 items-center justify-center rounded-full border border-dashed border-border bg-transparent transition-colors",
              textRole.tableCellPrimary,
              "text-muted-foreground group-hover:border-solid group-hover:text-foreground group-data-[state=selected]:border-solid group-data-[state=selected]:text-foreground"
            )}
          >
            {getOrganizationInitial(organizationName)}
          </span>
          <div className="min-w-0 flex-1">
            <div className={cn("truncate", textRole.tableCellPrimary)}>
              {organizationName}
            </div>
            <ExpandableCell
              value={row.original.description}
              variant="muted"
              textRoleName="tableCellSecondary"
              // Keep the initial comfortable row stable while retaining the
              // shared explicit expansion action for long descriptions.
              maxLines={1}
            />
          </div>
        </div>
      )
    },
  },
  {
    accessorKey: "targetCount",
    size: organizationTableColumnLayout.targetCount.size,
    minSize: organizationTableColumnLayout.targetCount.minSize,
    maxSize: organizationTableColumnLayout.targetCount.maxSize,
    meta: { title: t.columns.totalTargets },
    enableSorting: false,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.totalTargets} />
    ),
    cell: ({ row }) => {
      const targetCount = row.original.targetCount ?? 0
      return (
        <div className={textRole.tableCellSecondary}>
          <Badge variant="secondary" className="text-xs">
            {targetCount}
          </Badge>
        </div>
      )
    },
  },
  {
    accessorKey: "createdAt",
    size: organizationTableColumnLayout.createdAt.size,
    minSize: organizationTableColumnLayout.createdAt.minSize,
    maxSize: organizationTableColumnLayout.createdAt.maxSize,
    meta: {
      title: t.columns.added,
      orderBy: "createdAt",
      firstSortDirection: "desc",
      serverSortPerformance: "covered by idx_org_created_at_id_active for active organizations",
    },
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
        <TimestampCell
          value={createdAt && !isZeroTime ? formatDate(createdAt) : "-"}
          className="truncate"
        />
      )
    },
  },
  {
    id: "actions",
    size: organizationTableColumnLayout.actions.size,
    minSize: organizationTableColumnLayout.actions.minSize,
    maxSize: organizationTableColumnLayout.actions.maxSize,
    enableResizing: false,
    cell: ({ row }) => (
      <div className="flex justify-end">
        <OrganizationRowActions
          onViewDetail={() => handleViewDetail(row.original)}
          onInitiateScan={() => handleInitiateScan(row.original)}
          onScheduleScan={() => handleScheduleScan(row.original)}
          onDelete={() => handleDelete(row.original)}
          t={t}
        />
      </div>
    ),
    enableSorting: false,
    enableHiding: false,
  },
]
