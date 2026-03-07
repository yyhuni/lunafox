/**
 * API response parser
 * 
 * Unified processing of backend API responses, supporting new standardized response formats:
 * - Successful response: Return data T directly (without packaging)
 * - Pagination response: { results: T[], total, page, pageSize, totalPages }
 * - Error response: { error: { code: string, message?: string, details?: unknown[] } }
 * 
 * Note: The backend success_response() returns data directly and no longer uses { data: T } packaging.
 * The service layer has unpacked the axios response through res.data, so what the hook gets is the final data.
 */

/**
 * Standardized error response types
 */
export interface ApiErrorResponse {
  error: {
    code: string;
    message?: string;
    details?: unknown[];
  };
}

/**
 * Unified API response types
 * Success: Return data T directly
 * Error: { error: { code, message?, details? } }
 */
export type ApiResponse<T = unknown> = T | ApiErrorResponse;

/**
 * Legacy API response types (backward compatibility)
 */
export interface LegacyApiResponse<T = unknown> {
  code: string;
  state: string;
  message: string;
  data?: T;
}

/**
 * Determine whether the response is an error response
 * 
 * @param response - API response object
 * @returns returns true if it is an error response
 * 
 * @example
 * const response = await api.get('/scans');
 * if (isErrorResponse(response)) {
 *   // handle error
 * }
 */
export function isErrorResponse(response: unknown): response is ApiErrorResponse {
  return (
    typeof response === 'object' &&
    response !== null &&
    'error' in response &&
    typeof (response as ApiErrorResponse).error === 'object' &&
    (response as ApiErrorResponse).error !== null &&
    typeof (response as ApiErrorResponse).error.code === 'string'
  );
}

/**
 * Determine whether the response is a successful response (non-error response)
 * 
 * @param response - API response object
 * @returns Returns true if the response is successful
 */
export function isSuccessResponse(response: unknown): boolean {
  // non-object or null is not a successful response
  if (typeof response !== 'object' || response === null) {
    return false;
  }
  
  // If there is an error field, it is not a successful response
  if ('error' in response) {
    return false;
  }
  
  return true;
}

/**
 * Determine whether the response is in an old format
 * 
 * @param response - API response object
 * @returns true if it is an old format
 */
export function isLegacyResponse<T = unknown>(
  response: unknown
): response is LegacyApiResponse<T> {
  return (
    typeof response === 'object' &&
    response !== null &&
    'state' in response &&
    'code' in response &&
    typeof (response as LegacyApiResponse).state === 'string'
  );
}

/**
 * Determine whether the old response is an error
 * 
 * @param response - legacy API response object
 * @returns returns true if it is an error response
 */
export function isLegacyErrorResponse<T = unknown>(
  response: LegacyApiResponse<T>
): boolean {
  return response.state !== 'success';
}

/**
 * Parse data from response
 * 
 * Note: Under the new format, the service layer returns the final data (directly returned by the backend, without packaging)
 * This function is mainly used for:
 * - Check if it is an error response
 * - Compatible with old format { state: 'success', data: T }
 * 
 * @param response - API response object (usually already final data)
 * @returns The parsed data, or null if it is an error response
 * 
 * @example
 * const response = await quickScan(data);
 * const data = parseResponse<QuickScanResponse>(response);
 * if (data) {
 *   // use data.count
 * }
 */
export function parseResponse<T>(response: unknown): T | null {
  // Handle null/undefined
  if (response === null || response === undefined) {
    return null;
  }
  
  // Handle error response { error: { code, message } }
  if (isErrorResponse(response)) {
    return null;
  }
  
  // Handling old format responses { state: 'success', data: T }
  if (isLegacyResponse<T>(response)) {
    if (isLegacyErrorResponse(response)) {
      return null;
    }
    return response.data ?? null;
  }
  
  // New format: response itself is data (no packaging)
  // The service layer has already returned res.data, so response is returned directly here.
  return response as T;
}

/**
 * Get error code from response
 * 
 * Supports both old and new response formats:
 * - New formalism: { error: { code: 'ERROR_CODE' } }
 * - Old format: { state: 'error', code: '400' }
 * 
 * @param response - API response object
 * @returns error code string, or null if it is not an error response
 * 
 * @example
 * const response = await api.delete('/scans/123');
 * const errorCode = getErrorCode(response);
 * if (errorCode) {
 *   toast.error(t(`errors.${errorCode}`));
 * }
 */
export function getErrorCode(response: unknown): string | null {
  // Handling new format error responses
  if (isErrorResponse(response)) {
    return response.error.code;
  }
  
  // Handling old format error responses
  if (isLegacyResponse(response) && isLegacyErrorResponse(response)) {
    // The old format code is an HTTP status code, not an error code
    // Return common error code
    return 'SERVER_ERROR';
  }
  
  return null;
}

/**
 * Extract response payload from an error object.
 */
export function getErrorResponseData(error: unknown): unknown {
  if (!error || typeof error !== 'object') return undefined;
  if (!('response' in error)) return undefined;
  const response = (error as { response?: { data?: unknown } }).response;
  return response?.data;
}

/**
 * Get the error message from the response (for debugging)
 * 
 * @param response - API response object
 * @returns error message string, or null if not an error response
 */
export function getErrorMessage(response: unknown): string | null {
  // Handling new format error responses
  if (isErrorResponse(response)) {
    return response.error.message ?? null;
  }
  
  // Handling old format error responses
  if (isLegacyResponse(response) && isLegacyErrorResponse(response)) {
    return response.message;
  }
  
  return null;
}

/**
 * Paginated response metadata type
 */
export interface PaginationMeta {
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

/**
 * Get metadata from paginated response
 * 
 * @param response - API paging response object { results, total, page, pageSize, totalPages }
 * @returns pagination metadata, or null if not a paginated response
 */
export function getPaginationMeta(response: unknown): PaginationMeta | null {
  if (typeof response !== 'object' || response === null) {
    return null;
  }

  const r = response as Record<string, unknown>;
  if (
    typeof r.total !== 'number' ||
    typeof r.page !== 'number' ||
    typeof r.pageSize !== 'number' ||
    typeof r.totalPages !== 'number'
  ) {
    return null;
  }

  return {
    total: r.total,
    page: r.page,
    pageSize: r.pageSize,
    totalPages: r.totalPages,
  };
}
