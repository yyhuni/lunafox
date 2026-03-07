import { ColumnDef } from "@tanstack/react-table"
import { PaginationParams, PaginationInfo } from "./common.types"
import type { BatchCreateResponse } from "./api-response.types"

// Subdomain related type definitions (renamed from domain.types.ts)

// Basic subdomain type
export interface Subdomain {
  id: number
  name: string
  createdAt: string
}

// Get subdomains list request parameters
export interface GetSubdomainsParams extends PaginationParams {
  organizationId: number
}

// Get subdomains list response (field 'domains' kept consistent with backend)
export interface GetSubdomainsResponse {
  domains: Subdomain[]
  total: number
  page: number
  pageSize: number
  totalPages: number
}

// Get all subdomains request parameters
// Backend sorts by update time descending, custom sorting not supported
export interface GetAllSubdomainsParams {
  page?: number
  pageSize?: number
  search?: string
}

// Get all subdomains response (field 'domains' kept consistent with backend)
export interface GetAllSubdomainsResponse {
  domains: Subdomain[]
  total: number
  page: number
  pageSize: number
  totalPages: number
}

// Get single subdomain detail response (backend returns object directly)
export type GetSubdomainByIDResponse = Subdomain

// Subdomain data table component props type definition
export interface SubdomainDataTableProps {
  data: Subdomain[]
  columns: ColumnDef<Subdomain>[]
  onAddNew?: () => void
  onBulkDelete?: () => void
  onSelectionChange?: (selectedRows: Subdomain[]) => void
  searchPlaceholder?: string
  searchColumn?: string
  pagination?: {
    pageIndex: number
    pageSize: number
  }
  setPagination?: (pagination: { pageIndex: number; pageSize: number }) => void
  paginationInfo?: PaginationInfo
  onPaginationChange?: (pagination: { pageIndex: number; pageSize: number }) => void
}

// Subdomain batch create response (reuse common type)
export type BatchCreateSubdomainsResponse = BatchCreateResponse
