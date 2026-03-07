import { ColumnDef } from "@tanstack/react-table"
import { PaginationInfo } from "./common.types"

export interface OrganizationStats {
  totalDomains?: number
  totalEndpoints?: number
  totalTargets?: number
}

export interface Organization {
  id: number
  name: string
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
  onSelectionChange?: (selectedRows: Organization[]) => void
  searchPlaceholder?: string
  searchColumn?: string
  searchValue?: string
  onSearch?: (value: string) => void
  isSearching?: boolean
  pagination?: {
    pageIndex: number
    pageSize: number
  }
  setPagination?: (pagination: { pageIndex: number; pageSize: number }) => void
  paginationInfo?: PaginationInfo
  onPaginationChange?: (pagination: { pageIndex: number; pageSize: number }) => void
}
