import { act } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { authKeys, useChangePassword, useLogin, useLogout } from "@/hooks/use-auth"
import { renderHookWithProviders } from "@/test/utils/render-with-providers"
import { createTestQueryClient } from "@/test/utils/test-query-client"

const authServiceMocks = vi.hoisted(() => ({
  login: vi.fn(),
  logout: vi.fn(),
  getMe: vi.fn(),
  changePassword: vi.fn(),
}))

const pushMock = vi.hoisted(() => vi.fn())

const toastMocks = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  errorFromCode: vi.fn(),
  loading: vi.fn(),
  warning: vi.fn(),
  dismiss: vi.fn(),
}))

vi.mock("@/services/auth.service", () => ({
  login: authServiceMocks.login,
  logout: authServiceMocks.logout,
  getMe: authServiceMocks.getMe,
  changePassword: authServiceMocks.changePassword,
}))

vi.mock("@/lib/toast-helpers", () => ({
  useToastMessages: () => toastMocks,
}))

vi.mock("next/navigation", () => ({
  useRouter: () => ({
    push: pushMock,
  }),
}))

vi.mock("next-intl", () => ({
  useLocale: () => "en",
}))

describe("use-auth mutation", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("登出成功时保留提示、失效 auth key 与路由跳转", async () => {
    authServiceMocks.logout.mockResolvedValue({
      message: "ok",
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useLogout(), {
      queryClient,
    })

    await act(async () => {
      await result.current.mutateAsync()
    })

    expect(authServiceMocks.logout).toHaveBeenCalled()
    expect(toastMocks.success).toHaveBeenCalledWith("toast.auth.logout.success")
    expect(pushMock).toHaveBeenCalledWith("/en/login/")
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: authKeys.me,
    })
  })

  it("登录成功时提示成功文案", async () => {
    authServiceMocks.login.mockResolvedValue({
      accessToken: "token",
      refreshToken: "refresh",
      expiresIn: 3600,
      user: { id: 1, username: "admin" },
    })

    const { result } = renderHookWithProviders(() => useLogin())

    await act(async () => {
      await result.current.mutateAsync({
        username: "admin",
        password: "good-password",
      })
    })

    expect(toastMocks.success).toHaveBeenCalledWith("toast.auth.login.success")
  })

  it("登录失败时保留错误码映射与 fallback key", async () => {
    authServiceMocks.login.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "INVALID_CREDENTIALS",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useLogin())

    await act(async () => {
      await expect(
        result.current.mutateAsync({
          username: "admin",
          password: "bad-password",
        })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "INVALID_CREDENTIALS",
      "auth.loginFailed"
    )
  })

  it("登出失败时保留错误码映射与 fallback key", async () => {
    authServiceMocks.logout.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "FORBIDDEN",
          },
        },
      },
    })

    const queryClient = createTestQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHookWithProviders(() => useLogout(), {
      queryClient,
    })

    await act(async () => {
      await expect(result.current.mutateAsync()).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "FORBIDDEN",
      "errors.unknown"
    )
    expect(pushMock).not.toHaveBeenCalled()
    expect(invalidateSpy).not.toHaveBeenCalled()
  })

  it("修改密码成功时提示成功文案", async () => {
    authServiceMocks.changePassword.mockResolvedValue({
      message: "ok",
    })

    const { result } = renderHookWithProviders(() => useChangePassword())

    await act(async () => {
      await result.current.mutateAsync({
        oldPassword: "old",
        newPassword: "new",
      })
    })

    expect(toastMocks.success).toHaveBeenCalledWith("toast.auth.changePassword.success")
  })

  it("修改密码失败时保留错误码映射与 fallback key", async () => {
    authServiceMocks.changePassword.mockRejectedValue({
      response: {
        data: {
          error: {
            code: "WEAK_PASSWORD",
          },
        },
      },
    })

    const { result } = renderHookWithProviders(() => useChangePassword())

    await act(async () => {
      await expect(
        result.current.mutateAsync({
          oldPassword: "old",
          newPassword: "weak",
        })
      ).rejects.toBeDefined()
    })

    expect(toastMocks.errorFromCode).toHaveBeenCalledWith(
      "WEAK_PASSWORD",
      "toast.auth.changePassword.error"
    )
  })
})
