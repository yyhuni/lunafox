import type { QueryClient } from '@tanstack/react-query'

export type QuickScanResponse = {
  scans?: unknown[]
  targetStats?: { created?: number }
  count?: number
}

export type InitiateScanResponse = Record<string, unknown>
export type BulkDeleteScansResponse = { deletedCount?: number }
export type StopScanResponse = { revokedTaskCount?: number }

export type ScanMutationInvalidationKeys = {
  all: readonly unknown[]
  statistics: readonly unknown[]
}

export type ScanMutationSuccessOptions<TData> = {
  response: unknown
  parse: (response: unknown) => TData | null
  onValidData: (data: TData) => void
  queryClient: QueryClient
  invalidateKeys: ScanMutationInvalidationKeys
}

export const invalidateScanQueries = async (
  queryClient: QueryClient,
  keys: ScanMutationInvalidationKeys
): Promise<void> => {
  await Promise.all([
    queryClient.invalidateQueries({ queryKey: keys.all }),
    queryClient.invalidateQueries({ queryKey: keys.statistics }),
  ])
}

export const handleScanMutationSuccess = async <TData>(
  options: ScanMutationSuccessOptions<TData>
): Promise<boolean> => {
  const data = options.parse(options.response)
  if (!data) return false

  options.onValidData(data)
  await invalidateScanQueries(options.queryClient, options.invalidateKeys)
  return true
}

export const getQuickScanSuccessCount = (data: QuickScanResponse): number =>
  data.scans?.length || data.targetStats?.created || data.count || 0

export const getBulkDeleteSuccessCount = (
  data: BulkDeleteScansResponse,
  fallbackCount: number
): number => data.deletedCount || fallbackCount || 0

export const getStopScanSuccessCount = (data: StopScanResponse): number =>
  data.revokedTaskCount || 1
