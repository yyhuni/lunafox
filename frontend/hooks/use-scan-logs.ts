/**
 * Scan log polling hook
 * 
 * Function:
 * - Initial load to get all logs
 * - Incremental polling to obtain new logs (3s interval)
 * - Stop polling after scanning is complete
 */

import { useState, useEffect, useCallback, useRef } from 'react'
import { getScanLogs, type ScanLog } from '@/services/scan.service'

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
  const lastLogIDRef = useRef<number | null>(null)
  const isMounted = useRef(true)
  
  const clampLogs = useCallback((items: ScanLog[]) => {
    if (!maxLogs || maxLogs <= 0) return items
    return items.length > maxLogs ? items.slice(-maxLogs) : items
  }, [maxLogs])

  const fetchLogs = useCallback(async (incremental = false) => {
    if (!enabled || !isMounted.current) return
    
    setLoading(true)
    try {
      const params: { limit: number; afterId?: number } = { limit: 200 }
      if (incremental && lastLogIDRef.current !== null) {
        params.afterId = lastLogIDRef.current
      }
      
      const response = await getScanLogs(scanId, params)
      const newLogs = response.results
      if (newLogs.length > 0) {
        lastLogIDRef.current = newLogs[newLogs.length - 1].id
      }
      
      if (!isMounted.current) return
      
      if (newLogs.length > 0) {
        if (incremental) {
          // Deduplication by ID to prevent duplication caused by React Strict Mode or race conditions
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
      lastLogIDRef.current = null
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
    lastLogIDRef.current = null
    fetchLogs(false)
  }, [fetchLogs])

  // When maxLogs changes, proactively trim the cache to avoid long-running memory usage growth.
  useEffect(() => {
    if (!maxLogs || maxLogs <= 0) return
    setLogs(prev => (prev.length > maxLogs ? prev.slice(-maxLogs) : prev))
  }, [maxLogs])
  
  return { logs, loading, refetch }
}
