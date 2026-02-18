"use client"

import React from "react"
import { ColumnDef } from "@tanstack/react-table"
import { Checkbox } from "@/components/ui/checkbox"
import { DataTableColumnHeader } from "@/components/ui/data-table/column-header"
import { ExpandableCell, ExpandableMonoCell } from "@/components/ui/data-table/expandable-cell"
import { ChevronDown, ChevronUp } from "@/components/icons"
import { useTranslations } from "next-intl"
import type { GobyFingerprint, GobyRule } from "@/types/fingerprint.types"

interface ColumnOptions {
  formatDate: (date: string) => string
  selectLabels: {
    selectAll: string
    selectRow: string
  }
}

/**
 * Rule details cell component - displays raw JSON data
 */
function RuleDetailsCell({ rules }: { rules: GobyRule[] }) {
  const t = useTranslations("tooltips")
  const [expanded, setExpanded] = React.useState(false)
  
  if (!rules || rules.length === 0) return <span className="text-muted-foreground">-</span>
  
  const displayRules = expanded ? rules : rules.slice(0, 2)
  const hasMore = rules.length > 2
  
  return (
    <div className="flex flex-col gap-1">
      <div className="font-mono text-xs space-y-0.5">
        {displayRules.map((r, idx) => (
          <div key={idx} className={expanded ? "break-all" : "truncate"}>
            {JSON.stringify(r)}
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
 * Create Goby fingerprint table column definitions
 */
export function createGobyFingerprintColumns({
  formatDate,
  selectLabels,
}: ColumnOptions): ColumnDef<GobyFingerprint>[] {
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
      accessorKey: "name",
      meta: { title: "Name" },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Name" />
      ),
      cell: ({ row }) => (
        <ExpandableCell value={row.getValue("name")} maxLines={2} />
      ),
      enableResizing: true,
      size: 200,
    },
    {
      accessorKey: "logic",
      meta: { title: "Logic" },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Logic" />
      ),
      cell: ({ row }) => (
        <ExpandableMonoCell value={row.getValue("logic")} maxLines={1} />
      ),
      enableResizing: false,
      size: 100,
    },
    {
      accessorKey: "rule",
      meta: { title: "Rules" },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Rules" />
      ),
      cell: ({ row }) => {
        const rules = row.getValue("rule") as GobyRule[]
        return <span>{rules?.length || 0}</span>
      },
      enableResizing: false,
      size: 80,
    },
    {
      id: "ruleDetails",
      meta: { title: "Rule Details" },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Rule Details" />
      ),
      cell: ({ row }) => <RuleDetailsCell rules={row.original.rule || []} />,
      enableResizing: true,
      size: 300,
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
