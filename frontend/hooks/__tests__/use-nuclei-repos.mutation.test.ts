import { act } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import {
  nucleiRepoKeys,
  useCreateNucleiRepo,
  useDeleteNucleiRepo,
  useRefreshNucleiRepo,
  useUpdateNucleiRepo,
} from "@/hooks/use-nuclei-repos"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import { createTestQueryClient } from "@/test/utils/test-query-client"

const nucleiRepoServiceMocks = vi.hoisted(() => ({
  listRepos: vi.fn(),
  getRepo: vi.fn(),
  createRepo: vi.fn(),
  updateRepo: vi.fn(),
  deleteRepo: vi.fn(),
  refreshRepo: vi.fn(),
  getTemplateTree: vi.fn(),
  getTemplateContent: vi.fn(),
}))

const toastMocks = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  errorFromCode: vi.fn(),
  loading: vi.fn(),
  warning: vi.fn(),
  dismiss: vi.fn(),
}))

vi.mock("@/services/nuclei-repo.service", () => ({
  nucleiRepoApi: nucleiRepoServiceMocks,
}))

vi.mock("@/lib/toast-helpers", () => ({
  useToastMessages: () => toastMocks,
}))

describe("use-nuclei-repos mutation", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("创建仓库成功时提示成功并失效仓库列表缓存", async () => {
    nucleiRepoServiceMocks.createRepo.mockResolvedValue({
      id: 7,
      name: "nuclei-templates",
      repoUrl: "https://github.com/projectdiscovery/nuclei-templates",
      localPath: "/tmp/nuclei-templates",
      commitHash: null,
      lastSyncedAt: null,
      createdAt: "2026-02-11T11:00:00Z",
      updatedAt: "2026-02-11T11:00:00Z",
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useCreateNucleiRepo(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        name: "nuclei-templates",
        repoUrl: "https://github.com/projectdiscovery/nuclei-templates",
      })
    })

    expect(nucleiRepoServiceMocks.createRepo).toHaveBeenCalledWith(
      {
        name: "nuclei-templates",
        repoUrl: "https://github.com/projectdiscovery/nuclei-templates",
      },
      expect.any(Object)
    )
    expect(toastMocks.success).toHaveBeenCalledWith("toast.nucleiRepo.create.success")
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: nucleiRepoKeys.repos,
    })
  })

  it("创建仓库失败时保留错误码映射与 fallback key", async () => {
    nucleiRepoServiceMocks.createRepo.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "CONFLICT",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useCreateNucleiRepo())

    await act(async () => {
      await expect(
        result.current.mutateAsync({
          name: "nuclei-templates",
          repoUrl: "https://github.com/projectdiscovery/nuclei-templates",
        })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "CONFLICT",
      "toast.nucleiRepo.create.error"
    )
  })

  it("更新仓库成功时提示成功并失效列表与详情缓存", async () => {
    nucleiRepoServiceMocks.updateRepo.mockResolvedValue({
      id: 7,
      name: "nuclei-templates",
      repoUrl: "https://github.com/projectdiscovery/nuclei-templates",
      localPath: "/tmp/nuclei-templates",
      commitHash: null,
      lastSyncedAt: null,
      createdAt: "2026-02-11T11:00:00Z",
      updatedAt: "2026-02-11T11:00:00Z",
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useUpdateNucleiRepo(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        id: 7,
        repoUrl: "https://github.com/projectdiscovery/nuclei-templates",
      })
    })

    expect(nucleiRepoServiceMocks.updateRepo).toHaveBeenCalledWith(7, {
      repoUrl: "https://github.com/projectdiscovery/nuclei-templates",
    })
    expect(toastMocks.success).toHaveBeenCalledWith("toast.nucleiRepo.update.success")
    expect(invalidateSpy).toHaveBeenNthCalledWith(1, {
      queryKey: nucleiRepoKeys.repos,
    })
    expect(invalidateSpy).toHaveBeenNthCalledWith(2, {
      queryKey: nucleiRepoKeys.repo(7),
    })
  })

  it("更新仓库失败时保留错误码映射与 fallback key", async () => {
    nucleiRepoServiceMocks.updateRepo.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useUpdateNucleiRepo())

    await act(async () => {
      await expect(
        result.current.mutateAsync({
          id: 7,
          repoUrl: "https://github.com/projectdiscovery/nuclei-templates",
        })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.nucleiRepo.update.error"
    )
  })

  it("刷新仓库成功时失效列表、详情和目录树缓存", async () => {
    nucleiRepoServiceMocks.refreshRepo.mockResolvedValue({
      message: "refreshed",
      result: {},
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useRefreshNucleiRepo(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync(42)
    })

    expect(nucleiRepoServiceMocks.refreshRepo).toHaveBeenCalledWith(42, expect.any(Object))
    expect(toastMocks.success).toHaveBeenCalledWith("toast.nucleiRepo.sync.success")
    expect(invalidateSpy).toHaveBeenNthCalledWith(1, {
      queryKey: nucleiRepoKeys.repos,
    })
    expect(invalidateSpy).toHaveBeenNthCalledWith(2, {
      queryKey: nucleiRepoKeys.repo(42),
    })
    expect(invalidateSpy).toHaveBeenNthCalledWith(3, {
      queryKey: nucleiRepoKeys.tree(42),
    })
  })

  it("刷新仓库失败时保留错误码映射与 fallback key", async () => {
    nucleiRepoServiceMocks.refreshRepo.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "NETWORK_ERROR",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useRefreshNucleiRepo())

    await act(async () => {
      await expect(result.current.mutateAsync(42)).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "NETWORK_ERROR",
      "toast.nucleiRepo.sync.error"
    )
  })

  it("删除仓库成功时提示成功并失效仓库列表缓存", async () => {
    nucleiRepoServiceMocks.deleteRepo.mockResolvedValue({
      message: "deleted",
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useDeleteNucleiRepo(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync(9)
    })

    expect(nucleiRepoServiceMocks.deleteRepo).toHaveBeenCalledWith(9, expect.any(Object))
    expect(toastMocks.success).toHaveBeenCalledWith("toast.nucleiRepo.delete.success")
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: nucleiRepoKeys.repos,
    })
  })

  it("删除仓库失败时保留错误码映射与 fallback key", async () => {
    nucleiRepoServiceMocks.deleteRepo.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "REPO_IN_USE",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useDeleteNucleiRepo())

    await act(async () => {
      await expect(result.current.mutateAsync(9)).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "REPO_IN_USE",
      "toast.nucleiRepo.delete.error"
    )
    expect(toastMocks.success).not.toHaveBeenCalled()
  })
})
