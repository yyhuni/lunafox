import { act, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import {
  useCompleteWordlistCatalog,
  useDeleteWordlist,
  useUpdateWordlistContent,
  useUpdateWordlistMetadata,
  useUploadWordlist,
  useWordlistContent,
  useWordlistTags,
  useWordlists,
  wordlistKeys,
} from "@/hooks/use-wordlists"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import { createTestQueryClient } from "@/test/utils/test-query-client"

const mockGetWordlists = vi.fn()
const mockUploadWordlist = vi.fn()
const mockDeleteWordlist = vi.fn()
const mockGetWordlistTags = vi.fn()
const mockUpdateWordlistMetadata = vi.fn()
const mockGetWordlistContent = vi.fn()
const mockUpdateWordlistContent = vi.fn()

const mockToastMessages = {
  success: vi.fn(),
  error: vi.fn(),
  errorFromCode: vi.fn(),
  loading: vi.fn(),
  warning: vi.fn(),
  dismiss: vi.fn(),
}

vi.mock("@/services/wordlist.service", () => ({
  getWordlists: (...args: unknown[]) => mockGetWordlists(...args),
  uploadWordlist: (...args: unknown[]) => mockUploadWordlist(...args),
  deleteWordlist: (...args: unknown[]) => mockDeleteWordlist(...args),
  getWordlistTags: (...args: unknown[]) => mockGetWordlistTags(...args),
  updateWordlistMetadata: (...args: unknown[]) => mockUpdateWordlistMetadata(...args),
  getWordlistContent: (...args: unknown[]) => mockGetWordlistContent(...args),
  updateWordlistContent: (...args: unknown[]) => mockUpdateWordlistContent(...args),
}))

vi.mock("@/lib/toast-helpers", () => ({
  useToastMessages: () => mockToastMessages,
}))

describe("use-wordlists", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("默认列表参数会请求 canonical pageSize 和 orderBy", async () => {
    mockGetWordlists.mockResolvedValue({
      results: [],
      totalSize: 0,
    })

    const { result } = renderHookWithProviders(() => useWordlists())

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(mockGetWordlists).toHaveBeenCalledWith({
      pageSize: 10,
      pageToken: undefined,
      filter: undefined,
      orderBy: undefined,
    })
  })

  it("列表查询只透传 canonical filter 和 orderBy", async () => {
    mockGetWordlists.mockResolvedValue({
      results: [],
      totalSize: 0,
    })

    const { result } = renderHookWithProviders(() =>
      useWordlists({
        pageSize: 20,
        pageToken: "mock-page:2",
        filter: '(tags=="目录扫描" || tags=="API 路径") && fileName="api"',
        orderBy: "fileName",
      })
    )

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(mockGetWordlists).toHaveBeenCalledWith({
      pageSize: 20,
      pageToken: "mock-page:2",
      filter: '(tags=="目录扫描" || tags=="API 路径") && fileName="api"',
      orderBy: "fileName",
    })
  })

  it("完整 Catalog 查询会追完分页并按 canonical name 去重", async () => {
    mockGetWordlists.mockImplementation(({ pageToken }: { pageToken?: string }) => {
      if (!pageToken) {
        return Promise.resolve({
          results: [wordlistFixture(1, "alpha.txt"), wordlistFixture(2, "duplicate.txt")],
          nextPageToken: "page-2",
          totalSize: 4,
        })
      }
      return Promise.resolve({
        results: [wordlistFixture(3, "duplicate.txt"), wordlistFixture(4, "omega.txt")],
        totalSize: 4,
      })
    })

    const { result } = renderHookWithProviders(() => useCompleteWordlistCatalog())

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data?.map((wordlist) => wordlist.name)).toEqual([
      "wordlists/1",
      "wordlists/2",
      "wordlists/3",
      "wordlists/4",
    ])
    expect(mockGetWordlists).toHaveBeenNthCalledWith(1, {
      pageSize: 100,
      pageToken: undefined,
      filter: undefined,
      orderBy: "fileName",
    })
    expect(mockGetWordlists).toHaveBeenNthCalledWith(2, {
      pageSize: 100,
      pageToken: "page-2",
      filter: undefined,
      orderBy: "fileName",
    })
  })

  it("完整 Catalog 查询允许完整空结果", async () => {
    mockGetWordlists.mockResolvedValue({ results: [], totalSize: 0 })
    const { result } = renderHookWithProviders(() => useCompleteWordlistCatalog())
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual([])
  })

  it("完整 Catalog 后续页失败时不暴露部分结果", async () => {
    mockGetWordlists.mockImplementation(({ pageToken }: { pageToken?: string }) => {
      if (!pageToken) {
        return Promise.resolve({
          results: [wordlistFixture(1, "alpha.txt")],
          nextPageToken: "page-2",
          totalSize: 2,
        })
      }
      return Promise.reject(new Error("later page failed"))
    })
    const { result } = renderHookWithProviders(() => useCompleteWordlistCatalog())
    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.data).toBeUndefined()
  })

  it("标签摘要查询会转成 wordlistTags filter", async () => {
    mockGetWordlistTags.mockResolvedValue({
      results: [],
      totalSize: 0,
    })

    const { result } = renderHookWithProviders(() => useWordlistTags({ pageSize: 20, search: "fuzz" }))

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(mockGetWordlistTags).toHaveBeenCalledWith({
      pageSize: 20,
      pageToken: undefined,
      filter: "fuzz",
    })
  })

  it("删除词表会触发 toast 与 query 失效", async () => {
    mockDeleteWordlist.mockResolvedValue(undefined)
    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")

    const { result } = renderHookWithProviders(() => useDeleteWordlist(), { queryClient })

    await act(async () => {
      await result.current.mutateAsync(42)
    })

    expect(mockDeleteWordlist).toHaveBeenCalledWith(42)
    expect(mockToastMessages.loading).toHaveBeenCalledWith(
      "common.status.deleting",
      {},
      "delete-wordlist-42"
    )
    expect(mockToastMessages.dismiss).not.toHaveBeenCalledWith("delete-wordlist-42")
    expect(mockToastMessages.success).toHaveBeenCalledWith(
      "toast.wordlist.delete.success",
      undefined,
      "delete-wordlist-42"
    )
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: wordlistKeys.all })
  })

  it("上传词表成功时触发 toast 与列表失效", async () => {
    mockUploadWordlist.mockResolvedValue({
      id: 9,
      name: "common.txt",
      description: "test",
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useUploadWordlist(), {
      queryClient,
    })

    const file = { name: "common.txt" } as File

    await act(async () => {
      await result.current.mutateAsync({
        description: "test",
        file,
      })
    })

    expect(mockUploadWordlist).toHaveBeenCalledWith({
      description: "test",
      file,
    })
    expect(mockToastMessages.loading).toHaveBeenCalledWith(
      "common.status.uploading",
      {},
      "upload-wordlist"
    )
    expect(mockToastMessages.dismiss).not.toHaveBeenCalledWith("upload-wordlist")
    expect(mockToastMessages.success).toHaveBeenCalledWith(
      "toast.wordlist.upload.success",
      undefined,
      "upload-wordlist"
    )
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: wordlistKeys.all })
  })

  it("上传词表失败时保留错误码映射与 fallback key", async () => {
    mockUploadWordlist.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useUploadWordlist())

    const file = { name: "deny.txt" } as File

    await act(async () => {
      await expect(
        result.current.mutateAsync({
          file,
        })
      ).rejects.toBeDefined()
    })

    expect(mockToastMessages.loading).toHaveBeenCalledWith(
      "common.status.uploading",
      {},
      "upload-wordlist"
    )
    expect(mockToastMessages.dismiss).not.toHaveBeenCalledWith("upload-wordlist")
    expect(mockToastMessages.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.wordlist.upload.error",
      "upload-wordlist"
    )
  })

  it("更新词表内容成功时触发 toast 与内容失效", async () => {
    mockUpdateWordlistContent.mockResolvedValue({
      name: "wordlists/3/text",
      content: "root\nadmin\n",
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useUpdateWordlistContent(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        id: 3,
        content: "root\nadmin\n",
      })
    })

    expect(mockUpdateWordlistContent).toHaveBeenCalledWith(3, "root\nadmin\n")
    expect(mockToastMessages.loading).toHaveBeenCalledWith(
      "common.actions.saving",
      {},
      "update-wordlist-content"
    )
    expect(mockToastMessages.dismiss).not.toHaveBeenCalledWith("update-wordlist-content")
    expect(mockToastMessages.success).toHaveBeenCalledWith(
      "toast.wordlist.update.success",
      undefined,
      "update-wordlist-content"
    )
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: wordlistKeys.all })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: wordlistKeys.content(3) })
  })

  it("更新词表内容失败时显示通用更新失败 toast", async () => {
    mockUpdateWordlistContent.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "BAD_REQUEST",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useUpdateWordlistContent())

    await act(async () => {
      await expect(
        result.current.mutateAsync({
          id: 3,
          content: "root\nadmin\n",
        })
      ).rejects.toBeDefined()
    })

    expect(mockToastMessages.loading).toHaveBeenCalledWith(
      "common.actions.saving",
      {},
      "update-wordlist-content"
    )
    expect(mockToastMessages.dismiss).not.toHaveBeenCalledWith("update-wordlist-content")
    expect(mockToastMessages.error).toHaveBeenCalledWith(
      "toast.wordlist.update.error",
      undefined,
      "update-wordlist-content"
    )
  })

  it("更新词表内容遇到超限错误时显示可执行提示", async () => {
    mockUpdateWordlistContent.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "BAD_REQUEST",
            message: "File too large for online editing (max 5MB)",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useUpdateWordlistContent())

    await act(async () => {
      await expect(
        result.current.mutateAsync({
          id: 3,
          content: "root\nadmin\n",
        })
      ).rejects.toBeDefined()
    })

    expect(mockToastMessages.error).toHaveBeenCalledWith(
      "toast.wordlist.update.oversized",
      undefined,
      "update-wordlist-content"
    )
  })

  it("更新词表元信息成功时触发 toast 与列表失效", async () => {
    mockUpdateWordlistMetadata.mockResolvedValue({
      id: 3,
      name: "wordlists/3",
      fileName: "common.txt",
      tags: ["fuzz"],
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useUpdateWordlistMetadata(), { queryClient })

    await act(async () => {
      await result.current.mutateAsync({
        id: 3,
        name: "wordlists/3",
        description: "updated",
        tags: ["fuzz"],
        updateMask: "description,tags",
      })
    })

    expect(mockUpdateWordlistMetadata).toHaveBeenCalledWith({
      id: 3,
      name: "wordlists/3",
      description: "updated",
      tags: ["fuzz"],
      updateMask: "description,tags",
    })
    expect(mockToastMessages.success).toHaveBeenCalledWith(
      "toast.wordlist.update.success",
      undefined,
      "update-wordlist-metadata-3"
    )
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: wordlistKeys.all })
  })

  it("id 为 null 时不会请求内容", async () => {
    renderHookWithProviders(() => useWordlistContent(null))

    await waitFor(() => {
      expect(mockGetWordlistContent).not.toHaveBeenCalled()
    })
  })
})

function wordlistFixture(id: number, fileName: string) {
  return {
    id,
    name: `wordlists/${id}`,
    fileName,
    description: "",
    tags: [],
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-01-01T00:00:00Z",
  }
}
