// Common API response types
//
// New response format:
// - Successful response: Return data T directly (without packaging)
// - Pagination response: { results: T[], total, page, pageSize, totalPages }
// - Error response: { error: { code, message?, details? } }
//
// NOTE: The old { code, state, message, data } format is obsolete

/**
 * Error response type
 */
export interface ApiErrorResponse {
  error: {
    code: string;
    message?: string;
    details?: unknown[];
  };
}

/**
 * Legacy API response types (deprecated, retained only for backward compatibility)
 * @deprecated Please use data type T directly, the backend no longer uses this packaging format
 */
export interface LegacyApiResponse<T = unknown> {
  code: string;          // HTTP status code, e.g. "200", "400", "500"
  state: string;         // Business state, e.g. "success", "error"
  message: string;       // Response message
  data?: T;              // Response data
}

// Common batch create response data (corresponds to backend BaseBatchCreateResponseData)
// Applicable to: domains, endpoints and other batch create operations
export interface BatchCreateResponse {
  message: string          // Detailed description, e.g. "Processed 5 domains, created 3 new, 2 existed, 1 skipped"
  requestedCount: number   // Total requested count
  createdCount: number     // Newly created count
  existedCount: number     // Already existed count
  skippedCount?: number    // Skipped count (optional)
  skippedDomains?: Array<{  // Skipped domains list (optional)
    name: string
    reason: string
  }>
}


// Paginated response type
export interface PaginatedResponse<T> {
  results: T[]
  total: number
  page: number
  pageSize: number
  totalPages?: number
}
