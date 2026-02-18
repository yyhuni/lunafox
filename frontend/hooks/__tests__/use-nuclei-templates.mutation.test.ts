import { act } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import {
  useRefreshNucleiTemplates,
  useSaveNucleiTemplate,
  useUploadNucleiTemplate,
} from "@/hooks/use-nuclei-templates"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import { createTestQueryClient } from "@/test/utils/test-query-client"

const nucleiServiceMocks = vi.hoisted(() => ({
  getNucleiTemplateTree: vi.fn(),
  getNucleiTemplateContent: vi.fn(),
  refreshNucleiTemplates: vi.fn(),
  saveNucleiTemplate: vi.fn(),
  uploadNucleiTemplate: vi.fn(),
}))

const toastMocks = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  errorFromCode: vi.fn(),
  loading: vi.fn(),
  warning: vi.fn(),
  dismiss: vi.fn(),
}))

vi.mock("@/services/nuclei.service", () => ({
  getNucleiTemplateTree: nucleiServiceMocks.getNucleiTemplateTree,
  getNucleiTemplateContent: nucleiServiceMocks.getNucleiTemplateContent,
  refreshNucleiTemplates: nucleiServiceMocks.refreshNucleiTemplates,
  saveNucleiTemplate: nucleiServiceMocks.saveNucleiTemplate,
  uploadNucleiTemplate: nucleiServiceMocks.uploadNucleiTemplate,
}))

vi.mock("@/lib/toast-helpers", () => ({
  useToastMessages: () => toastMocks,
}))

describe("use-nuclei-templates mutation", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("刷新模板成功时保留成功提示并失效 tree 缓存", async () => {
    nucleiServiceMocks.refreshNucleiTemplates.mockResolvedValue(undefined)

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useRefreshNucleiTemplates(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync(undefined)
    })

    expect(toastMocks.loading).toHaveBeenCalledWith(
      "toast.nucleiTemplate.refresh.loading",
      {},
      "refresh-nuclei-templates"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("refresh-nuclei-templates")
    expect(nucleiServiceMocks.refreshNucleiTemplates).toHaveBeenCalled()
    expect(toastMocks.success).toHaveBeenCalledWith("toast.nucleiTemplate.refresh.success")
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["nuclei", "templates", "tree"],
    })
  })

  it("刷新模板失败时保留错误码映射与 fallback key", async () => {
    nucleiServiceMocks.refreshNucleiTemplates.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useRefreshNucleiTemplates())

    await act(async () => {
      await expect(result.current.mutateAsync(undefined)).rejects.toBeDefined()
    })

    expect(toastMocks.loading).toHaveBeenCalledWith(
      "toast.nucleiTemplate.refresh.loading",
      {},
      "refresh-nuclei-templates"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("refresh-nuclei-templates")
    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.nucleiTemplate.refresh.error"
    )
  })

  it("保存模板成功时按 path 失效 content 缓存", async () => {
    nucleiServiceMocks.saveNucleiTemplate.mockResolvedValue(undefined)

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useSaveNucleiTemplate(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        path: "http/test.yaml",
        content: "id: test-template",
      })
    })

    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.actions.saving",
      {},
      "save-nuclei-template"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("save-nuclei-template")
    expect(toastMocks.success).toHaveBeenCalledWith("toast.nucleiTemplate.save.success")
    expect(nucleiServiceMocks.saveNucleiTemplate).toHaveBeenCalledWith({
      path: "http/test.yaml",
      content: "id: test-template",
    })
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["nuclei", "templates", "content", "http/test.yaml"],
    })
  })

  it("保存模板失败时保留错误码映射与 fallback key", async () => {
    nucleiServiceMocks.saveNucleiTemplate.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useSaveNucleiTemplate())

    await act(async () => {
      await expect(
        result.current.mutateAsync({
          path: "http/blocked.yaml",
          content: "id: blocked",
        })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.actions.saving",
      {},
      "save-nuclei-template"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("save-nuclei-template")
    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.nucleiTemplate.save.error"
    )
  })

  it("上传模板成功时保留 loading/success 并失效 tree 缓存", async () => {
    nucleiServiceMocks.uploadNucleiTemplate.mockResolvedValue(undefined)

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useUploadNucleiTemplate(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        scope: "custom",
        file: new File(["id: ok"], "ok.yaml", { type: "application/x-yaml" }),
      })
    })

    expect(toastMocks.loading).toHaveBeenCalledWith(
      "common.status.uploading",
      {},
      "upload-nuclei-template"
    )
    expect(toastMocks.dismiss).toHaveBeenCalledWith("upload-nuclei-template")
    expect(toastMocks.success).toHaveBeenCalledWith("toast.nucleiTemplate.upload.success")
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["nuclei", "templates", "tree"],
    })
  })

  it("上传模板失败时保留错误码映射与 fallback key", async () => {
    nucleiServiceMocks.uploadNucleiTemplate.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FILE_TOO_LARGE",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useUploadNucleiTemplate())

    await act(async () => {
      await expect(
        result.current.mutateAsync({
          scope: "custom",
          file: new File(["id: x"], "x.yaml", { type: "application/x-yaml" }),
        })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FILE_TOO_LARGE",
      "toast.nucleiTemplate.upload.error"
    )
  })
})
