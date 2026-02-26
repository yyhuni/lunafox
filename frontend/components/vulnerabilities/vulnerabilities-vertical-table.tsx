"use client"

import React, { useCallback, useEffect } from "react"
import { useTranslations } from "next-intl"
import { Badge } from "@/components/ui/badge"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { CheckCircle2, Circle, Info } from "@/components/icons"
import { cn } from "@/lib/utils"
import { SEVERITY_STYLES } from "@/lib/severity-config"
import type { Vulnerability, VulnerabilitySeverity } from "@/types/vulnerability.types"

const SEVERITY_CONFIG: Record<VulnerabilitySeverity, { className: string; label: string }> = {
  critical: { className: SEVERITY_STYLES.critical.className, label: "Critical" },
  high: { className: SEVERITY_STYLES.high.className, label: "High" },
  medium: { className: SEVERITY_STYLES.medium.className, label: "Medium" },
  low: { className: SEVERITY_STYLES.low.className, label: "Low" },
  info: { className: SEVERITY_STYLES.info.className, label: "Info" },
}

interface VulnerabilitiesVerticalTableProps {
  items: Vulnerability[]
  selectedId: number | null
  selectedRows: Vulnerability[]
  onSelect: (vulnerability: Vulnerability) => void
  onSelectionChange: (rows: Vulnerability[]) => void
  onToggleReview?: (vulnerability: Vulnerability) => void
}

