"use client"

import { useBroadcastSSE } from "@/hooks/use-broadcast-sse"
import { useNudgeGuardian } from "@/hooks/use-nudge-guardian"

/**
 * Non-critical background hooks used by the sidebar.
 * Split into a separate chunk to avoid inflating initial shell payload.
 */
export function AppSidebarBackgroundTasks() {
  useBroadcastSSE()
  useNudgeGuardian()
  return null
}

