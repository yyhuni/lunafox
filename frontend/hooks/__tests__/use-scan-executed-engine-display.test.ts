import { waitFor } from "@testing-library/react"
import { describe, expect, it } from "vitest"

import { useScanExecutedEngineDisplay } from "@/hooks/use-scan-executed-engine-display"
import { getMockScanById } from "@/mock/data/scans"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"

describe("useScanExecutedEngineDisplay", () => {
  it("resolves scan engine IDs through the installed engine catalog", async () => {
    const scan = getMockScanById(9)
    if (!scan) {
      throw new Error("mock scan 9 is missing")
    }
    const { result } = renderHookWithProviders(() => useScanExecutedEngineDisplay([scan]))

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })

    expect(result.current.error).toBeUndefined()
    expect(result.current.engineNamesByScanId.get(scan.id)).toEqual([
      "Subdomain Discovery",
      "Port Scan",
      "Web Crawling",
      "Screenshot",
    ])
  })
})
