import React from "react"
import { IconSearch, CheckCircle2, Circle } from "@/components/icons"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { VulnFilter } from "./types"

interface VulnListHeaderProps {
  filter: VulnFilter
  onFilterChange: (filter: VulnFilter) => void
  search: string
  onSearchChange: (search: string) => void
  totalCount: number
  pendingCount: number
  reviewedCount: number
  selectedCount: number
  onMarkReviewed: () => void
  onMarkPending: () => void
}

export function VulnListHeader({
  filter,
  onFilterChange,
  search,
  onSearchChange,
  totalCount,
  pendingCount,
  reviewedCount,
  selectedCount,
  onMarkReviewed,
  onMarkPending,
}: VulnListHeaderProps) {
  return (
    <div className="h-14 border-b bg-card flex items-center justify-between px-4 shrink-0 gap-4">
      {/* Filters or Bulk Actions */}
      {selectedCount > 0 ? (
        <div className="flex items-center gap-3 animate-in fade-in duration-200">
          <Badge variant="secondary" className="h-7 px-2 font-mono text-xs">
            {selectedCount} selected
          </Badge>
          <div className="h-4 w-px bg-border/60 mx-1" />
          <Button
            size="sm"
            variant="outline"
            className="h-8 gap-2 text-green-600 hover:text-green-700 hover:bg-green-50 border-green-200/50"
            onClick={onMarkReviewed}
          >
            <CheckCircle2 className="h-3.5 w-3.5" />
            Mark Reviewed
          </Button>
          <Button
            size="sm"
            variant="outline"
            className="h-8 gap-2 text-blue-600 hover:text-blue-700 hover:bg-blue-50 border-blue-200/50"
            onClick={onMarkPending}
          >
            <Circle className="h-3.5 w-3.5" />
            Mark Pending
          </Button>
        </div>
      ) : (
        <Tabs value={filter} onValueChange={(v) => onFilterChange(v as VulnFilter)} className="w-auto">
          <TabsList className="h-9 p-1 bg-muted/50 border">
            <TabsTrigger value="all" className="text-xs px-3 h-7 data-[state=active]:shadow-sm">
              All <Badge variant="secondary" className="ml-2 bg-background/50 text-[10px] h-4 font-mono px-1 rounded-sm border-0">{totalCount}</Badge>
            </TabsTrigger>
            <TabsTrigger value="pending" className="text-xs px-3 h-7 data-[state=active]:shadow-sm">
              Pending <Badge variant="secondary" className="ml-2 bg-background/50 text-[10px] h-4 font-mono px-1 rounded-sm border-0">{pendingCount}</Badge>
            </TabsTrigger>
            <TabsTrigger value="reviewed" className="text-xs px-3 h-7 data-[state=active]:shadow-sm">
              Reviewed <Badge variant="secondary" className="ml-2 bg-background/50 text-[10px] h-4 font-mono px-1 rounded-sm border-0">{reviewedCount}</Badge>
            </TabsTrigger>
          </TabsList>
        </Tabs>
      )}

      {/* Search & Actions */}
      <div className="flex flex-1 items-center justify-end gap-4 min-w-0">
        <div className="relative w-full max-w-sm ml-auto">
          <IconSearch className="absolute left-2.5 top-2.5 size-4 text-muted-foreground" />
          <Input
            type="search"
            value={search}
            onChange={(e) => onSearchChange(e.target.value)}
            placeholder="Search vulnerabilities..."
            className="h-9 pl-9 bg-background/50"
          />
        </div>
      </div>
    </div>
  )
}
