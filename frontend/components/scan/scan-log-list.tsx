"use client"

import { ScanLogListContent, ScanLogListEmptyState, ScanLogListLoadingState } from "@/components/scan/scan-log-list-sections"
import { useScanLogListState } from "@/components/scan/scan-log-list-state"

import type { ScanLog } from "@/services/scan.service"

interface ScanLogListProps {
  logs: ScanLog[]
  loading?: boolean
}

/**
 * Scan log list component
 * Reuse the AnsiLogViewer component
 */
export function ScanLogList({ logs, loading }: ScanLogListProps) {
  const state = useScanLogListState({ logs })

  if (loading && logs.length === 0) {
    return <ScanLogListLoadingState />
  }

  if (logs.length === 0) {
    return <ScanLogListEmptyState />
  }

  return <ScanLogListContent content={state.content} />
}
