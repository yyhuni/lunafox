import { useMutation, useQueryClient } from "@tanstack/react-query"
import { agentKeys } from "@/hooks/use-agents"
import { activateAgentLicense } from "@/services/agent-license.service"
import { getErrorResponseData } from "@/lib/response-parser"

export function useActivateAgentLicense() {
  const queryClient = useQueryClient()
  const mutation = useMutation({
    mutationFn: activateAgentLicense,
    retry: false,
    onSuccess: async () => {
      // Eligibility belongs to the whole registry, including cached picker pages and details.
      await queryClient.invalidateQueries({ queryKey: agentKeys.all })
    },
  })
  const body = getErrorResponseData(mutation.error) as { error?: { details?: unknown[] } } | undefined
  const detail = Array.isArray(body?.error?.details) ? body.error.details.find((value) => {
    if (!value || typeof value !== "object") return false
    const row = value as Record<string, unknown>
    return row["@type"] === "type.googleapis.com/google.rpc.ErrorInfo" && row.domain === "lunafox.agentLicense"
  }) as { reason?: string } | undefined : undefined
  return { ...mutation, errorReason: detail?.reason }
}
