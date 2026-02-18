/**
 * Unified error handling utility
 * 
 * According to project rule 24:
 * - Success messages: Frontend constructs them
 * - Error messages: Frontend constructs them, providing more specific error reasons
 */

import { toast } from "sonner"

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
 * @param toastId Optional toast ID (for dismissing loading toast)
 */
export function handleMutationError(
  error: unknown,
  userMessage: string,
  toastId?: string
) {
  // Dismiss loading toast (if any)
  if (toastId) {
    toast.dismiss(toastId)
  }

  // Show frontend custom user-friendly error message
  toast.error(userMessage)
}

/**
 * Handle query errors (generic)
 * @param error Error object
 * @param userMessage Frontend custom user-friendly error message
 */
export function handleQueryError(error: unknown, userMessage: string) {
  // Show frontend custom user-friendly error message
  toast.error(userMessage)
}

/**
 * Handle success response (generic)
 * @param response Backend response
 * @param successMessage Frontend custom success message
 * @param toastId Optional toast ID (for dismissing loading toast)
 */
export function handleSuccess(
  response: unknown,
  successMessage: string,
  toastId?: string
) {
  // Dismiss loading toast (if any)
  if (toastId) {
    toast.dismiss(toastId)
  }

  // Show frontend custom success message
  toast.success(successMessage)
}

/**
 * Handle warning response (partial success scenarios)
 * @param response Backend response
 * @param warningMessage Frontend custom warning message
 * @param toastId Optional toast ID (for dismissing loading toast)
 */
export function handleWarning(
  response: unknown,
  warningMessage: string,
  toastId?: string
) {
  // Dismiss loading toast (if any)
  if (toastId) {
    toast.dismiss(toastId)
  }

  // Show frontend custom warning message
  toast.warning(warningMessage)
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
