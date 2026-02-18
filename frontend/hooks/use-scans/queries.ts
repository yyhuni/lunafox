import { useQuery, keepPreviousData } from "@tanstack/react-query"
import {
  getScans,
  getScan,
  getScanStatistics,
} from "@/services/scan.service"
import type { GetScansParams } from "@/types/scan.types"
import { scanKeys } from "./keys"

export function useScans(params: GetScansParams = { page: 1, pageSize: 10 }) {
  return useQuery({
    queryKey: scanKeys.list(params),
    queryFn: () => getScans(params),
    placeholderData: keepPreviousData,
  })
}

export function useRunningScans(page = 1, pageSize = 10) {
  return useScans({ page, pageSize, status: "running" })
}

/**
 * Get the scan history of a target
 */
export function useTargetScans(targetId: number, pageSize = 5) {
  return useQuery({
    queryKey: scanKeys.target(targetId, pageSize),
    queryFn: () => getScans({ target: targetId, pageSize }),
    enabled: !!targetId,
  })
}

export function useScan(id: number) {
  return useQuery({
    queryKey: scanKeys.detail(id),
    queryFn: () => getScan(id),
    enabled: !!id,
  })
}

/**
 * Get scan statistics
 */
export function useScanStatistics() {
  return useQuery({
    queryKey: scanKeys.statistics(),
    queryFn: getScanStatistics,
  })
}
