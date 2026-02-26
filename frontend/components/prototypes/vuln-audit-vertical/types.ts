
export type VulnFilter = "all" | "pending" | "reviewed"

export interface VulnListState {
  filter: VulnFilter
  search: string
  page: number
  selectedId: number | null
  selection: Set<number> // IDs of checked items
}

export type VulnSortField = "severity" | "createdAt" | "vulnType"
export type SortDirection = "asc" | "desc"

export interface VulnSortConfig {
  field: VulnSortField
  direction: SortDirection
}
