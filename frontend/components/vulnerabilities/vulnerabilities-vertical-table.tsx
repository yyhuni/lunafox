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

const SEVERITY_CONFIG: Record<VulnerabilitySeverity, { label: string; color: string; bgColor: string }> = {
  critical: { label: "Critical", color: SEVERITY_STYLES.critical.color, bgColor: SEVERITY_STYLES.critical.bgColor },
  high: { label: "High", color: SEVERITY_STYLES.high.color, bgColor: SEVERITY_STYLES.high.bgColor },
  medium: { label: "Medium", color: SEVERITY_STYLES.medium.color, bgColor: SEVERITY_STYLES.medium.bgColor },
  low: { label: "Low", color: SEVERITY_STYLES.low.color, bgColor: SEVERITY_STYLES.low.bgColor },
  info: { label: "Info", color: SEVERITY_STYLES.info.color, bgColor: SEVERITY_STYLES.info.bgColor },
}

function hexToRgba(hex: string, alpha: number): string {
  const normalized = hex.replace("#", "")
  const full = normalized.length === 3
    ? normalized.split("").map((c) => `${c}${c}`).join("")
    : normalized
  const int = Number.parseInt(full, 16)
  const r = (int >> 16) & 255
  const g = (int >> 8) & 255
  const b = int & 255
  return `rgba(${r}, ${g}, ${b}, ${alpha})`
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
  const tVuln = useTranslations("vulnerabilities")
  const tColumns = useTranslations("columns")
  const tActions = useTranslations("common.actions")

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
        if (selectedRows.some((r) => r.id === item.id)) {
          return
        }
        onSelectionChange([...selectedRows, item])
      } else {
        onSelectionChange(selectedRows.filter((r) => r.id !== item.id))
      }
    },
    [selectedRows, onSelectionChange]
  )

  return (
    <div className="flex-1 overflow-auto bg-background relative" role="grid" aria-label={tVuln("listAriaLabel")}>
      <Table>
        <TableHeader className="sticky top-0 bg-muted/80 backdrop-blur-sm z-10 box-border shadow-none">
          <TableRow className="hover:bg-transparent border-b-border/60">
            <TableHead className="w-10 md:w-12 text-center h-[var(--vuln-table-head-h)] leading-none">
              <Checkbox
                checked={allSelected || (someSelected ? "indeterminate" : false)}
                onCheckedChange={(checked) => handleToggleAll(checked === true)}
                className="h-4 w-4 md:h-3.5 md:w-3.5"
                aria-label={tActions("selectAll")}
              />
            </TableHead>
            <TableHead className="w-10 md:w-12 text-center h-[var(--vuln-table-head-h)] text-[11px] md:text-xs uppercase font-bold tracking-wider">
              {tColumns("common.status")}
            </TableHead>
            <TableHead className="w-[72px] md:w-[100px] h-[var(--vuln-table-head-h)] text-[11px] md:text-xs uppercase font-bold tracking-wider">
              {tColumns("vulnerability.severity")}
            </TableHead>
            <TableHead className="min-w-[160px] md:min-w-[250px] max-w-[400px] h-[var(--vuln-table-head-h)] text-[11px] md:text-xs uppercase font-bold tracking-wider">
              {tColumns("vulnerability.vulnType")}
            </TableHead>
            <TableHead className="hidden md:table-cell w-[120px] h-[var(--vuln-table-head-h)] text-[11px] md:text-xs uppercase font-bold tracking-wider">
              {tColumns("vulnerability.source")}
            </TableHead>
            <TableHead className="hidden lg:table-cell min-w-[200px] max-w-[350px] h-[var(--vuln-table-head-h)] text-[11px] md:text-xs uppercase font-bold tracking-wider">
              {tColumns("common.url")}
            </TableHead>
            <TableHead className="hidden lg:table-cell text-right w-[160px] h-[var(--vuln-table-head-h)] text-[11px] md:text-xs uppercase font-bold tracking-wider pr-6">
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
                  <p className="text-sm">{tVuln("emptyFiltered")}</p>
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
                    "group h-[var(--vuln-row-h)] cursor-pointer hover:bg-muted/40 transition-colors border-b-border/40",
                    isActive && "bg-primary/5"
                  )}
                  onClick={() => {
                    onSelect(item)
                    handleToggleRow(item, !isChecked)
                  }}
                  role="row"
                  aria-selected={isActive}
                >
                  <TableCell
                    className="text-center leading-none"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <Checkbox
                      checked={isChecked}
                      onCheckedChange={(checked) => handleToggleRow(item, checked === true)}
                      className={cn(
                        "h-4 w-4 md:h-3.5 md:w-3.5 transition-opacity",
                        isChecked || isActive ? "opacity-100" : "opacity-100 md:opacity-0 md:group-hover:opacity-100"
                      )}
                      aria-label={`${tActions("selectRow")} ${item.id}`}
                    />
                  </TableCell>

                  <TableCell className="text-center">
                    <button
                      type="button"
                      className="h-11 w-11 md:h-7 md:w-7 inline-flex items-center justify-center rounded-md hover:bg-muted/60 transition-colors"
                      title={item.isReviewed ? tVuln("markAsPending") : tVuln("markAsReviewed")}
                      aria-label={item.isReviewed ? tVuln("markAsPending") : tVuln("markAsReviewed")}
                      onClick={(e) => {
                        e.stopPropagation()
                        onToggleReview?.(item)
                      }}
                    >
                      {item.isReviewed ? (
                        <CheckCircle2 className="h-4 w-4 text-green-600" />
                      ) : (
                        <Circle className="h-4 w-4 text-blue-500" />
                      )}
                    </button>
                  </TableCell>

                  <TableCell>
                    <Badge
                      className={cn(
                        "h-5 px-1.5 text-[10px] font-bold uppercase rounded-sm border w-[4ch] md:w-[5ch] justify-center tracking-wider !bg-[var(--sev-bg)] !text-[var(--sev-fg)] !border-[var(--sev-border)]"
                      )}
                      style={{
                        ["--sev-fg" as string]: severity.color,
                        ["--sev-bg" as string]: severity.bgColor,
                        ["--sev-border" as string]: hexToRgba(severity.color, 0.24),
                      }}
                    >
                      <span className="sr-only">{severity.label}</span>
                      <span aria-hidden="true">{item.severity.slice(0, 3)}</span>
                    </Badge>
                  </TableCell>

                  <TableCell className="font-medium text-sm">
                    <div className="truncate w-full hover:text-primary transition-colors hover:underline underline-offset-4" title={item.vulnType}>
                      {item.vulnType}
                    </div>
                  </TableCell>

                  <TableCell className="hidden md:table-cell">
                    <Badge variant="outline" className="h-5 px-2 text-[10px] font-medium text-muted-foreground justify-center shadow-none bg-background/50">
                      {item.source}
                    </Badge>
                  </TableCell>

                  <TableCell className="hidden lg:table-cell text-xs text-muted-foreground font-mono opacity-80">
                    <div className="truncate max-w-[350px]" title={item.url}>{item.url}</div>
                  </TableCell>

                  <TableCell className="hidden lg:table-cell text-right text-xs text-muted-foreground tabular-nums pr-6">
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
