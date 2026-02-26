import React, { useRef, useEffect } from "react"
import { useVirtualizer } from "@tanstack/react-virtual"
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
import { CheckCircle2, Circle } from "@/components/icons"
import { cn } from "@/lib/utils"
import type { Vulnerability } from "@/types/vulnerability.types"
import { SEVERITY_CONFIG } from "./utils"

interface VulnListTableProps {
  items: Vulnerability[]
  selectedId: number | null
  selection: Set<number>
  onSelect: (id: number) => void
  onToggleSelection: (id: number, checked: boolean) => void
  onToggleAll: (checked: boolean) => void
}

export function VulnListTable({
  items,
  selectedId,
  selection,
  onSelect,
  onToggleSelection,
  onToggleAll,
}: VulnListTableProps) {
  const parentRef = useRef<HTMLDivElement>(null)

  const rowVirtualizer = useVirtualizer({
    count: items.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => 48, // Estimate row height
    overscan: 5,
  })

  // Keyboard navigation
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (!selectedId) return
      
      const currentIndex = items.findIndex((i) => i.id === selectedId)
      if (currentIndex === -1) return

      if (e.key === "ArrowDown") {
        e.preventDefault()
        const nextIndex = Math.min(items.length - 1, currentIndex + 1)
        onSelect(items[nextIndex].id)
        rowVirtualizer.scrollToIndex(nextIndex)
      } else if (e.key === "ArrowUp") {
        e.preventDefault()
        const prevIndex = Math.max(0, currentIndex - 1)
        onSelect(items[prevIndex].id)
        rowVirtualizer.scrollToIndex(prevIndex)
      }
    }

    window.addEventListener("keydown", handleKeyDown)
    return () => window.removeEventListener("keydown", handleKeyDown)
  }, [selectedId, items, onSelect, rowVirtualizer])

  const allSelected = items.length > 0 && items.every((i) => selection.has(i.id))
  const someSelected = items.some((i) => selection.has(i.id)) && !allSelected

  return (
    <div ref={parentRef} className="flex-1 overflow-auto bg-background relative" role="grid" aria-label="Vulnerability List">
      <Table>
        <TableHeader className="sticky top-0 bg-muted/40/80 backdrop-blur-sm z-10 box-border shadow-sm">
          <TableRow className="hover:bg-transparent border-b-border/60">
            <TableHead className="w-12 text-center h-9">
              <Checkbox
                checked={allSelected || (someSelected ? "indeterminate" : false)}
                onCheckedChange={(checked) => onToggleAll(checked === true)}
                className="h-3.5 w-3.5 translate-y-[2px]"
                aria-label="Select all vulnerabilities"
              />
            </TableHead>
            <TableHead className="w-12 text-center h-9 text-[10px] uppercase font-bold tracking-wider">St</TableHead>
            <TableHead className="w-[100px] h-9 text-[10px] uppercase font-bold tracking-wider">Severity</TableHead>
            <TableHead className="min-w-[250px] max-w-[400px] h-9 text-[10px] uppercase font-bold tracking-wider">Vulnerability</TableHead>
            <TableHead className="w-[120px] h-9 text-[10px] uppercase font-bold tracking-wider">Source</TableHead>
            <TableHead className="min-w-[200px] max-w-[350px] h-9 text-[10px] uppercase font-bold tracking-wider">Target URL</TableHead>
            <TableHead className="text-right w-[160px] h-9 text-[10px] uppercase font-bold tracking-wider pr-6">Found At</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody 
          className="relative w-full"
          style={{
            height: `${rowVirtualizer.getTotalSize()}px`,
          }}
        >
          {items.length === 0 ? (
             <TableRow>
              <TableCell colSpan={7} className="h-24 text-center text-muted-foreground">
                <div className="flex flex-col items-center justify-center gap-2 py-8">
                  <CheckCircle2 className="h-8 w-8 text-muted-foreground/30" />
                  <p>No vulnerabilities found matching your criteria.</p>
                </div>
              </TableCell>
            </TableRow>
          ) : (
             rowVirtualizer.getVirtualItems().map((virtualRow) => {
              const item = items[virtualRow.index]
              const isSelected = selectedId === item.id
              const isChecked = selection.has(item.id)
              const severity = SEVERITY_CONFIG[item.severity]

              return (
                <TableRow
                  key={item.id}
                  data-state={isSelected ? "selected" : undefined}
                  className={cn(
                    "group cursor-pointer hover:bg-muted/40 data-[state=selected]:bg-primary/5 transition-colors border-b-border/40 absolute top-0 left-0 w-full flex items-center",
                    isSelected && "bg-primary/5"
                  )}
                  style={{
                    height: `${virtualRow.size}px`,
                    transform: `translateY(${virtualRow.start}px)`,
                  }}
                  onClick={() => onSelect(item.id)}
                  role="row"
                  aria-selected={isSelected}
                >
                  <TableCell className="text-center w-12 shrink-0 py-0" onClick={(e) => e.stopPropagation()}>
                    <Checkbox
                      checked={isChecked}
                      onCheckedChange={(checked) => onToggleSelection(item.id, checked === true)}
                      className={cn(
                        "h-3.5 w-3.5 transition-opacity", 
                        isChecked || isSelected ? "opacity-100" : "opacity-0 group-hover:opacity-100"
                      )}
                      aria-label={`Select vulnerability ${item.id}`}
                    />
                  </TableCell>

                  <TableCell className="text-center w-12 shrink-0 py-0">
                    {item.isReviewed ? (
                      <div className="text-green-600 flex justify-center" title="Reviewed" aria-label="Reviewed">
                        <CheckCircle2 className="h-4 w-4" />
                      </div>
                    ) : (
                      <div className="text-blue-500 flex justify-center" title="Pending" aria-label="Pending">
                        <Circle className="h-4 w-4" />
                      </div>
                    )}
                  </TableCell>

                  <TableCell className="w-[100px] shrink-0 py-0">
                    <Badge className={cn("h-5 px-1.5 text-[10px] font-bold uppercase rounded-sm border-0 w-[5ch] justify-center tracking-wider", severity.className)}>
                      <span className="sr-only">{severity.label}</span>
                      <span aria-hidden="true">{item.severity.slice(0, 3)}</span>
                    </Badge>
                  </TableCell>

                  <TableCell className="font-medium text-sm flex-1 min-w-[250px] max-w-[400px] py-0">
                    <div className="truncate w-full hover:text-primary transition-colors hover:underline underline-offset-4" title={item.vulnType}>
                      {item.vulnType}
                    </div>
                  </TableCell>

                  <TableCell className="w-[120px] shrink-0 py-0">
                    <Badge variant="outline" className="h-5 px-2 text-[10px] font-medium text-muted-foreground justify-center shadow-none bg-background/50">
                      {item.source}
                    </Badge>
                  </TableCell>

                  <TableCell className="text-xs text-muted-foreground font-mono opacity-80 min-w-[200px] max-w-[350px] flex-1 py-0">
                    <div className="truncate max-w-[350px]" title={item.url}>{item.url}</div>
                  </TableCell>

                  <TableCell className="text-right text-xs text-muted-foreground tabular-nums w-[160px] shrink-0 pr-6 py-0">
                    {new Date(item.createdAt).toLocaleDateString()} <span className="opacity-50 ml-1">{new Date(item.createdAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</span>
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
