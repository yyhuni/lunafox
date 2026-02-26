"use client"

import React, { useState, useMemo, useCallback } from "react"
import { useUrlState } from "@/hooks/use-url-state"
import { useVerticalResize } from "@/hooks/use-vertical-resize"
import { MOCK_VULNS } from "@/lib/mock-vulnerabilities"
import { VulnerabilityAuditDetail } from "@/components/prototypes/vulnerability-audit-detail"
import { toast } from "sonner"

import { VulnListHeader } from "./vuln-list-header"
import { VulnListTable } from "./vuln-list-table"
import { ResizeHandle } from "./resize-handle"
import { filterVulnerabilities } from "./utils"
import type { VulnFilter } from "./types"

export function VulnAuditVertical() {
  // 1. State Management
  const [selectedIdStr, setSelectedId] = useUrlState("id", String(MOCK_VULNS[0].id))
  const [filter, setFilter] = useUrlState<VulnFilter>("filter", "all")
  const [search, setSearch] = useState("") // Local state for search to avoid URL thrashing on every keystroke
  
  // Selection state for bulk actions
  const [selection, setSelection] = useState<Set<number>>(new Set())

  const selectedId = parseInt(selectedIdStr, 10) || MOCK_VULNS[0].id

  // 2. Vertical Resizing
  const { height, startResizing, containerRef, isResizing } = useVerticalResize(45)

  // 3. Data Processing
  const filteredItems = useMemo(() => {
    return filterVulnerabilities(MOCK_VULNS, filter, search)
  }, [filter, search])

  const selectedItem = useMemo(() => {
    return MOCK_VULNS.find(i => i.id === selectedId) || MOCK_VULNS[0]
  }, [selectedId])

  // 4. Handlers
  const handleSelectionToggle = useCallback((id: number, checked: boolean) => {
    setSelection(prev => {
      const next = new Set(prev)
      if (checked) next.add(id)
      else next.delete(id)
      return next
    })
  }, [])

  const handleToggleAll = useCallback((checked: boolean) => {
    if (checked) {
      setSelection(new Set(filteredItems.map(i => i.id)))
    } else {
      setSelection(new Set())
    }
  }, [filteredItems])

  const handleMarkReviewed = useCallback(() => {
    toast.success(`Marked ${selection.size} items as reviewed`)
    setSelection(new Set())
    // In a real app, you'd trigger a mutation here
  }, [selection])

  const handleMarkPending = useCallback(() => {
    toast.success(`Marked ${selection.size} items as pending`)
    setSelection(new Set())
    // In a real app, you'd trigger a mutation here
  }, [selection])

  const handleFilterChange = useCallback((newFilter: VulnFilter) => {
    setFilter(newFilter)
    setSelection(new Set()) // Clear selection on filter change
  }, [setFilter])

  // Counts for tabs
  const counts = useMemo(() => ({
    total: MOCK_VULNS.length,
    pending: MOCK_VULNS.filter(i => !i.isReviewed).length,
    reviewed: MOCK_VULNS.filter(i => i.isReviewed).length
  }), [])

  return (
    <div ref={containerRef} className="flex flex-col h-full min-h-0 bg-background border rounded-lg overflow-hidden select-none">
      {/* Top: List View (Table) */}
      <div
        className="flex flex-col border-b resize-y overflow-hidden relative min-h-[200px]"
        style={{ height: `${height}%` }}
      >
        <VulnListHeader
          filter={filter}
          onFilterChange={handleFilterChange}
          search={search}
          onSearchChange={setSearch}
          totalCount={counts.total}
          pendingCount={counts.pending}
          reviewedCount={counts.reviewed}
          selectedCount={selection.size}
          onMarkReviewed={handleMarkReviewed}
          onMarkPending={handleMarkPending}
        />

        <VulnListTable
          items={filteredItems}
          selectedId={selectedId}
          selection={selection}
          onSelect={(id) => setSelectedId(String(id))}
          onToggleSelection={handleSelectionToggle}
          onToggleAll={handleToggleAll}
        />

        <ResizeHandle isResizing={isResizing} onMouseDown={startResizing} />
      </div>

      {/* Bottom: Detail View */}
      <div className="flex-1 flex flex-col min-h-0 bg-background relative overflow-hidden">
        <VulnerabilityAuditDetail
          vulnerability={selectedItem}
          className="h-full"
        />
      </div>
    </div>
  )
}
