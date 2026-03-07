// Common type definitions

// Pagination info interface
export interface PaginationInfo {
  total: number
  page: number
  pageSize: number
  totalPages: number
}

// Pagination and sorting parameters interface
export interface PaginationParams {
  page?: number
  pageSize?: number
  sortBy?: string
  sortOrder?: "asc" | "desc"
  search?: string
}
