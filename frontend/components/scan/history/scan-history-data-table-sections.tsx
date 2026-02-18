"use client"

import { Filter } from "@/components/icons"

import { SimpleSearchToolbar } from "@/components/ui/data-table/simple-search-toolbar"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

import type { ScanStatus } from "@/types/scan.types"
import type { ScanHistoryDataTableState } from "./scan-history-data-table-state"

interface ScanHistoryToolbarProps {
  state: ScanHistoryDataTableState
  loading: boolean
  placeholder?: string
  status: ScanStatus | "all"
  onStatusChange?: (status: ScanStatus | "all") => void
}

export function ScanHistoryToolbar({
  state,
  loading,
  placeholder,
  status,
  onStatusChange,
}: ScanHistoryToolbarProps) {
  return (
    <SimpleSearchToolbar
      value={state.localSearchValue}
      onChange={state.setLocalSearchValue}
      onSubmit={state.handleSearchSubmit}
      loading={loading}
      placeholder={placeholder || state.tScan("searchPlaceholder")}
      after={onStatusChange ? (
        <Select
          value={status}
          onValueChange={(value) => onStatusChange(value as ScanStatus | "all")}
        >
          <SelectTrigger size="sm" className="w-auto">
            <Filter className="h-4 w-4" />
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {state.statusOptions.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      ) : null}
    />
  )
}
