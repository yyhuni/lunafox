"use client"

import * as React from "react"
import { USE_MOCK } from "@/mock/config"
import { cleanupLegacyMockWorker } from "@/mock/legacy-worker-cleanup"

export function MockProvider({
  children,
  enabled = USE_MOCK,
}: {
  children: React.ReactNode
  enabled?: boolean
}) {
  React.useEffect(() => {
    async function ensureMockPlatform() {
      if (!enabled) {
        await cleanupLegacyMockWorker()
        return
      }

      const { startMockWorker } = await import("@/mock/browser")
      await startMockWorker()
    }

    void ensureMockPlatform()
  }, [enabled])

  // Keep the server and first client tree stable; requests wait for the same
  // mock bootstrap promise in the transport layer before they leave the browser.
  return <>{children}</>
}
