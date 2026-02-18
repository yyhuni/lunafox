"use client"

import React from "react"
import { ColumnDef } from "@tanstack/react-table"
import { Checkbox } from "@/components/ui/checkbox"
import { Badge } from "@/components/ui/badge"
import { DataTableColumnHeader } from "@/components/ui/data-table/column-header"
import { ExpandableCell } from "@/components/ui/data-table/expandable-cell"
import { ChevronDown, ChevronUp } from "@/components/icons"
import { useTranslations } from "next-intl"
import type { EholeFingerprint } from "@/types/fingerprint.types"

interface ColumnOptions {
  formatDate: (date: string) => string
  selectLabels: {
    selectAll: string
    selectRow: string
  }
}

/**
 * Keyword list cell - displays 3 by default, expandable for more
 */
function KeywordListCell({ keywords }: { keywords: string[] }) {
  const t = useTranslations("tooltips")
  const [expanded, setExpanded] = React.useState(false)
  
  if (!keywords || keywords.length === 0) return <span className="text-muted-foreground">-</span>
  
  const displayKeywords = expanded ? keywords : keywords.slice(0, 3)
  const hasMore = keywords.length > 3
  
  return (
    <div className="flex flex-col gap-1">
      <div className="font-mono text-xs space-y-0.5">
        {displayKeywords.map((kw, idx) => (
          <div key={idx} className={expanded ? "break-all" : "truncate"}>
            {kw}
          </div>
        ))}
      </div>
      {hasMore && (
        <button type="button"
          onClick={() => setExpanded(!expanded)}
          className="text-xs text-primary hover:underline self-start flex items-center gap-1"
        >
          {expanded ? (
            <>
              <ChevronUp className="h-3 w-3" />
              {t("collapse")}
            </>
          ) : (
            <>
              <ChevronDown className="h-3 w-3" />
              {t("expand")}
            </>
          )}
        </button>
      )}
    </div>
  )
}

/**
 * Create EHole fingerprint table column definitions
 */
export function createEholeFingerprintColumns({
  formatDate,
  selectLabels,
}: ColumnOptions): ColumnDef<EholeFingerprint>[] {
  return [
    {
      id: "select",
      header: ({ table }) => (
        <Checkbox
          checked={
            table.getIsAllPageRowsSelected() ||
            (table.getIsSomePageRowsSelected() && "indeterminate")
          }
          onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
          aria-label={selectLabels.selectAll}
        />
      ),
      cell: ({ row }) => (
        <Checkbox
          checked={row.getIsSelected()}
          onCheckedChange={(value) => row.toggleSelected(!!value)}
          aria-label={selectLabels.selectRow}
        />
      ),
      enableSorting: false,
      enableHiding: false,
      enableResizing: false,
      size: 40,
    },
    {
      accessorKey: "cms",
      meta: { title: "CMS" },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="CMS" />
      ),
      cell: ({ row }) => (
        <ExpandableCell value={row.getValue("cms")} maxLines={2} />
      ),
      enableResizing: true,
      size: 200,
    },
    {
      accessorKey: "method",
      meta: { title: "Method" },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Method" />
      ),
      cell: ({ row }) => {
        const method = row.getValue("method") as string
        return (
          <Badge variant="outline" className="font-mono text-xs">
            {method}
          </Badge>
        )
      },
      enableResizing: false,
      size: 120,
    },
    {
      accessorKey: "location",
      meta: { title: "Location" },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Location" />
      ),
      cell: ({ row }) => {
        const location = row.getValue("location") as string
        return (
          <Badge variant="secondary" className="font-mono text-xs">
            {location}
          </Badge>
        )
      },
      enableResizing: false,
      size: 100,
    },
    {
      accessorKey: "keyword",
      meta: { title: "Keyword" },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Keyword" />
      ),
      cell: ({ row }) => <KeywordListCell keywords={row.getValue("keyword") || []} />,
      enableResizing: true,
      size: 300,
    },
    {
      accessorKey: "type",
      meta: { title: "Type" },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Type" />
      ),
      cell: ({ row }) => {
        const type = row.getValue("type") as string
        if (!type || type === "-") return "-"
        return <Badge variant="outline">{type}</Badge>
      },
      enableResizing: false,
      size: 100,
    },
    {
      accessorKey: "isImportant",
      meta: { title: "Important" },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Important" />
      ),
      cell: ({ row }) => {
        const isImportant = row.getValue("isImportant")
        return <span>{String(isImportant)}</span>
      },
      enableResizing: false,
      size: 100,
    },
    {
      accessorKey: "createdAt",
      meta: { title: "Created" },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Created" />
      ),
      cell: ({ row }) => {
        const date = row.getValue("createdAt") as string
        return (
          <div className="text-sm text-muted-foreground">
            {formatDate(date)}
          </div>
        )
      },
      enableResizing: false,
      size: 160,
    },
  ]
}
