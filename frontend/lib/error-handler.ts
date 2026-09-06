/**
 * Unified error handling utility
 * 
 * According to project rule 24:
 * - Success messages: Frontend constructs them
 * - Error messages: Frontend constructs them, providing more specific error reasons
 */

import { toastFeedback } from "@/lib/toast-helpers"

/**
 * API error information interface
 */
export interface ApiError {
  response?: {
    data?: unknown
  }
  message?: string
}

/**
 * Handle mutation errors (generic)
 * @param error Error object
 * @param userMessage Frontend custom user-friendly error message
 * @param toastId Optional toast ID for replacing loading feedback in place
 */
export function handleMutationError(
  error: unknown,
  userMessage: string,
  toastId?: string
) {
  toastFeedback.error(userMessage, toastId ? { id: toastId } : undefined)
}

/**
 * Handle query errors (generic)
 * @param error Error object
 * @param userMessage Frontend custom user-friendly error message
 */
export function handleQueryError(error: unknown, userMessage: string) {
  // Show frontend custom user-friendly error message
  toastFeedback.error(userMessage)
}

/**
 * Handle success response (generic)
 * @param response Backend response
 * @param successMessage Frontend custom success message
 * @param toastId Optional toast ID for replacing loading feedback in place
 */
export function handleSuccess(
  response: unknown,
  successMessage: string,
  toastId?: string
) {
  toastFeedback.success(successMessage, toastId ? { id: toastId } : undefined)
}

/**
 * Handle warning response (partial success scenarios)
 * @param response Backend response
 * @param warningMessage Frontend custom warning message
 * @param toastId Optional toast ID for replacing loading feedback in place
 */
export function handleWarning(
  response: unknown,
  warningMessage: string,
  toastId?: string
) {
  toastFeedback.warning(warningMessage, toastId ? { id: toastId } : undefined)
}

/**
 * Check if response is successful
 * @param response API response
 * @returns Whether successful
 */
export function isSuccessResponse(response: unknown): boolean {
  return (response as { state?: string })?.state === 'success'
}

/**
 * Extract data from response
 * @param response API response
 * @param defaultValue Default value
 * @returns Response data
 */
export function extractData<T>(response: unknown, defaultValue: T): T {
  if (isSuccessResponse(response) && (response as { data?: T }).data) {
    return (response as { data: T }).data
  }
  return defaultValue
}
