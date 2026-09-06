import { act, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import {
	scheduledScanOverviewKey,
	useScheduledScanOverviewSummary,
} from "@/hooks/use-scheduled-scans"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"

const scheduledScanServiceMocks = vi.hoisted(() => ({
	getScheduledScanOverviewSummary: vi.fn(),
}))

vi.mock("@/services/scheduled-scan.service", () => ({
	getScheduledScanOverviewSummary: scheduledScanServiceMocks.getScheduledScanOverviewSummary,
}))

const overview = (enabledScheduledScanCount: number) => ({
	asOfTime: "2026-06-14T08:00:00Z",
	enabledScheduledScanCount,
	pausedScheduledScanCount: 1,
	todayScheduledScanCount: 2,
	next24HoursScheduledScanCount: 3,
	upcomingScheduledScans: [],
})

describe("useScheduledScanOverviewSummary", () => {
	beforeEach(() => {
		vi.clearAllMocks()
	})

	it("uses a fixed summary key and identifies an initial failure", async () => {
		scheduledScanServiceMocks.getScheduledScanOverviewSummary.mockRejectedValue(new Error("unavailable"))
		const { result } = renderHookWithProviders(() => useScheduledScanOverviewSummary())

		await waitFor(() => {
			expect(result.current.isInitialError).toBe(true)
		})
		expect(result.current.data).toBeUndefined()
		expect(result.current.isRefetchStale).toBe(false)
		expect(result.current.lastSuccessfulAt).toBeNull()
		expect(scheduledScanOverviewKey).toEqual(["scheduled-scans", "overview"])
		expect(scheduledScanServiceMocks.getScheduledScanOverviewSummary).toHaveBeenCalledWith()
	})

	it("retains the last successful snapshot after refresh failure and clears stale state on recovery", async () => {
		const initial = overview(3)
		scheduledScanServiceMocks.getScheduledScanOverviewSummary.mockResolvedValueOnce(initial)
		const { result } = renderHookWithProviders(() => useScheduledScanOverviewSummary())

		await waitFor(() => {
			expect(result.current.data).toBe(initial)
		})
		const firstSuccessfulAt = result.current.lastSuccessfulAt
		scheduledScanServiceMocks.getScheduledScanOverviewSummary.mockRejectedValueOnce(new Error("refresh unavailable"))
		await act(async () => {
			await result.current.refetch()
		})
		await waitFor(() => {
			expect(result.current.isRefetchStale).toBe(true)
		})
		expect(result.current.data).toBe(initial)
		expect(result.current.lastSuccessfulAt).toBe(firstSuccessfulAt)

		const recovered = overview(4)
		scheduledScanServiceMocks.getScheduledScanOverviewSummary.mockResolvedValueOnce(recovered)
		await act(async () => {
			await result.current.refetch()
		})
		await waitFor(() => {
			expect(result.current.data).toEqual(recovered)
			expect(result.current.isRefetchStale).toBe(false)
		})
	})
})
