import { act } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { agentKeys } from "@/hooks/use-agents"
import { usePrefetchOverviewData } from "@/hooks/use-overview-prefetch"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"

describe("usePrefetchOverviewData", () => {
  it("prefetches the authoritative Agent summary and complete location map", async () => {
    const { queryClient, result } = renderHookWithProviders(() => usePrefetchOverviewData())
    const prefetch = vi.spyOn(queryClient, "prefetchQuery").mockResolvedValue(undefined)

    await act(async () => {
      await result.current()
    })

    const queryKeys = prefetch.mock.calls.map(([options]) => options.queryKey)
    expect(queryKeys).toContainEqual(agentKeys.clusterSummary())
    expect(queryKeys).toContainEqual(agentKeys.locationMap())
    expect(prefetch).toHaveBeenCalledTimes(6)
  })
})
