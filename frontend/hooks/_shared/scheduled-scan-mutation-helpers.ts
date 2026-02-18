import { parseResponse } from "@/lib/response-parser"

export type ScheduledScanSuccessOptions = {
  response: unknown
  onSuccess: () => void
  parse?: (response: unknown) => unknown
}

export const handleScheduledScanMutationSuccess = (
  options: ScheduledScanSuccessOptions
): void => {
  const parse = options.parse ?? parseResponse
  parse(options.response)
  options.onSuccess()
}
