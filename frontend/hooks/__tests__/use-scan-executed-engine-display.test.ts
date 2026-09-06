import { waitFor } from "@testing-library/react"
import { describe, expect, it } from "vitest"

import { useScanExecutedEngineDisplay } from "@/hooks/use-scan-executed-engine-display"
import { getMockScans } from "@/mock/data/scans"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"

describe("useScanExecutedEngineDisplay", () => {
  it("resolves scan engine IDs through the installed engine catalog", async () => {
    const scan = getMockScans({ pageSize: 1 }).results[0]
    const { result } = renderHookWithProviders(() => useScanExecutedEngineDisplay([scan]))

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })

    expect(result.current.error).toBeUndefined()
    expect(result.current.engineNamesByScanId.get(scan.id)).toEqual([
      "Subdomain Discovery",
      "Port Scan",
    ])
  })
})
