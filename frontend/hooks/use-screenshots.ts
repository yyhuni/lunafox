import { useQuery, keepPreviousData } from '@tanstack/react-query'
import { createResourceKeys } from "@/hooks/_shared/query-keys"
import { ScreenshotService } from '@/services/screenshot.service'

// Query Keys
const screenshotKeyBase = createResourceKeys("screenshots")

export const screenshotKeys = {
  ...screenshotKeyBase,
  target: (targetId: number, params: { page: number; pageSize: number; filter?: string }) =>
    [...screenshotKeyBase.all, 'target', targetId, params] as const,
  scan: (scanId: number, params: { page: number; pageSize: number; filter?: string }) =>
    [...screenshotKeyBase.all, 'scan', scanId, params] as const,
}

// Get a list of screenshots of the target
export function useTargetScreenshots(
  targetId: number,
  params: { page: number; pageSize: number; filter?: string },
  options?: { enabled?: boolean }
) {
  return useQuery({
    queryKey: screenshotKeys.target(targetId, params),
    queryFn: () => ScreenshotService.getByTarget(targetId, params),
    enabled: options?.enabled ?? true,
    placeholderData: keepPreviousData,
  })
}

// Get a list of scanned screenshots
export function useScanScreenshots(
  scanId: number,
  params: { page: number; pageSize: number; filter?: string },
  options?: { enabled?: boolean }
) {
  return useQuery({
    queryKey: screenshotKeys.scan(scanId, params),
    queryFn: () => ScreenshotService.getByScan(scanId, params),
    enabled: options?.enabled ?? true,
    placeholderData: keepPreviousData,
  })
}
