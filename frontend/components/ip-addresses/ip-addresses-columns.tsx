"use client"

import React from "react"
import { ColumnDef } from "@tanstack/react-table"
import { Checkbox } from "@/components/ui/checkbox"
import { DataTableColumnHeader } from "@/components/shared/data-table/column-header"
import type { IPAddress } from "@/types/ip-address.types"
import { ExpandableBadgeList, ExpandableCell } from "@/components/shared/data-table/expandable-cell"
import { TimestampCell } from "@/components/shared/data-table/timestamp-cell"
import { textRole } from "@/lib/typography"

// Translation type definitions
export interface IPAddressTranslations {
  columns: {
    ipAddress: string
    hosts: string
    createdAt: string
    openPorts: string
  }
  actions: {
    selectAll: string
    selectRow: string
  }
  tooltips: {
    allHosts: string
    allOpenPorts: string
  }
}

interface CreateColumnsProps {
  formatDate: (value: string) => string
  t: IPAddressTranslations
  includeSelection?: boolean
}

export function createIPAddressColumns({
  formatDate,
  t,
  includeSelection = true,
}: CreateColumnsProps): ColumnDef<IPAddress>[] {
  const selectionColumn: ColumnDef<IPAddress> = {
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
    }

  const dataColumns: ColumnDef<IPAddress>[] = [
    {
      accessorKey: "ip",
      size: 150,
      minSize: 100,
      maxSize: 200,
      meta: {
        title: t.columns.ipAddress,
        orderBy: "ip",
        firstSortDirection: "asc",
        serverSortPerformance: "covered by idx_hpm_target_ip and idx_hpm_snap_scan_ip for scoped IP aggregation and sorting",
      },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.ipAddress} />
      ),
      cell: ({ row }) => (
        <ExpandableCell value={row.original.ip} textRoleName="tableCellPrimary" />
      ),
    },
    {
      accessorKey: "hosts",
      size: 200,
      minSize: 150,
      maxSize: 350,
      enableSorting: false,
      meta: { title: t.columns.hosts },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.hosts} />
      ),
      cell: ({ getValue }) => {
        const hosts = getValue<string[]>()
        const value = hosts?.length ? hosts.join("\n") : null
        return <ExpandableCell value={value} maxLines={3} textRoleName="tableCellSecondary" />
      },
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
        serverSortPerformance: "covered by idx_hpm_target_created_at and idx_hpm_snap_scan_created_at query-plan support for aggregated MIN(created_at) sorting",
      },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.createdAt} />
      ),
      cell: ({ getValue }) => {
        const value = getValue<string | undefined>()
        return <TimestampCell value={value ? formatDate(value) : "-"} />
      },
    },
    {
      accessorKey: "ports",
      size: 250,
      minSize: 150,
      maxSize: 360,
      enableSorting: false,
      meta: { title: t.columns.openPorts },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.openPorts} />
      ),
      cell: ({ getValue }) => {
        const ports = getValue<number[]>()
        
        if (!ports || ports.length === 0) {
          return <span className={textRole.tableCellSecondary}>-</span>
        }

        const sortedPorts = [...ports].sort((a, b) => a - b)
        return (
          <ExpandableBadgeList
            items={sortedPorts.map((port) => ({ id: String(port), name: String(port) }))}
            maxVisible={4}
            singleLinePreview
            variant="outline"
          />
        )
      },
    },
  ]

  return includeSelection ? [selectionColumn, ...dataColumns] : dataColumns
}
