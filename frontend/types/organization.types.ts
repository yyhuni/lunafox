import { ColumnDef, type SortingState } from "@tanstack/react-table"
import { PaginationInfo } from "./common.types"
import type { CursorPaginationNavigation } from "./data-table.types"
import type { CursorPaginationSummary } from "./data-table.types"

export interface OrganizationStats {
  totalDomains?: number
  totalEndpoints?: number
  totalTargets?: number
}

export interface Organization {
  id: number
  name: string
  resourceName?: string
  description: string
  createdAt: string
  updatedAt: string
  targets?: Array<{
    id: number
    name: string
  }>
  stats?: OrganizationStats
  targetCount?: number
  domainCount?: number
  endpointCount?: number
}

export interface OrganizationsResponse<T = Organization> {
  results: T[]
  total: number
  totalSize?: number
  nextPageToken?: string
  page: number
  pageSize: number
  totalPages: number
}

export interface CreateOrganizationRequest {
  name: string
  description: string
}

export interface UpdateOrganizationRequest {
  name: string
  description: string
}

export interface OrganizationDataTableProps {
  data: Organization[]
  columns: ColumnDef<Organization>[]
  onAddNew?: () => void
  onBulkDelete?: () => void
  onBulkInitiateScan?: () => void
  onViewDetail?: (organization: Organization) => void
  onDetailIntent?: () => void
  onAddIntent?: () => void
  onSelectionChange?: (selectedRows: Organization[]) => void
  selectedRows?: Organization[]
  searchPlaceholder?: string
  searchColumn?: string
  searchValue?: string
  onSearch?: (value: string) => void
  isSearching?: boolean
  pagination?: {
    pageIndex: number
    pageSize: number
  }
  paginationInfo?: PaginationInfo
  cursorPaginationSummary?: CursorPaginationSummary
  paginationNavigation?: CursorPaginationNavigation
  onPaginationChange?: (pagination: { pageIndex: number; pageSize: number }) => void
  sorting?: SortingState
  onSortingChange?: (sorting: SortingState) => void
  loading?: boolean
  loadingRowCount?: number
  stableSurfaceRowCount?: number
}
