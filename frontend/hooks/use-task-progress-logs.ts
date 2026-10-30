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
  const generationRef = useRef(0)
  const fetchingRef = useRef(false)
  const refreshPendingRef = useRef(false)
  const pollingEnabledRef = useRef(false)
  
  const clampLogs = useCallback((items: ScanLog[]) => {
    if (!maxLogs || maxLogs <= 0) return items
    return items.length > maxLogs ? items.slice(-maxLogs) : items
  }, [maxLogs])

  const fetchLogs = useCallback(async (incremental = false) => {
    if (!enabled || !isMounted.current) return
    if (fetchingRef.current) {
      refreshPendingRef.current = true
      return
    }

    const generation = generationRef.current
    fetchingRef.current = true
    setLoading(true)
    let pageToken = incremental ? nextPageTokenRef.current : undefined
    let replace = !incremental
    try {
      // Drain every available page on initial load and each polling tick. A
      // completed task has no future poll to recover pages omitted here.
      while (isMounted.current && generation === generationRef.current) {
        const params = { pageSize: 200, ...(pageToken ? { pageToken } : {}) }
        const response = await getScanLogs(scanId, params)
        if (!isMounted.current || generation !== generationRef.current) return

        const replacePage = replace
        setLogs(prev => {
          const existing = replacePage ? [] : prev
          const lastId = existing.at(-1)?.id ?? 0
          // Tail rereads can include IDs already evicted from a small cache.
          // Only append newer IDs, or those old rows would displace recent logs.
          const additions = response.results.filter(log => log.id > lastId)
          // An unchanged polling response must not publish a new array. The
          // runtime detail view groups and renders these logs, so preserving
          // the reference keeps a no-op poll from invalidating that subtree.
          if (!replacePage && additions.length === 0) return prev
          return clampLogs([...existing, ...additions])
        })
        replace = false

        // AIP omits nextPageToken at the current tail. Keep the token used to
        // request that tail page, so append-only polling re-reads/deduplicates
        // the tail instead of jumping back to page one after >200 entries.
        nextPageTokenRef.current = response.nextPageToken || pageToken
        if (!response.nextPageToken || response.nextPageToken === pageToken) break
        pageToken = response.nextPageToken
      }
    } catch (error) {
      void error
    } finally {
      if (isMounted.current && generation === generationRef.current) {
        fetchingRef.current = false
        setLoading(false)
        if (refreshPendingRef.current) {
          refreshPendingRef.current = false
          void fetchLogs(true)
        }
      }
    }
  }, [scanId, enabled, clampLogs])

  // initial load
  useEffect(() => {
    isMounted.current = true
    generationRef.current++
    fetchingRef.current = false
    refreshPendingRef.current = false
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
    const wasPolling = pollingEnabledRef.current
    const isPolling = enabled && pollingInterval > 0
    pollingEnabledRef.current = isPolling
    // pollingInterval <= 0 means disabling polling (to avoid high-frequency requests/stuck caused by setInterval(0))
    if (!isPolling) {
      // A terminal scan stops polling after Server commits its final event.
      // Fetch once more, including after any older in-flight request finishes.
      if (enabled && wasPolling) void fetchLogs(true)
      return
    }

    const interval = setInterval(() => {
      fetchLogs(true) // incremental query
    }, pollingInterval)

    return () => clearInterval(interval)
  }, [enabled, pollingInterval, fetchLogs])
  
  const refetch = useCallback(() => {
    generationRef.current++
    fetchingRef.current = false
    refreshPendingRef.current = false
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
