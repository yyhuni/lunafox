"use client"

import { ColumnDef } from "@tanstack/react-table"
import { Checkbox } from "@/components/ui/checkbox"
import { DataTableColumnHeader } from "@/components/ui/data-table/column-header"
import { ExpandableCell } from "@/components/ui/data-table/expandable-cell"
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
    meta: { title: t.columns.subdomain },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.subdomain} />
    ),
    cell: ({ row }) => (
      <ExpandableCell value={row.getValue("name")} />
    ),
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
    cell: ({ getValue }) => {
      const value = getValue<string | undefined>()
      return value ? formatDate(value) : "-"
    },
  },
]
