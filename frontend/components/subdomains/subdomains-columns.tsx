"use client"

import { ColumnDef } from "@tanstack/react-table"
import { Checkbox } from "@/components/ui/checkbox"
import { DataTableColumnHeader } from "@/components/shared/data-table/column-header"
import { ExpandableCell } from "@/components/shared/data-table/expandable-cell"
import { TimestampCell } from "@/components/shared/data-table/timestamp-cell"
import type { Subdomain } from "@/types/subdomain.types"

// Translation type definitions
export interface SubdomainTranslations {
  columns: {
    subdomain: string
    createdAt: string
  }
  actions: {
    selectAll: string
    selectRow: string
  }
}

interface CreateColumnsProps {
  formatDate: (dateString: string) => string
  t: SubdomainTranslations
}

/**
 * Create subdomain table column definitions
 */
export const createSubdomainColumns = ({
  formatDate,
  t,
}: CreateColumnsProps): ColumnDef<Subdomain>[] => [
  {
    id: "select",
    size: 40,
    minSize: 40,
    maxSize: 40,
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
    size: 350,
    minSize: 250,
    meta: {
      title: t.columns.subdomain,
      orderBy: "dnsName",
      firstSortDirection: "asc",
      serverSortPerformance: "covered by idx_subdomain_target_dns_name_id and idx_subdomain_snap_scan_dns_name_id for parent-scoped sorting",
    },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.subdomain} />
    ),
    cell: ({ row }) => (
      <ExpandableCell value={row.getValue("name")} textRoleName="tableCellPrimary" />
    ),
  },
  {
    accessorKey: "createdAt",
    size: 176,
    minSize: 176,
    maxSize: 220,
    enableResizing: false,
    meta: {
      title: t.columns.createdAt,
      orderBy: "createdAt",
      firstSortDirection: "desc",
      serverSortPerformance: "covered by idx_subdomain_target_created_at_id and idx_subdomain_snap_scan_created_at_id for parent-scoped sorting",
    },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.createdAt} />
    ),
    cell: ({ getValue }) => {
      const value = getValue<string | undefined>()
      return <TimestampCell value={value ? formatDate(value) : "-"} />
    },
  },
]
