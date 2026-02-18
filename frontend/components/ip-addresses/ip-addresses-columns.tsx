"use client"

import React from "react"
import { ColumnDef } from "@tanstack/react-table"
import { Checkbox } from "@/components/ui/checkbox"
import { Badge } from "@/components/ui/badge"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover"
import { DataTableColumnHeader } from "@/components/ui/data-table/column-header"
import type { IPAddress } from "@/types/ip-address.types"
import { ExpandableCell } from "@/components/ui/data-table/expandable-cell"

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
}

export function createIPAddressColumns({
  formatDate,
  t,
}: CreateColumnsProps): ColumnDef<IPAddress>[] {
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
      accessorKey: "ip",
      size: 150,
      minSize: 100,
      maxSize: 200,
      meta: { title: t.columns.ipAddress },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.ipAddress} />
      ),
      cell: ({ row }) => (
        <ExpandableCell value={row.original.ip} />
      ),
    },
    {
      accessorKey: "hosts",
      size: 200,
      minSize: 150,
      maxSize: 350,
      meta: { title: t.columns.hosts },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.hosts} />
      ),
      cell: ({ getValue }) => {
        const hosts = getValue<string[]>()
        const value = hosts?.length ? hosts.join("\n") : null
        return <ExpandableCell value={value} maxLines={3} />
      },
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
    {
      accessorKey: "ports",
      size: 250,
      minSize: 150,
      meta: { title: t.columns.openPorts },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t.columns.openPorts} />
      ),
      cell: ({ getValue }) => {
        const ports = getValue<number[]>()
        
        if (!ports || ports.length === 0) {
          return <span className="text-muted-foreground">-</span>
        }

        const sortedPorts = [...ports].sort((a, b) => a - b)
        const displayPorts = sortedPorts.slice(0, 8)
        const hasMore = sortedPorts.length > 8

        return (
          <div className="flex flex-wrap items-center gap-1.5">
            {displayPorts.map((port, index) => (
              <Badge 
                key={index} 
                variant="outline"
                className="text-xs font-mono"
              >
                {port}
              </Badge>
            ))}
            {hasMore && (
              <Popover>
                <PopoverTrigger asChild>
                  <Badge variant="secondary" className="text-xs cursor-pointer hover:bg-muted">
                    +{sortedPorts.length - 8} more
                  </Badge>
                </PopoverTrigger>
                <PopoverContent className="w-80 p-3">
                  <div className="space-y-2">
                    <h4 className="font-medium text-sm">{t.tooltips.allOpenPorts} ({sortedPorts.length})</h4>
                    <div className="flex flex-wrap gap-1.5 max-h-48 overflow-y-auto">
                      {sortedPorts.map((port, index) => (
                        <Badge 
                          key={index} 
                          variant="outline"
                          className="text-xs font-mono"
                        >
                          {port}
                        </Badge>
                      ))}
                    </div>
                  </div>
                </PopoverContent>
              </Popover>
            )}
          </div>
        )
      },
    },
  ]
}
