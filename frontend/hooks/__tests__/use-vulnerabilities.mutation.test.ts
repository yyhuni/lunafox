import { act, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import {
  useBulkMarkAsReviewed,
  useBulkMarkAsUnreviewed,
  useMarkAsReviewed,
  useMarkAsUnreviewed,
  useTargetVulnerabilities,
  vulnerabilityKeys,
} from "@/hooks/use-vulnerabilities"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import { createTestQueryClient } from "@/test/utils/test-query-client"

const vulnerabilityServiceMocks = vi.hoisted(() => ({
  getAllVulnerabilities: vi.fn(),
  getVulnerabilityById: vi.fn(),
  getVulnerabilitiesByScanId: vi.fn(),
  getVulnerabilitiesByTargetId: vi.fn(),
  markAsReviewed: vi.fn(),
  markAsUnreviewed: vi.fn(),
  bulkMarkAsReviewed: vi.fn(),
  bulkMarkAsUnreviewed: vi.fn(),
  getStats: vi.fn(),
  getStatsByTargetId: vi.fn(),
}))

const toastMocks = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  errorFromCode: vi.fn(),
  loading: vi.fn(),
  warning: vi.fn(),
  dismiss: vi.fn(),
}))

vi.mock("@/services/vulnerability.service", () => ({
  VulnerabilityService: vulnerabilityServiceMocks,
}))

vi.mock("@/lib/toast-helpers", () => ({
  useToastMessages: () => toastMocks,
}))

describe("use-vulnerabilities", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("target 列表查询会保持字段标准化与 target 回填", async () => {
    vulnerabilityServiceMocks.getVulnerabilitiesByTargetId.mockResolvedValue({
      results: [
        {
          id: 11,
          severity: "unknown",
          cvssScore: "5.6",
          vulnType: "xss",
          url: "https://example.com",
          source: "nuclei",
          isReviewed: false,
          reviewedAt: null,
          createdAt: "2026-02-11T00:00:00Z",
        },
      ],
      total: 1,
      page: 2,
      pageSize: 20,
      totalPages: 1,
    })

    const { result } = renderHookWithProviders(() =>
      useTargetVulnerabilities(99, { page: 2, pageSize: 20 })
    )

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(vulnerabilityServiceMocks.getVulnerabilitiesByTargetId).toHaveBeenCalledWith(
      99,
      { page: 2, pageSize: 20 },
      undefined
    )
    expect(result.current.data?.vulnerabilities[0]).toMatchObject({
      id: 11,
      severity: "info",
      cvssScore: 5.6,
      target: 99,
    })
    expect(result.current.data?.pagination).toMatchObject({
      page: 2,
      pageSize: 20,
      total: 1,
      totalPages: 1,
    })
  })

  it("bulk reviewed 成功时保持成功提示与全量失效范围", async () => {
    vulnerabilityServiceMocks.bulkMarkAsReviewed.mockResolvedValue({
      updatedCount: 3,
    })
    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useBulkMarkAsReviewed(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync([1, 2, 3])
    })

    expect(vulnerabilityServiceMocks.bulkMarkAsReviewed).toHaveBeenCalledWith([1, 2, 3])
    expect(toastMocks.success).toHaveBeenCalledWith(
      "vulnerabilities.bulkReviewSuccess",
      { count: 3 }
    )
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: vulnerabilityKeys.all,
    })
  })

  it("bulk reviewed 失败时保留错误码映射与 fallback key", async () => {
    vulnerabilityServiceMocks.bulkMarkAsReviewed.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useBulkMarkAsReviewed())

    await act(async () => {
      await expect(result.current.mutateAsync([1, 2, 3])).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "vulnerabilities.bulkReviewError"
    )
  })

  it("标记已审阅成功时提示成功并失效列表", async () => {
    vulnerabilityServiceMocks.markAsReviewed.mockResolvedValue({
      message: "ok",
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useMarkAsReviewed(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync(11)
    })

    expect(vulnerabilityServiceMocks.markAsReviewed).toHaveBeenCalledWith(11)
    expect(toastMocks.success).toHaveBeenCalledWith("vulnerabilities.reviewSuccess")
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: vulnerabilityKeys.all })
  })

  it("标记已审阅失败时保留错误码映射与 fallback key", async () => {
    vulnerabilityServiceMocks.markAsReviewed.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useMarkAsReviewed())

    await act(async () => {
      await expect(result.current.mutateAsync(11)).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "vulnerabilities.reviewError"
    )
  })

  it("标记未审阅成功时提示成功并失效列表", async () => {
    vulnerabilityServiceMocks.markAsUnreviewed.mockResolvedValue({
      message: "ok",
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useMarkAsUnreviewed(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync(12)
    })

    expect(vulnerabilityServiceMocks.markAsUnreviewed).toHaveBeenCalledWith(12)
    expect(toastMocks.success).toHaveBeenCalledWith("vulnerabilities.unreviewSuccess")
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: vulnerabilityKeys.all })
  })

  it("标记未审阅失败时保留错误码映射与 fallback key", async () => {
    vulnerabilityServiceMocks.markAsUnreviewed.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useMarkAsUnreviewed())

    await act(async () => {
      await expect(result.current.mutateAsync(12)).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "vulnerabilities.unreviewError"
    )
  })

  it("bulk 未审阅成功时提示成功并失效列表", async () => {
    vulnerabilityServiceMocks.bulkMarkAsUnreviewed.mockResolvedValue({
      updatedCount: 2,
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useBulkMarkAsUnreviewed(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync([4, 5])
    })

    expect(vulnerabilityServiceMocks.bulkMarkAsUnreviewed).toHaveBeenCalledWith([4, 5])
    expect(toastMocks.success).toHaveBeenCalledWith(
      "vulnerabilities.bulkUnreviewSuccess",
      { count: 2 }
    )
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: vulnerabilityKeys.all })
  })

  it("bulk 未审阅失败时保留错误码映射与 fallback key", async () => {
    vulnerabilityServiceMocks.bulkMarkAsUnreviewed.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useBulkMarkAsUnreviewed())

    await act(async () => {
      await expect(result.current.mutateAsync([4, 5])).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "vulnerabilities.bulkUnreviewError"
    )
  })
})
