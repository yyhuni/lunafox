import { act } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { useUpdateGlobalBlacklist } from "@/hooks/use-global-blacklist"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import { createTestQueryClient } from "@/test/utils/test-query-client"

const blacklistServiceMocks = vi.hoisted(() => ({
  getGlobalBlacklist: vi.fn(),
  updateGlobalBlacklist: vi.fn(),
}))

const helperToastMocks = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  errorFromCode: vi.fn(),
  loading: vi.fn(),
  warning: vi.fn(),
  dismiss: vi.fn(),
}))

const sonnerToastMocks = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
}))

vi.mock("@/services/global-blacklist.service", () => ({
  getGlobalBlacklist: blacklistServiceMocks.getGlobalBlacklist,
  updateGlobalBlacklist: blacklistServiceMocks.updateGlobalBlacklist,
}))

vi.mock("@/lib/toast-helpers", () => ({
  useToastMessages: () => helperToastMocks,
}))

vi.mock("next-intl", () => ({
  useTranslations: (namespace?: string) => (key: string) =>
    namespace ? `${namespace}.${key}` : key,
}))

vi.mock("sonner", () => ({
  toast: sonnerToastMocks,
}))

describe("use-global-blacklist mutation", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("更新成功时使用既有 i18n 成功提示并失效缓存", async () => {
    blacklistServiceMocks.updateGlobalBlacklist.mockResolvedValue({
      patterns: ["*.internal"],
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useUpdateGlobalBlacklist(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync({
        patterns: ["*.internal"],
      })
    })

    expect(sonnerToastMocks.success).toHaveBeenCalledWith(
      "pages.settings.blacklist.toast.saveSuccess"
    )
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["global-blacklist"],
    })
  })

  it("更新失败时使用既有 i18n 错误提示且不触发默认 errorFromCode", async () => {
    blacklistServiceMocks.updateGlobalBlacklist.mockRejectedValue(new Error("network"))

    const { result } = renderHookWithProviders(() => useUpdateGlobalBlacklist())

    await act(async () => {
      await expect(
        result.current.mutateAsync({
          patterns: ["*.fail"],
        })
      ).rejects.toBeDefined()
    })

    expect(sonnerToastMocks.error).toHaveBeenCalledWith(
      "pages.settings.blacklist.toast.saveError"
    )
    expect(helperToastMocks.errorFromCode).not.toHaveBeenCalled()
  })
})
