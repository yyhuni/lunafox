"use client"

import type { ColumnDef } from "@tanstack/react-table"
import { DataTableColumnHeader } from "@/components/shared/data-table/column-header"
import { ExpandableCell, ExpandableMonoCell } from "@/components/shared/data-table/expandable-cell"

type FingerprintValueColumnOptions = {
  formatValue?: (value: unknown) => unknown
  mono?: boolean
}

/** Fixed native fields use the same three-line, independently expandable view. */
export function fingerprintValueColumn<T extends object>(key: keyof T & string, title: string, options: boolean | FingerprintValueColumnOptions = false): ColumnDef<T> {
  const { formatValue, mono = false } = typeof options === "boolean" ? { mono: options } : options
  return {
    accessorKey: key,
    meta: { title, widthPolicy: { mode: "flex", flex: 1 } },
    header: ({ column }) => <DataTableColumnHeader column={column} title={title} />,
    cell: ({ row }) => {
      const value = formatValue ? formatValue(row.getValue(key)) : row.getValue(key)
      return mono
        ? <ExpandableMonoCell value={value} maxLines={3} textRoleName="tableCellSecondary" />
        : <ExpandableCell value={value} maxLines={3} textRoleName="tableCellSecondary" />
    },
    enableResizing: true,
    size: 240,
    minSize: 180,
    maxSize: 720,
  }
}
