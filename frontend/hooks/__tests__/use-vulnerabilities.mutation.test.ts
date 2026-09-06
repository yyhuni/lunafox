import { act, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import {
  useBulkDeleteVulnerabilities,
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
  bulkDelete: vi.fn(),
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
      useTargetVulnerabilities(99, { pageToken: "target-vulnerability-cursor-2", pageSize: 20 })
    )

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(vulnerabilityServiceMocks.getVulnerabilitiesByTargetId).toHaveBeenCalledWith(
      99,
      {
        pageSize: 20,
        pageToken: "target-vulnerability-cursor-2",
        filter: undefined,
        orderBy: undefined,
      }
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

  it("bulk reviewed 成功时原位更新 loading 并保持全量失效范围", async () => {
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
    expect(toastMocks.loading).toHaveBeenCalledWith(
      "vulnerabilities.bulkReviewLoading",
      { count: 3 },
      "vulnerabilities-bulk-review"
    )
    expect(toastMocks.success).toHaveBeenCalledWith(
      "vulnerabilities.bulkReviewSuccess",
      { count: 3 },
      "vulnerabilities-bulk-review"
    )
    expect(toastMocks.dismiss).not.toHaveBeenCalled()
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

    expect(toastMocks.loading).toHaveBeenCalledWith(
      "vulnerabilities.bulkReviewLoading",
      { count: 3 },
      "vulnerabilities-bulk-review"
    )
    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "vulnerabilities.bulkReviewError",
      "vulnerabilities-bulk-review"
    )
    expect(toastMocks.dismiss).not.toHaveBeenCalled()
  })

  it("bulk delete 成功时原位更新 loading 并保持全量失效范围", async () => {
    vulnerabilityServiceMocks.bulkDelete.mockResolvedValue({
      deletedCount: 2,
    })
    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useBulkDeleteVulnerabilities(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync([7, 8])
    })

    expect(vulnerabilityServiceMocks.bulkDelete).toHaveBeenCalledWith([7, 8])
    expect(toastMocks.loading).toHaveBeenCalledWith(
      "vulnerabilities.bulkDeleteLoading",
      { count: 2 },
      "vulnerabilities-bulk-delete"
    )
    expect(toastMocks.success).toHaveBeenCalledWith(
      "vulnerabilities.bulkDeleteSuccess",
      { count: 2 },
      "vulnerabilities-bulk-delete"
    )
    expect(toastMocks.dismiss).not.toHaveBeenCalled()
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: vulnerabilityKeys.all,
    })
  })

  it("bulk delete 失败时保留错误码映射与 fallback key", async () => {
    vulnerabilityServiceMocks.bulkDelete.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useBulkDeleteVulnerabilities())

    await act(async () => {
      await expect(result.current.mutateAsync([7, 8])).rejects.toBeDefined()
    })

    expect(toastMocks.loading).toHaveBeenCalledWith(
      "vulnerabilities.bulkDeleteLoading",
      { count: 2 },
      "vulnerabilities-bulk-delete"
    )
    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "vulnerabilities.bulkDeleteError",
      "vulnerabilities-bulk-delete"
    )
    expect(toastMocks.dismiss).not.toHaveBeenCalled()
  })

  it("标记已审阅成功时原位更新 loading 并失效列表", async () => {
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
    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.updating",
      {},
      "mark-vulnerability-reviewed-11"
    )
    expect(toastMocks.success).toHaveBeenCalledWith(
      "vulnerabilities.reviewSuccess",
      undefined,
      "mark-vulnerability-reviewed-11"
    )
    expect(toastMocks.dismiss).not.toHaveBeenCalled()
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

    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.updating",
      {},
      "mark-vulnerability-reviewed-11"
    )
    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "vulnerabilities.reviewError",
      "mark-vulnerability-reviewed-11"
    )
    expect(toastMocks.dismiss).not.toHaveBeenCalled()
  })

  it("标记未审阅成功时原位更新 loading 并失效列表", async () => {
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
    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.updating",
      {},
      "mark-vulnerability-unreviewed-12"
    )
    expect(toastMocks.success).toHaveBeenCalledWith(
      "vulnerabilities.unreviewSuccess",
      undefined,
      "mark-vulnerability-unreviewed-12"
    )
    expect(toastMocks.dismiss).not.toHaveBeenCalled()
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

    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.updating",
      {},
      "mark-vulnerability-unreviewed-12"
    )
    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "vulnerabilities.unreviewError",
      "mark-vulnerability-unreviewed-12"
    )
    expect(toastMocks.dismiss).not.toHaveBeenCalled()
  })

  it("bulk 未审阅成功时原位更新 loading 并失效列表", async () => {
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
    expect(toastMocks.loading).toHaveBeenCalledWith(
      "vulnerabilities.bulkUnreviewLoading",
      { count: 2 },
      "vulnerabilities-bulk-unreview"
    )
    expect(toastMocks.success).toHaveBeenCalledWith(
      "vulnerabilities.bulkUnreviewSuccess",
      { count: 2 },
      "vulnerabilities-bulk-unreview"
    )
    expect(toastMocks.dismiss).not.toHaveBeenCalled()
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

    expect(toastMocks.loading).toHaveBeenCalledWith(
      "vulnerabilities.bulkUnreviewLoading",
      { count: 2 },
      "vulnerabilities-bulk-unreview"
    )
    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "vulnerabilities.bulkUnreviewError",
      "vulnerabilities-bulk-unreview"
    )
    expect(toastMocks.dismiss).not.toHaveBeenCalled()
  })
})
