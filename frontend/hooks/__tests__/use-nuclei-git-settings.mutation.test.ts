import { act } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import {
  nucleiGitKeys,
  useUpdateNucleiGitSettings,
} from "@/hooks/use-nuclei-git-settings"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import { createTestQueryClient } from "@/test/utils/test-query-client"

const nucleiGitServiceMocks = vi.hoisted(() => ({
  getSettings: vi.fn(),
  updateSettings: vi.fn(),
}))

const toastMocks = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  errorFromCode: vi.fn(),
  loading: vi.fn(),
  warning: vi.fn(),
  dismiss: vi.fn(),
}))

vi.mock("@/services/nuclei-git.service", () => ({
  NucleiGitService: nucleiGitServiceMocks,
}))

vi.mock("@/lib/toast-helpers", () => ({
  useToastMessages: () => toastMocks,
}))

describe("use-nuclei-git-settings mutation", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("更新设置成功时保留成功提示和 query 失效", async () => {
    nucleiGitServiceMocks.updateSettings.mockResolvedValue({
      message: "updated",
      settings: {
        repoUrl: "https://github.com/projectdiscovery/nuclei-templates",
        authType: "none",
        authToken: "",
      },
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useUpdateNucleiGitSettings(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        repoUrl: "https://github.com/projectdiscovery/nuclei-templates",
        authType: "none",
        authToken: "",
      })
    })

    expect(toastMocks.success).toHaveBeenCalledWith("toast.nucleiGit.settings.success")
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: nucleiGitKeys.settings,
    })
  })

  it("更新设置失败时保留错误码映射与 fallback key", async () => {
    nucleiGitServiceMocks.updateSettings.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useUpdateNucleiGitSettings())

    await act(async () => {
      await expect(
        result.current.mutateAsync({
          repoUrl: "https://github.com/projectdiscovery/nuclei-templates",
          authType: "token",
          authToken: "bad-token",
        })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "toast.nucleiGit.settings.error"
    )
  })
})
