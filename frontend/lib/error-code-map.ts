/**
 * Mapping of error codes to i18n keys
 * 
 * Adopt a simplified solution (refer to the practices of major manufacturers such as Stripe and GitHub):
 * - Only map common error codes (5-10)
 * - For unknown error codes use errors.unknown
 * - Error codes are consistent with the backend ErrorCodes class
 * 
 * Backend error code definition: backend/apps/common/error_codes.py
 */

/**
 * Error code to i18n key mapping table
 * 
 * Key: Error code returned by the backend (capital letters and underscores)
 * Value: frontend i18n key (defined in messages/en.json and messages/zh.json)
 */
export const ERROR_CODE_MAP: Record<string, string> = {
  // Common error codes (8, consistent with the backend ErrorCodes class)
  VALIDATION_ERROR: 'errors.validation',
  NOT_FOUND: 'errors.notFound',
  PERMISSION_DENIED: 'errors.permissionDenied',
  SERVER_ERROR: 'errors.serverError',
  BAD_REQUEST: 'errors.badRequest',
  CONFLICT: 'errors.conflict',
  UNAUTHORIZED: 'errors.unauthorized',
  RATE_LIMITED: 'errors.rateLimited',
};

/**
 * Default error i18n key
 * Fallback for unknown error codes
 */
export const DEFAULT_ERROR_KEY = 'errors.unknown';

/**
 * Get the i18n key corresponding to the error code
 * 
 * @param code - the error code returned by the backend
 * @returns corresponding i18n key, unknown error code returns 'errors.unknown'
 * 
 * @example
 * const errorKey = getErrorI18nKey('NOT_FOUND');
 * // Return: 'errors.notFound'
 * 
 * const unknownKey = getErrorI18nKey('SOME_UNKNOWN_ERROR');
 * // Return: 'errors.unknown'
 */
export function getErrorI18nKey(code: string): string {
  return ERROR_CODE_MAP[code] ?? DEFAULT_ERROR_KEY;
}

/**
 * Check if the error code is known
 * 
 * @param code - the error code returned by the backend
 * @returns true if the error code is in the mapping table
 */
export function isKnownErrorCode(code: string): boolean {
  return code in ERROR_CODE_MAP;
}

/**
 * Get a list of all known error codes
 * 
 * @returns error code array
 */
export function getAllErrorCodes(): string[] {
  return Object.keys(ERROR_CODE_MAP);
}
