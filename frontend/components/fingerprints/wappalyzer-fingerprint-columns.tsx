"use client"

import React from "react"
import { ColumnDef } from "@tanstack/react-table"
import { Checkbox } from "@/components/ui/checkbox"
import { DataTableColumnHeader } from "@/components/ui/data-table/column-header"
import { ExpandableCell, ExpandableMonoCell } from "@/components/ui/data-table/expandable-cell"
import { ChevronDown, ChevronUp } from "@/components/icons"
import { useTranslations } from "next-intl"
import type { WappalyzerFingerprint } from "@/types/fingerprint.types"

interface ColumnOptions {
  formatDate: (date: string) => string
  selectLabels: {
    selectAll: string
    selectRow: string
  }
}

interface RuleItem {
  key: string
  value: unknown
}

/**
 * Extract all rules from fingerprint (keeping original format)
 */
function extractRules(fp: WappalyzerFingerprint): RuleItem[] {
  const rules: RuleItem[] = []
  const ruleKeys = ['cookies', 'headers', 'scriptSrc', 'js', 'meta', 'html'] as const
  
  for (const key of ruleKeys) {
    const value = fp[key]
    if (value && (Array.isArray(value) ? value.length > 0 : Object.keys(value).length > 0)) {
      rules.push({ key, value })
    }
  }
  
  return rules
}

/**
 * Rules list cell - displays raw JSON format
 */
function RulesCell({ fp }: { fp: WappalyzerFingerprint }) {
  const t = useTranslations("tooltips")
  const [expanded, setExpanded] = React.useState(false)
  const rules = extractRules(fp)
  
  if (rules.length === 0) {
    return <span className="text-muted-foreground">-</span>
  }
  
  const displayRules = expanded ? rules : rules.slice(0, 2)
  const hasMore = rules.length > 2
  
  return (
    <div className="flex flex-col gap-1">
      <div className="font-mono text-xs space-y-0.5">
        {displayRules.map((rule, idx) => (
          <div key={idx} className={expanded ? "break-all" : "truncate"}>
            {`${JSON.stringify(rule.key)}: `}{JSON.stringify(rule.value)}
          </div>
        ))}
      </div>
      {hasMore && (
        <button type="button"
          onClick={() => setExpanded(!expanded)}
          className="text-xs text-primary hover:underline self-start inline-flex items-center gap-0.5"
        >
          {expanded ? (
            <>
              <ChevronUp className="h-3 w-3" />
              <span>{t("collapse")}</span>
            </>
          ) : (
            <>
              <ChevronDown className="h-3 w-3" />
              <span>{t("expand")}</span>
            </>
          )}
        </button>
      )}
    </div>
  )
}

/**
 * Create Wappalyzer fingerprint table column definitions
 */
export function createWappalyzerFingerprintColumns({
  formatDate,
  selectLabels,
}: ColumnOptions): ColumnDef<WappalyzerFingerprint>[] {
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
      size: 180,
    },
    {
      accessorKey: "cats",
      meta: { title: "Categories" },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Categories" />
      ),
      cell: ({ row }) => {
        const cats = row.getValue("cats") as number[]
        if (!cats || cats.length === 0) return <span className="text-muted-foreground">-</span>
        return <ExpandableMonoCell value={JSON.stringify(cats)} maxLines={1} />
      },
      enableResizing: true,
      size: 100,
    },
    {
      id: "rules",
      meta: { title: "Rules" },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Rules" />
      ),
      cell: ({ row }) => <RulesCell fp={row.original} />,
      enableResizing: true,
      size: 350,
    },
    {
      accessorKey: "implies",
      meta: { title: "Implies" },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Implies" />
      ),
      cell: ({ row }) => {
        const implies = row.getValue("implies") as string[]
        if (!implies || implies.length === 0) return <span className="text-muted-foreground">-</span>
        return <ExpandableMonoCell value={implies.join(", ")} maxLines={1} />
      },
      enableResizing: true,
      size: 150,
    },
    {
      accessorKey: "description",
      meta: { title: "Description" },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Description" />
      ),
      cell: ({ row }) => <ExpandableCell value={row.getValue("description")} maxLines={2} />,
      enableResizing: true,
      size: 250,
    },
    {
      accessorKey: "website",
      meta: { title: "Website" },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Website" />
      ),
      cell: ({ row }) => <ExpandableCell value={row.getValue("website")} variant="url" maxLines={1} />,
      enableResizing: true,
      size: 180,
    },
    {
      accessorKey: "cpe",
      meta: { title: "CPE" },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="CPE" />
      ),
      cell: ({ row }) => (
        <ExpandableMonoCell value={row.getValue("cpe")} maxLines={1} />
      ),
      enableResizing: true,
      size: 150,
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
