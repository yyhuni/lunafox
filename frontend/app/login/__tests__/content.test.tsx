import * as React from "react"
import { act, render, screen } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const mockReplaceWithRouteProgress = vi.fn()
const mockPrefetchOverviewData = vi.fn()
const mockMutateAsync = vi.fn()
const mockSetQueryData = vi.fn()
const mockRouterPrefetch = vi.fn()
const mockRouterReplace = vi.fn()
const mockUseAuth = vi.fn()
const mockSearchParams = vi.fn(() => new URLSearchParams())
const scheduledTimeouts: Array<() => void> = []
let setTimeoutSpy: ReturnType<typeof vi.spyOn> | null = null
let autoVisualReady = true

vi.mock("next/navigation", () => ({
  useRouter: () => ({
    prefetch: mockRouterPrefetch,
    replace: mockRouterReplace,
  }),
  useSearchParams: () => mockSearchParams(),
}))

vi.mock("next-intl", () => ({
  useLocale: () => "zh",
  useTranslations: () => (key: string) => key,
}))

vi.mock("@tanstack/react-query", () => ({
  useQueryClient: () => ({
    setQueryData: mockSetQueryData,
  }),
}))

vi.mock("@/components/route-progress", () => ({
  replaceWithRouteProgress: (...args: unknown[]) => mockReplaceWithRouteProgress(...args),
}))

vi.mock("@/components/auth/visual-split-login", () => ({
  VisualSplitLogin: ({
    onLogin,
    onVisualReady,
  }: {
    onLogin: (username: string, password: string) => Promise<void>
    onVisualReady?: () => void
  }) => {
    React.useEffect(() => {
      if (autoVisualReady) {
        onVisualReady?.()
      }
    }, [onVisualReady])

    return (
      <button type="button" onClick={() => void onLogin("demo", "secret")}>
        login
      </button>
    )
  },
}))

vi.mock("@/components/shared/loading/content-reveal", () => ({
  ContentReveal: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}))

vi.mock("@/hooks/use-overview-prefetch", () => ({
  usePrefetchOverviewData: () => mockPrefetchOverviewData,
}))

vi.mock("@/hooks/use-auth", () => ({
  useAuth: () => mockUseAuth(),
  useLogin: () => ({
    mutateAsync: mockMutateAsync,
    isPending: false,
  }),
}))

describe("LoginPage content", () => {
  beforeEach(() => {
    mockReplaceWithRouteProgress.mockReset()
    mockPrefetchOverviewData.mockReset()
    mockMutateAsync.mockReset()
    mockSetQueryData.mockReset()
    mockRouterPrefetch.mockReset()
    mockRouterReplace.mockReset()
    mockSearchParams.mockReset()

    mockPrefetchOverviewData.mockResolvedValue(undefined)
    mockMutateAsync.mockResolvedValue({
      user: {
        id: 1,
        username: "demo",
      },
    })
    mockSearchParams.mockReturnValue(new URLSearchParams())
    autoVisualReady = true
    scheduledTimeouts.length = 0
    setTimeoutSpy = vi.spyOn(window, "setTimeout").mockImplementation((handler) => {
      if (typeof handler === "function") {
        scheduledTimeouts.push(handler)
      }
      return {} as ReturnType<typeof window.setTimeout>
    })
  })

  afterEach(() => {
    setTimeoutSpy?.mockRestore()
    setTimeoutSpy = null
  })

  async function flushAsyncWork() {
    await act(async () => {
      await Promise.resolve()
      await Promise.resolve()
      await Promise.resolve()
      await Promise.resolve()
    })
  }

  it("已登录时优先跳回 canonical returnTo", async () => {
    mockUseAuth.mockReturnValue({
      data: { authenticated: true },
      isLoading: false,
    })
    mockSearchParams.mockReturnValue(
      new URLSearchParams("returnTo=%2Ftargets%2F42%2Foverview%2F%3Ftab%3Dsummary")
    )

    const { default: LoginPage } = await import("@/app/login/content")

    render(<LoginPage />)

    await flushAsyncWork()

    expect(mockPrefetchOverviewData).toHaveBeenCalled()

    expect(scheduledTimeouts).toHaveLength(1)

    await act(async () => {
      scheduledTimeouts[0]()
    })

    expect(mockReplaceWithRouteProgress).toHaveBeenCalledWith(
      {
        prefetch: mockRouterPrefetch,
        replace: mockRouterReplace,
      },
      "/targets/42/overview/?tab=summary"
    )
  })

  it("认证状态未决时仍保留登录 surface，避免客户端 token 读取造成 hydration mismatch", async () => {
    autoVisualReady = false
    mockUseAuth.mockReturnValue({
      data: undefined,
      isLoading: true,
    })

    const { default: LoginPage } = await import("@/app/login/content")

    const { container } = render(<LoginPage />)

    expect(screen.getByRole("button", { name: "login" })).toBeInTheDocument()
    expect(container.firstElementChild).toHaveAttribute("data-boot-handoff-pending", "true")
  })

  it("客户端检测到已登录后隐藏登录 surface 并进入跳转遮罩", async () => {
    mockUseAuth.mockReturnValue({
      data: { authenticated: true },
      isLoading: false,
    })

    const { default: LoginPage } = await import("@/app/login/content")

    const { container } = render(<LoginPage />)

    await flushAsyncWork()

    expect(screen.queryByRole("button", { name: "login" })).not.toBeInTheDocument()
    expect(container.firstElementChild).not.toHaveAttribute("data-boot-handoff-pending")
  })

  it("登录完成后会拒绝外部 returnTo 并回到 canonical overview", async () => {
    mockUseAuth.mockReturnValue({
      data: { authenticated: false },
      isLoading: false,
    })
    mockSearchParams.mockReturnValue(
      new URLSearchParams("returnTo=https%3A%2F%2Fevil.example%2Fphish")
    )

    const { default: LoginPage } = await import("@/app/login/content")

    render(<LoginPage />)

    await act(async () => {
      screen.getByRole("button", { name: "login" }).click()
    })

    await flushAsyncWork()

    expect(mockRouterPrefetch).toHaveBeenCalledWith("/overview/")

    expect(scheduledTimeouts).toHaveLength(1)

    await act(async () => {
      scheduledTimeouts[0]()
    })

    expect(mockSetQueryData).toHaveBeenCalledWith(["auth", "me"], {
      authenticated: true,
      user: {
        id: 1,
        username: "demo",
      },
    })
    expect(mockReplaceWithRouteProgress).toHaveBeenCalledWith(
      {
        prefetch: mockRouterPrefetch,
        replace: mockRouterReplace,
      },
      "/overview/"
    )
  })
})
