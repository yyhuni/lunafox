import { act, waitFor } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import type { NucleiPocSyncTask } from "@/types/nuclei-poc.types"

const serviceMocks = vi.hoisted(() => ({
  getNucleiPoc: vi.fn(),
  getNucleiPocActiveSyncTaskName: vi.fn(),
  getNucleiPocErrorBody: vi.fn(),
  getNucleiPocErrorReason: vi.fn(),
  getNucleiPocFilterOptions: vi.fn(),
  getNucleiPocHttpStatus: vi.fn(),
  getNucleiPocSource: vi.fn(),
  getNucleiPocSyncTask: vi.fn(),
  listNucleiPocs: vi.fn(),
  syncNucleiPocSource: vi.fn(),
  updateNucleiPoc: vi.fn(),
}))

vi.mock("@/services/nuclei-poc.service", () => serviceMocks)

import { nucleiPocKeys, useNucleiPocSyncTask } from "@/hooks/use-nuclei-pocs"

const taskName = "nucleiPocSyncTasks/00000000-0000-4000-8000-000000000001" as const

function task(overrides: Partial<NucleiPocSyncTask> = {}): NucleiPocSyncTask {
  return {
    name: taskName,
    requestId: "00000000-0000-4000-8000-000000000002",
    sourceType: "git",
    state: "SCANNING_FILES",
    phase: "SCANNING_FILES",
    counters: { filesSeen: 2, yamlFilesSeen: 1, templatesValidated: 0, bytesRead: 512 },
    diagnostics: { samples: [], total: 0, truncated: false },
    cleanupStatus: "pending",
    createdAt: "2026-08-19T00:00:00.000Z",
    startedAt: "2026-08-19T00:00:00.000Z",
    completedAt: null,
    updatedAt: "2026-08-19T00:00:00.000Z",
    ...overrides,
  }
}

describe("useNucleiPocSyncTask", () => {
  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
    vi.clearAllMocks()
    serviceMocks.getNucleiPocHttpStatus.mockReturnValue(undefined)
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it("continues one-second polling while the page owns a non-terminal task", async () => {
    serviceMocks.getNucleiPocSyncTask.mockResolvedValue(task())

    const { result } = renderHookWithProviders(() => useNucleiPocSyncTask(taskName))
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    const initialCalls = serviceMocks.getNucleiPocSyncTask.mock.calls.length

    await act(async () => {
      await vi.advanceTimersByTimeAsync(2100)
    })

    expect(serviceMocks.getNucleiPocSyncTask.mock.calls.length).toBeGreaterThan(initialCalls)
  })

  it("stops polling at a terminal task and invalidates catalog projections exactly once", async () => {
    serviceMocks.getNucleiPocSyncTask.mockResolvedValue(task({
      state: "SUCCEEDED",
      phase: "SUCCEEDED",
      completedAt: "2026-08-19T00:01:00.000Z",
      cleanupStatus: "clean",
      commitSha: "abcdef1234567890",
      committedPocCount: 1,
    }))

    const { queryClient, result } = renderHookWithProviders(() => useNucleiPocSyncTask(taskName))
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    await waitFor(() => expect(invalidateSpy).toHaveBeenCalledTimes(4))

    await act(async () => {
      await vi.advanceTimersByTimeAsync(3000)
    })

    expect(serviceMocks.getNucleiPocSyncTask).toHaveBeenCalledTimes(1)
    expect(invalidateSpy.mock.calls.map(([filters]) => filters?.queryKey)).toEqual([
      nucleiPocKeys.source(),
      nucleiPocKeys.lists(),
      nucleiPocKeys.filterOptions("tags"),
      nucleiPocKeys.details(),
    ])
  })
})
