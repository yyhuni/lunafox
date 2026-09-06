/**
 * Scan log polling hook
 * 
 * Function:
 * - Initial load to get all logs
 * - Incremental polling to obtain new logs (3s interval)
 * - Polling is controlled by the caller through `enabled`
 */

import { useState, useEffect, useCallback, useRef } from 'react'
import { getScanLogs } from '@/services/scan.service'
import type { ScanLog } from '@/types/scan.types'

interface UseScanLogsOptions {
  scanId: number
  enabled?: boolean
  pollingInterval?: number  // Default 3000ms
  maxLogs?: number  // Default is 5000, <=0 means no limit
}

interface UseScanLogsReturn {
  logs: ScanLog[]
  loading: boolean
  refetch: () => void
}

export function useScanLogs({
  scanId,
  enabled = true,
  pollingInterval = 3000,
  maxLogs = 5000,
}: UseScanLogsOptions): UseScanLogsReturn {
  const [logs, setLogs] = useState<ScanLog[]>([])
  const [loading, setLoading] = useState(false)
  const nextPageTokenRef = useRef<string | undefined>(undefined)
  const isMounted = useRef(true)
  
  const clampLogs = useCallback((items: ScanLog[]) => {
    if (!maxLogs || maxLogs <= 0) return items
    return items.length > maxLogs ? items.slice(-maxLogs) : items
  }, [maxLogs])

  const fetchLogs = useCallback(async (incremental = false) => {
    if (!enabled || !isMounted.current) return
    
    setLoading(true)
    try {
      const params: { pageSize: number; pageToken?: string } = { pageSize: 200 }
      // Empty nextPageToken is not a terminal signal for append-only runtime logs.
      // Keep polling from the first page and de-duplicate locally so newly appended
      // engine events still appear after a short or empty page.
      if (incremental && nextPageTokenRef.current) {
        params.pageToken = nextPageTokenRef.current
      }
      
      const response = await getScanLogs(scanId, params)
      const newLogs = response.results
      nextPageTokenRef.current = response.nextPageToken || undefined
      
      if (!isMounted.current) return
      
      if (newLogs.length > 0) {
        if (incremental) {
          setLogs(prev => {
            const existingIds = new Set(prev.map(l => l.id))
            const uniqueNewLogs = newLogs.filter(l => !existingIds.has(l.id))
            if (uniqueNewLogs.length === 0) return prev
            return clampLogs([...prev, ...uniqueNewLogs])
          })
        } else {
          setLogs(clampLogs(newLogs))
        }
      }
    } catch (error) {
      void error
    } finally {
      if (isMounted.current) {
        setLoading(false)
      }
    }
  }, [scanId, enabled, clampLogs])
  
  // initial load
  useEffect(() => {
    isMounted.current = true
    if (enabled) {
      // reset state
      setLogs([])
      nextPageTokenRef.current = undefined
      fetchLogs(false)
    }
    return () => {
      isMounted.current = false
    }
  }, [scanId, enabled, fetchLogs])
  
  // polling
  useEffect(() => {
    if (!enabled) return
    // pollingInterval <= 0 means disabling polling (to avoid high-frequency requests/stuck caused by setInterval(0))
    if (!pollingInterval || pollingInterval <= 0) return

    const interval = setInterval(() => {
      fetchLogs(true) // incremental query
    }, pollingInterval)

    return () => clearInterval(interval)
  }, [enabled, pollingInterval, fetchLogs])
  
  const refetch = useCallback(() => {
    setLogs([])
    nextPageTokenRef.current = undefined
    fetchLogs(false)
  }, [fetchLogs])

  // When maxLogs changes, proactively trim the cache to avoid long-running memory usage growth.
  useEffect(() => {
    if (!maxLogs || maxLogs <= 0) return
    setLogs(prev => (prev.length > maxLogs ? prev.slice(-maxLogs) : prev))
  }, [maxLogs])

  return { logs, loading, refetch }
}

export { useScanLogs as useTaskProgressLogs }
