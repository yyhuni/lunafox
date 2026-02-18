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
  sortBy?: string    // Sort field: id, name, created_at, updated_at (using snake_case)
  sortOrder?: "asc" | "desc"  // Sort direction: asc, desc
  search?: string    // Search keyword
}

