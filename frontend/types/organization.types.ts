import { ColumnDef } from "@tanstack/react-table"
import { PaginationInfo } from "./common.types"

// Organization statistics data
export interface OrganizationStats {
  totalDomains?: number    // Total domains count
  totalEndpoints?: number  // Total endpoints count
  totalTargets?: number    // Total targets count
}

// Organization related type definitions (matches backend Organization model)
export interface Organization {
  id: number
  name: string
  description: string
  createdAt: string      // Backend created_at auto-converted to camelCase by Django
  updatedAt: string      // Backend updated_at
  // Associated data (added via serializer)
  targets?: Array<{
    id: number
    name: string
  }>
  // Statistics data (optional, obtained via aggregate query)
  stats?: OrganizationStats
  targetCount?: number   // Target count (for list display)
  domainCount?: number   // Domain count (for list display)
  endpointCount?: number // Endpoint count (for list display)
}

// Organization list response type (matches backend actual response format)
export interface OrganizationsResponse<T = Organization> {
  results: T[]          // Organization data list
  total: number         // Total record count (backend actual field)
  page: number          // Current page number
  pageSize: number      // Page size
  totalPages: number    // Total pages
  // Compatibility fields
  count?: number        // DRF standard field (backward compatible)
  next?: string | null   // Next page link (DRF standard field)
  previous?: string | null // Previous page link (DRF standard field)
  organizations?: T[]
  pagination?: {
    total: number
    page: number
    pageSize: number
    totalPages: number
  }
}


// Create organization request type
export interface CreateOrganizationRequest {
  name: string
  description: string
}

// Update organization request type
export interface UpdateOrganizationRequest {
  name: string
  description: string
}

// Organization data table component props type definition
export interface OrganizationDataTableProps {
  data: Organization[]                           // Organization data array
  columns: ColumnDef<Organization>[]             // Column definitions array
  onAddNew?: () => void                          // Add new organization callback
  onBulkDelete?: () => void                      // Bulk delete callback
  onSelectionChange?: (selectedRows: Organization[]) => void  // Selected rows change callback
  searchPlaceholder?: string                     // Search input placeholder
  searchColumn?: string                          // Column name to search
  searchValue?: string                           // Controlled: search input current value (server-side search)
  onSearch?: (value: string) => void             // Controlled: search input change callback (server-side search)
  isSearching?: boolean                          // Searching state (show loading animation)
  // Pagination related props
  pagination?: {
    pageIndex: number
    pageSize: number
  }
  setPagination?: (pagination: { pageIndex: number; pageSize: number }) => void
  paginationInfo?: PaginationInfo
  onPaginationChange?: (pagination: { pageIndex: number; pageSize: number }) => void
}
