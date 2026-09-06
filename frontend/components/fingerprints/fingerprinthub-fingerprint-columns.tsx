"use client"

import type { ColumnDef } from "@tanstack/react-table"

import { DataTableColumnHeader } from "@/components/shared/data-table/column-header"
import { ExpandableCell } from "@/components/shared/data-table/expandable-cell"
import { SingleBadgeCell } from "@/components/shared/data-table/single-badge-cell"
import { TimestampCell } from "@/components/shared/data-table/timestamp-cell"
import { Checkbox } from "@/components/ui/checkbox"
import { formatFingerprintFieldValue } from "./fingerprint-field-display"
import { fingerprintValueColumn } from "./fingerprint-value-column"
import type { FingerPrintHubFingerprint } from "@/types/fingerprint.types"

interface ColumnOptions {
  formatDate: (date: string) => string
  unsetLabel: string
  selectLabels: { selectAll: string; selectRow: string }
  getColumnLabel: (key: string) => string
  getFormLabel: (key: string) => string
}

/** Every fixed FingerPrintHub business field is a default-visible collection projection. */
export function createFingerPrintHubFingerprintColumns({ formatDate, unsetLabel, selectLabels, getColumnLabel, getFormLabel }: ColumnOptions): ColumnDef<FingerPrintHubFingerprint>[] {
  return [
    {
      id: "select",
      header: ({ table }) => <Checkbox checked={table.getIsAllPageRowsSelected()} indeterminate={!table.getIsAllPageRowsSelected() && table.getIsSomePageRowsSelected()} onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)} aria-label={selectLabels.selectAll} />,
      cell: ({ row }) => <Checkbox checked={row.getIsSelected()} onCheckedChange={(value) => row.toggleSelected(!!value)} aria-label={selectLabels.selectRow} />,
      enableSorting: false,
      enableHiding: false,
      enableResizing: false,
      size: 40,
      minSize: 40,
      maxSize: 40,
    },
    {
      accessorKey: "displayName",
      meta: { title: getColumnLabel("name"), orderBy: "displayName", firstSortDirection: "asc", serverSortPerformance: "idx_fingerprint_fingerprinthub_name_resource_id", widthPolicy: { mode: "flex", flex: 1, fill: true } },
      header: ({ column }) => <DataTableColumnHeader column={column} title={getColumnLabel("name")} />,
      cell: ({ row }) => <ExpandableCell value={row.getValue("displayName")} maxLines={2} textRoleName="tableCellPrimary" />,
      enableResizing: true,
      size: 320,
      minSize: 220,
      maxSize: 720,
    },
    fingerprintValueColumn<FingerPrintHubFingerprint>("fingerprintId", getFormLabel("fpId"), true),
    fingerprintValueColumn<FingerPrintHubFingerprint>("author", getFormLabel("author"), { formatValue: (value) => formatFingerprintFieldValue("fingerprinthub", "author", value as FingerPrintHubFingerprint["author"]) }),
    fingerprintValueColumn<FingerPrintHubFingerprint>("tags", getFormLabel("tags"), { formatValue: (value) => formatFingerprintFieldValue("fingerprinthub", "tags", value as FingerPrintHubFingerprint["tags"]) }),
    {
      accessorKey: "severity",
      meta: { title: getFormLabel("severity"), singleBadge: true },
      header: ({ column }) => <DataTableColumnHeader column={column} title={getFormLabel("severity")} />,
      cell: ({ row }) => <SingleBadgeCell value={(row.getValue("severity") as string | null) ?? unsetLabel} />,
      enableResizing: false,
      size: 116,
      minSize: 116,
      maxSize: 150,
    },
    fingerprintValueColumn<FingerPrintHubFingerprint>("metadata", getFormLabel("metadata"), true),
    fingerprintValueColumn<FingerPrintHubFingerprint>("http", getFormLabel("http"), true),
    fingerprintValueColumn<FingerPrintHubFingerprint>("sourceFile", getFormLabel("sourceFile")),
    {
      accessorKey: "createdAt",
      meta: { title: getColumnLabel("created"), orderBy: "createdAt", firstSortDirection: "desc", serverSortPerformance: "idx_fingerprint_fingerprinthub_created_at_resource_id" },
      header: ({ column }) => <DataTableColumnHeader column={column} title={getColumnLabel("created")} />,
      cell: ({ row }) => <TimestampCell value={formatDate(row.getValue("createdAt") as string)} />,
      enableResizing: false,
      size: 176,
      minSize: 176,
      maxSize: 220,
    },
  ]
}
