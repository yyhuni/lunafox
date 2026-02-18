import { act, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import {
  useDeleteWordlist,
  useUpdateWordlistContent,
  useUploadWordlist,
  useWordlistContent,
  useWordlists,
  wordlistKeys,
} from "@/hooks/use-wordlists"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import { createTestQueryClient } from "@/test/utils/test-query-client"

const mockGetWordlists = vi.fn()
const mockUploadWordlist = vi.fn()
const mockDeleteWordlist = vi.fn()
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

  it("默认分页参数会请求 page=1,pageSize=10", async () => {
    mockGetWordlists.mockResolvedValue({
      results: [],
      total: 0,
      page: 1,
      pageSize: 10,
      totalPages: 0,
    })

    const { result } = renderHookWithProviders(() => useWordlists())

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(mockGetWordlists).toHaveBeenCalledWith(1, 10)
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
    expect(mockToastMessages.dismiss).toHaveBeenCalledWith("delete-wordlist-42")
    expect(mockToastMessages.success).toHaveBeenCalledWith("toast.wordlist.delete.success")
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
        name: "common.txt",
        description: "test",
        file,
      })
    })

    expect(mockUploadWordlist).toHaveBeenCalledWith({
      name: "common.txt",
      description: "test",
      file,
    })
    expect(mockToastMessages.loading).toHaveBeenCalledWith(
      "common.status.uploading",
      {},
      "upload-wordlist"
    )
    expect(mockToastMessages.dismiss).toHaveBeenCalledWith("upload-wordlist")
    expect(mockToastMessages.success).toHaveBeenCalledWith("toast.wordlist.upload.success")
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
          name: "deny.txt",
          file,
        })
      ).rejects.toBeDefined()
    })

    expect(mockToastMessages.loading).toHaveBeenCalledWith(
      "common.status.uploading",
      {},
      "upload-wordlist"
    )
    expect(mockToastMessages.dismiss).toHaveBeenCalledWith("upload-wordlist")
    expect(mockToastMessages.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.wordlist.upload.error"
    )
  })

  it("更新词表内容成功时触发 toast 与内容失效", async () => {
    mockUpdateWordlistContent.mockResolvedValue({
      id: 3,
      name: "common.txt",
      description: "test",
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
    expect(mockToastMessages.dismiss).toHaveBeenCalledWith("update-wordlist-content")
    expect(mockToastMessages.success).toHaveBeenCalledWith("toast.wordlist.update.success")
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: wordlistKeys.all })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: wordlistKeys.content(3) })
  })

  it("更新词表内容失败时保留错误码映射与 fallback key", async () => {
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
    expect(mockToastMessages.dismiss).toHaveBeenCalledWith("update-wordlist-content")
    expect(mockToastMessages.errorFromCode).toHaveBeenCalledWith(
      "BAD_REQUEST",
      "toast.wordlist.update.error"
    )
  })

  it("id 为 null 时不会请求内容", async () => {
    renderHookWithProviders(() => useWordlistContent(null))

    await waitFor(() => {
      expect(mockGetWordlistContent).not.toHaveBeenCalled()
    })
  })
})
