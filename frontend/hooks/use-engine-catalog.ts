import { useQueries, useQuery } from "@tanstack/react-query"
import { useCallback, useRef } from "react"
import { getEngineCatalog, getEngineCatalogDetail, installEngine, type EngineInstallRequest } from "@/services/engine-catalog.service"
import { getEngineReplacementConflict } from "@/lib/engine-catalog"
import { useResourceMutation } from "@/hooks/_shared/create-resource-mutation"
import { getErrorCode, getErrorResponseData } from "@/lib/response-parser"

export const engineCatalogKeys = {
  list: ["engine-catalog"] as const,
  detail: (engineId: string) => ["engine-catalog", engineId] as const,
}

export function useEngineCatalog(enabled = true) {
  return useQuery({ queryKey: engineCatalogKeys.list, queryFn: getEngineCatalog, enabled })
}

export function useEngineCatalogDetail(engineId: string, enabled = true) {
  const normalizedEngineId = engineId.trim()
  return useQuery({
    queryKey: engineCatalogKeys.detail(normalizedEngineId),
    queryFn: () => getEngineCatalogDetail(normalizedEngineId),
    enabled: enabled && normalizedEngineId.length > 0,
    staleTime: Number.POSITIVE_INFINITY,
  })
}

export function useInstallEngine() {
  const mutation = useResourceMutation({
    mutationFn: (request: EngineInstallRequest) => installEngine(request),
    loadingToast: { key: "common.status.creating", id: "install-engine" },
    invalidate: [{ queryKey: engineCatalogKeys.list }],
    errorFallbackKey: "common.errors.operationFailed",
    onError: ({ error, toast }) => {
      // A replacement conflict is resolved by the page's confirmation dialog,
      // so reporting it as a terminal installation failure is misleading.
      if (getEngineReplacementConflict(error)) return
      toast.errorFromCode(getErrorCode(getErrorResponseData(error)), "common.errors.operationFailed")
    },
  })
  const inFlight = useRef<Promise<Awaited<ReturnType<typeof installEngine>>> | null>(null)
  const runInstallation = mutation.mutateAsync
  const mutateAsync = useCallback((request: EngineInstallRequest) => {
    if (inFlight.current) return inFlight.current
    const pending = runInstallation(request)
    inFlight.current = pending
    void pending.then(
      () => { inFlight.current = null },
      () => { inFlight.current = null },
    )
    return pending
  }, [runInstallation])

  return { ...mutation, mutateAsync }
}

export function useEngineCatalogDetails(engineIds: string[]) {
  const distinctEngineIds = Array.from(new Set(engineIds.map((engineId) => engineId.trim()).filter(Boolean)))
  const queries = useQueries({
    queries: distinctEngineIds.map((engineId) => ({
      queryKey: engineCatalogKeys.detail(engineId),
      queryFn: () => getEngineCatalogDetail(engineId),
      staleTime: Number.POSITIVE_INFINITY,
    })),
  })
  return {
    data: queries.every((query) => query.data !== undefined) ? queries.map((query) => query.data!) : undefined,
    isLoading: queries.some((query) => query.isPending),
    isError: queries.some((query) => query.isError),
    errors: queries.flatMap((query) => query.error ? [query.error] : []),
  }
}
