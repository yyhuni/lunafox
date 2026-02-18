"use client"

import React from "react"
import { ColumnDef } from "@tanstack/react-table"
import { Checkbox } from "@/components/ui/checkbox"
import { Badge } from "@/components/ui/badge"
import { DataTableColumnHeader } from "@/components/ui/data-table/column-header"
import { ExpandableCell } from "@/components/ui/data-table/expandable-cell"
import { ChevronDown, ChevronUp } from "@/components/icons"
import { useTranslations } from "next-intl"
import type { FingersFingerprint, FingersRule } from "@/types/fingerprint.types"

interface ColumnOptions {
  formatDate: (date: string) => string
  selectLabels: {
    selectAll: string
    selectRow: string
  }
}

/**
 * Tag list cell - displays tags as badges
 */
function TagListCell({ tags }: { tags: string[] }) {
  const t = useTranslations("tooltips")
  const [expanded, setExpanded] = React.useState(false)
  
  if (!tags || tags.length === 0) return <span className="text-muted-foreground">-</span>
  
  const displayTags = expanded ? tags : tags.slice(0, 3)
  const hasMore = tags.length > 3
  
  return (
    <div className="flex flex-col gap-1">
      <div className="flex flex-wrap gap-1">
        {displayTags.map((tag, idx) => (
          <Badge key={idx} variant="secondary" className="text-xs">
            {tag}
          </Badge>
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
 * Extract rule items from rules array
 */
interface RuleItem {
  key: string
  value: unknown
}

function extractRuleItems(rules: FingersRule[]): RuleItem[] {
  const items: RuleItem[] = []
  
  rules.forEach((rule) => {
    if (Array.isArray(rule.regexps)) {
      rule.regexps.forEach((regexp, index) => {
        items.push({ key: `regexps[${index}]`, value: regexp })
      })
    }
  })
  
  return items
}

/**
 * Rules cell - displays rules in JSON format (matching Wappalyzer style)
 */
function RulesCell({ rules }: { rules: FingersRule[] }) {
  const t = useTranslations("tooltips")
  const [expanded, setExpanded] = React.useState(false)
  const ruleItems = extractRuleItems(rules)
  
  if (ruleItems.length === 0) {
    return <span className="text-muted-foreground">-</span>
  }
  
  const displayItems = expanded ? ruleItems : ruleItems.slice(0, 2)
  const hasMore = ruleItems.length > 2
  
  return (
    <div className="flex flex-col gap-1 w-full">
      <div className="font-mono text-xs space-y-0.5 w-full">
        {displayItems.map((item, idx) => (
          <div key={idx} className={expanded ? "break-all w-full" : "truncate w-full"}>
            {`${JSON.stringify(item.key)}: `}{JSON.stringify(item.value)}
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
 * Port list cell - displays ports
 */
function PortListCell({ ports }: { ports: number[] }) {
  if (!ports || ports.length === 0) return <span className="text-muted-foreground">-</span>
  return (
    <div className="flex flex-wrap gap-1">
      {ports.slice(0, 5).map((port, idx) => (
        <Badge key={idx} variant="outline" className="font-mono text-xs">
          {port}
        </Badge>
      ))}
      {ports.length > 5 && (
        <Badge variant="secondary" className="text-xs">
          +{ports.length - 5}
        </Badge>
      )}
    </div>
  )
}

/**
 * Create Fingers fingerprint table column definitions
 */
export function createFingersFingerprintColumns({
  formatDate,
  selectLabels,
}: ColumnOptions): ColumnDef<FingersFingerprint>[] {
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
      accessorKey: "focus",
      meta: { title: "Focus" },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Focus" />
      ),
      cell: ({ row }) => {
        const focus = row.getValue("focus") as boolean
        return (
          <span className={focus ? "text-foreground" : "text-muted-foreground"}>
            {String(focus)}
          </span>
        )
      },
      enableResizing: false,
      size: 80,
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
      accessorKey: "link",
      meta: { title: "Link" },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Link" />
      ),
      cell: ({ row }) => {
        const link = row.getValue("link") as string
        if (!link) return <span className="text-muted-foreground">-</span>
        return (
          <a 
            href={link} 
            target="_blank" 
            rel="noopener noreferrer"
            className="text-primary hover:underline"
          >
            <ExpandableCell value={link} variant="url" maxLines={1} />
          </a>
        )
      },
      enableResizing: true,
      size: 200,
    },
    {
      accessorKey: "rule",
      meta: { title: "Rule" },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Rule" />
      ),
      cell: ({ row }) => <RulesCell rules={row.getValue("rule") || []} />,
      enableResizing: true,
      size: 300,
    },
    {
      accessorKey: "tag",
      meta: { title: "Tag" },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Tag" />
      ),
      cell: ({ row }) => <TagListCell tags={row.getValue("tag") || []} />,
      enableResizing: true,
      size: 200,
    },
    {
      accessorKey: "defaultPort",
      meta: { title: "Default Port" },
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Default Port" />
      ),
      cell: ({ row }) => <PortListCell ports={row.getValue("defaultPort") || []} />,
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