export function VulnerabilitiesVerticalTable({
  items,
  selectedId,
  selectedRows,
  onSelect,
  onSelectionChange,
  onToggleReview,
}: VulnerabilitiesVerticalTableProps) {
  const tColumns = useTranslations("columns")
  const tTooltips = useTranslations("tooltips")

  const selectedSet = React.useMemo(() => new Set(selectedRows.map((r) => r.id)), [selectedRows])

  // Keyboard navigation
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (!selectedId || items.length === 0) return
      const target = e.target as HTMLElement
      // Don't interfere with input elements
      if (target.tagName === "INPUT" || target.tagName === "TEXTAREA") return

      const currentIndex = items.findIndex((i) => i.id === selectedId)
      if (currentIndex === -1) return

      if (e.key === "ArrowDown") {
        e.preventDefault()
        const nextIndex = Math.min(items.length - 1, currentIndex + 1)
        onSelect(items[nextIndex])
      } else if (e.key === "ArrowUp") {
        e.preventDefault()
        const prevIndex = Math.max(0, currentIndex - 1)
        onSelect(items[prevIndex])
      }
    }

    window.addEventListener("keydown", handleKeyDown)
    return () => window.removeEventListener("keydown", handleKeyDown)
  }, [selectedId, items, onSelect])

  const allSelected = items.length > 0 && items.every((i) => selectedSet.has(i.id))
  const someSelected = items.some((i) => selectedSet.has(i.id)) && !allSelected

  const handleToggleAll = useCallback(
    (checked: boolean) => {
      onSelectionChange(checked ? [...items] : [])
    },
    [items, onSelectionChange]
  )

  const handleToggleRow = useCallback(
    (item: Vulnerability, checked: boolean) => {
      if (checked) {
        onSelectionChange([...selectedRows, item])
      } else {
        onSelectionChange(selectedRows.filter((r) => r.id !== item.id))
      }
    },
    [selectedRows, onSelectionChange]
  )

  return (
    <div className="flex-1 overflow-auto bg-background relative" role="grid" aria-label="Vulnerability List">
      <Table>
        <TableHeader className="sticky top-0 bg-muted/40/80 backdrop-blur-sm z-10 box-border shadow-none">
          <TableRow className="hover:bg-transparent border-b-border/60">
            <TableHead className="w-12 text-center h-9">
              <Checkbox
                checked={allSelected || (someSelected ? "indeterminate" : false)}
                onCheckedChange={(checked) => handleToggleAll(checked === true)}
                className="h-3.5 w-3.5 translate-y-[2px]"
                aria-label="Select all"
              />
            </TableHead>
            <TableHead className="w-12 text-center h-9 text-[10px] uppercase font-bold tracking-wider">
              {tColumns("common.status")}
            </TableHead>
            <TableHead className="w-[100px] h-9 text-[10px] uppercase font-bold tracking-wider">
              {tColumns("vulnerability.severity")}
            </TableHead>
            <TableHead className="min-w-[250px] max-w-[400px] h-9 text-[10px] uppercase font-bold tracking-wider">
              {tColumns("vulnerability.vulnType")}
            </TableHead>
            <TableHead className="w-[120px] h-9 text-[10px] uppercase font-bold tracking-wider">
              {tColumns("vulnerability.source")}
            </TableHead>
            <TableHead className="min-w-[200px] max-w-[350px] h-9 text-[10px] uppercase font-bold tracking-wider">
              {tColumns("common.url")}
            </TableHead>
            <TableHead className="text-right w-[160px] h-9 text-[10px] uppercase font-bold tracking-wider pr-6">
              {tColumns("common.createdAt")}
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {items.length === 0 ? (
            <TableRow>
              <TableCell colSpan={7} className="h-24 text-center text-muted-foreground">
                <div className="flex flex-col items-center justify-center gap-2 py-8">
                  <Info className="h-8 w-8 text-muted-foreground/30" />
                  <p className="text-sm">No vulnerabilities found matching your criteria.</p>
                </div>
              </TableCell>
            </TableRow>
          ) : (
            items.map((item) => {
              const isActive = selectedId === item.id
              const isChecked = selectedSet.has(item.id)
              const severity = SEVERITY_CONFIG[item.severity]

              return (
                <TableRow
                  key={item.id}
                  data-state={isActive ? "selected" : undefined}
                  className={cn(
                    "group cursor-pointer hover:bg-muted/40 transition-colors border-b-border/40",
                    isActive && "bg-primary/5"
                  )}
                  onClick={() => onSelect(item)}
                  role="row"
                  aria-selected={isActive}
                >
                  <TableCell className="text-center" onClick={(e) => e.stopPropagation()}>
                    <Checkbox
                      checked={isChecked}
                      onCheckedChange={(checked) => handleToggleRow(item, checked === true)}
                      className={cn(
                        "h-3.5 w-3.5 transition-opacity",
                        isChecked || isActive ? "opacity-100" : "opacity-0 group-hover:opacity-100"
                      )}
                      aria-label={`Select vulnerability ${item.id}`}
                    />
                  </TableCell>

                  <TableCell className="text-center" onClick={(e) => { e.stopPropagation(); onToggleReview?.(item) }}>
                    {item.isReviewed ? (
                      <div className="text-green-600 flex justify-center cursor-pointer" title={tTooltips("reviewed")} aria-label={tTooltips("reviewed")}>
                        <CheckCircle2 className="h-4 w-4" />
                      </div>
                    ) : (
                      <div className="text-blue-500 flex justify-center cursor-pointer" title={tTooltips("pending")} aria-label={tTooltips("pending")}>
                        <Circle className="h-4 w-4" />
                      </div>
                    )}
                  </TableCell>

                  <TableCell>
                    <Badge className={cn("h-5 px-1.5 text-[10px] font-bold uppercase rounded-sm border-0 w-[5ch] justify-center tracking-wider", severity.className)}>
                      <span className="sr-only">{severity.label}</span>
                      <span aria-hidden="true">{item.severity.slice(0, 3)}</span>
                    </Badge>
                  </TableCell>

                  <TableCell className="font-medium text-sm">
                    <div className="truncate w-full hover:text-primary transition-colors hover:underline underline-offset-4" title={item.vulnType}>
                      {item.vulnType}
                    </div>
                  </TableCell>

                  <TableCell>
                    <Badge variant="outline" className="h-5 px-2 text-[10px] font-medium text-muted-foreground justify-center shadow-none bg-background/50">
                      {item.source}
                    </Badge>
                  </TableCell>

                  <TableCell className="text-xs text-muted-foreground font-mono opacity-80">
                    <div className="truncate max-w-[350px]" title={item.url}>{item.url}</div>
                  </TableCell>

                  <TableCell className="text-right text-xs text-muted-foreground tabular-nums pr-6">
                    {new Date(item.createdAt).toLocaleDateString()}{" "}
                    <span className="opacity-50 ml-1">
                      {new Date(item.createdAt).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}
                    </span>
                  </TableCell>
                </TableRow>
              )
            })
          )}
        </TableBody>
      </Table>
    </div>
  )
}
