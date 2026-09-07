"use client"

import React, { useCallback, useEffect } from "react"
import { useTranslations } from "next-intl"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  TABLE_DENSE_CELL_RHYTHM_CLASS,
  TABLE_DENSE_ROW_CLASS,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { CheckCircle2, Circle, Info } from "@/components/icons"
import { getStatusToneTextClass } from "@/lib/status-config"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import { SEVERITY_VARIANTS, VULNERABILITY_SEVERITY_BADGE_CLASS } from "@/lib/severity-config"
import type { Vulnerability, VulnerabilitySeverity } from "@/types/vulnerability.types"

const VULNERABILITIES_DENSE_ROW_CLASS = cn("group cursor-pointer", TABLE_DENSE_ROW_CLASS)
const VULNERABILITIES_DENSE_CELL_CLASS = TABLE_DENSE_CELL_RHYTHM_CLASS

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
  const tSeverity = useTranslations("vulnerabilities.severity")

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
    <div className="bg-card flex-1 overflow-auto relative" role="grid" aria-label={tVuln("listAriaLabel")}>
      <Table data-row-rhythm="dense">
        <TableHeader className="sticky top-0 z-10">
          <TableRow>
            <TableHead className="leading-none md:w-12 text-center w-10">
              <Checkbox
                checked={allSelected}
                indeterminate={!allSelected && someSelected}
                onCheckedChange={(checked) => handleToggleAll(checked === true)}
                className="h-4 md:h-3.5 md:w-3.5 w-4"
                aria-label={tActions("selectAll")}
              />
            </TableHead>
            <TableHead className="md:w-12 text-center w-10">
              {tColumns("common.status")}
            </TableHead>
            <TableHead className="md:w-[100px] w-[72px]">
              {tColumns("vulnerability.severity")}
            </TableHead>
            <TableHead className="max-w-[400px] md:min-w-[250px] min-w-[160px]">
              {tColumns("vulnerability.vulnType")}
            </TableHead>
            <TableHead className="hidden md:table-cell w-[120px]">
              {tColumns("vulnerability.source")}
            </TableHead>
            <TableHead className="hidden lg:table-cell max-w-[350px] min-w-[200px]">
              {tColumns("common.url")}
            </TableHead>
            <TableHead className="hidden lg:table-cell pr-6 text-right w-[160px]">
              {tColumns("common.createdAt")}
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {items.length === 0 ? (
            <TableRow>
              <TableCell colSpan={7} className={cn("h-24 text-center", textRole.bodySubtle)}>
                <div className="flex flex-col gap-2 items-center justify-center py-8">
                  <Info className="h-8 text-muted-foreground/30 w-8" />
                  <p className="text-sm">{tVuln("emptyFiltered")}</p>
                </div>
              </TableCell>
            </TableRow>
          ) : (
            items.map((item) => {
              const isActive = selectedId === item.id
              const isChecked = selectedSet.has(item.id)
              const severityLabel = tSeverity(item.severity as VulnerabilitySeverity)

              return (
                <TableRow
                  key={item.id}
                  data-state={isActive ? "selected" : undefined}
                  className={VULNERABILITIES_DENSE_ROW_CLASS}
                  onClick={() => {
                    onSelect(item)
                    handleToggleRow(item, !isChecked)
                  }}
                  role="row"
                  aria-selected={isActive}
                >
                  <TableCell
                    className={cn(VULNERABILITIES_DENSE_CELL_CLASS, "leading-none text-center")}
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

                  <TableCell className={cn(VULNERABILITIES_DENSE_CELL_CLASS, "text-center")}>
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon-sm"
                      className="hover:text-foreground text-muted-foreground"
                      title={item.isReviewed ? tVuln("markAsPending") : tVuln("markAsReviewed")}
                      aria-label={item.isReviewed ? tVuln("markAsPending") : tVuln("markAsReviewed")}
                      onClick={(e) => {
                        e.stopPropagation()
                        onToggleReview?.(item)
                      }}
                    >
                      {item.isReviewed ? (
                        <CheckCircle2 className={cn("h-4 w-4", getStatusToneTextClass("success"))} />
                      ) : (
                        <Circle className={cn("h-4 w-4", getStatusToneTextClass("muted"))} />
                      )}
                    </Button>
                  </TableCell>

                  <TableCell className={VULNERABILITIES_DENSE_CELL_CLASS}>
                    <Badge
                      variant={SEVERITY_VARIANTS[item.severity]}
                      className={VULNERABILITY_SEVERITY_BADGE_CLASS}
                    >
                      {severityLabel}
                    </Badge>
                  </TableCell>

                  <TableCell className={cn(VULNERABILITIES_DENSE_CELL_CLASS, textRole.tableCellPrimary)}>
                    <div className="hover:text-primary hover:underline transition-colors truncate underline-offset-4 w-full" title={item.vulnType}>
                      {item.vulnType}
                    </div>
                  </TableCell>

                  <TableCell className={cn(VULNERABILITIES_DENSE_CELL_CLASS, "hidden md:table-cell")}>
                    <Badge variant="outline" className={cn("h-5 justify-center px-2 shadow-none", textRole.badgeSubtle)}>
                      {item.source}
                    </Badge>
                  </TableCell>

                  <TableCell className={cn(VULNERABILITIES_DENSE_CELL_CLASS, "font-mono hidden lg:table-cell", textRole.tableCellSecondary)}>
                    <div className="max-w-[350px] truncate" title={item.url}>{item.url}</div>
                  </TableCell>

                  <TableCell className={cn(VULNERABILITIES_DENSE_CELL_CLASS, "hidden lg:table-cell pr-6 tabular-nums text-right", textRole.tableCellSecondary)}>
                    {new Date(item.createdAt).toLocaleDateString()}{" "}
                    <span className="ml-1 opacity-50">
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
