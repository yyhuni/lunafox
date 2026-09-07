const ERROR_INFO_TYPE = "type.googleapis.com/google.rpc.ErrorInfo"

export function hasApiErrorReason(error: unknown, reason: string): boolean {
  if (typeof error !== "object" || error === null || !("response" in error)) return false
  const response = error.response
  if (typeof response !== "object" || response === null || !("data" in response)) return false
  const data = response.data
  if (typeof data !== "object" || data === null || !("error" in data)) return false
  const apiError = data.error
  if (typeof apiError !== "object" || apiError === null || !("details" in apiError) || !Array.isArray(apiError.details)) return false

  return apiError.details.some((detail) => (
    typeof detail === "object"
    && detail !== null
    && detail["@type"] === ERROR_INFO_TYPE
    && detail.reason === reason
  ))
}
