import * as React from "react"
import { render, screen, waitFor } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

vi.mock("@/components/app-sidebar", () => ({
  AppSidebar: () => <div data-testid="app-sidebar" />,
}))

vi.mock("@/components/unified-header", () => ({
  UnifiedHeader: () => <div data-testid="unified-header" />,
}))

vi.mock("@/components/ui/sonner", () => ({
  Toaster: () => <div data-testid="global-toaster" />,
}))

vi.mock("@/components/shared/loading/app-shell-warmup", () => ({
  AppShellWarmup: ({
    owner,
    delayMs = 120,
  }: {
    owner: string
    delayMs?: number
  }) => {
    const [visible, setVisible] = React.useState(delayMs <= 0)

    React.useEffect(() => {
      if (delayMs <= 0) {
        setVisible(true)
        return
      }

      setVisible(false)
      const timer = window.setTimeout(() => {
        setVisible(true)
      }, delayMs)
      return () => window.clearTimeout(timer)
    }, [delayMs])

    return visible ? <div data-testid={`app-shell-warmup:${owner}`}>warmup</div> : null
  },
}))

vi.mock("@/components/shared/loading/app-warmup-loader", () => ({
  AppWarmupLoader: ({
    owner,
    intent,
  }: {
    owner: string
    intent: string
  }) => <div data-testid={`app-warmup-loader:${owner}`} data-intent={intent}>auth-pending</div>,
}))

vi.mock("@/components/auth/protected-auth-layout", () => ({
  ProtectedAuthLayout: ({
    children,
  }: {
    children: React.ReactNode
  }) => <>{children}</>,
}))

vi.mock("next/navigation", () => ({
  usePathname: () => "/targets/",
  useSearchParams: () => new URLSearchParams("tab=summary"),
  useRouter: () => ({
    replace: mockRouterReplace,
  }),
}))

const mockUseAuth = vi.fn()
const mockRouterReplace = vi.fn()

vi.mock("@/hooks/use-auth", () => ({
  useAuth: () => mockUseAuth(),
}))

vi.mock("next-intl", () => ({
  useLocale: () => "zh",
}))

describe("AuthLayout", () => {
  beforeEach(() => {
    mockRouterReplace.mockReset()
    mockUseAuth.mockReturnValue({
      data: { authenticated: true },
      isLoading: false,
    })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it("不再在业务壳子内部挂载 Toaster", async () => {
    const { AuthLayout } = await import("@/components/auth/auth-layout")

    render(
      <AuthLayout>
        <div data-testid="page-content">page</div>
      </AuthLayout>
    )

    await waitFor(() => {
      expect(screen.getByTestId("page-content")).toBeInTheDocument()
    })

    expect(screen.queryByTestId("global-toaster")).not.toBeInTheDocument()
  })

  it("isLoading 时由 boot handoff blocker 持有首屏，不出现 AUTH 卡片或 AppShellWarmup", async () => {
    mockUseAuth.mockReturnValue({
      data: { authenticated: true },
      isLoading: true,
    })

    const { AuthLayout } = await import("@/components/auth/auth-layout")

    render(
      <AuthLayout>
        <div data-testid="page-content">page</div>
      </AuthLayout>
    )

    // Boot-level pending, not a second visible auth loading scene.
    expect(screen.getByTestId("auth-boot-handoff-blocker")).toHaveAttribute(
      "data-boot-handoff-pending",
      "true"
    )
    expect(screen.queryByTestId("app-warmup-loader:auth-layout-pending")).not.toBeInTheDocument()
    expect(screen.queryByTestId("page-content")).not.toBeInTheDocument()
    // No protected shell warmup (sidebar/header)
    expect(screen.queryByTestId("app-shell-warmup:auth-layout-warmup")).not.toBeInTheDocument()
    expect(screen.queryByTestId("app-shell-warmup:auth-layout-suspense-fallback")).not.toBeInTheDocument()
  })

  it("authenticated 后才挂载 ProtectedAuthLayout 并渲染 children", async () => {
    mockUseAuth.mockReturnValue({
      data: { authenticated: true },
      isLoading: false,
    })

    const { AuthLayout } = await import("@/components/auth/auth-layout")

    render(
      <AuthLayout>
        <div data-testid="page-content">page</div>
      </AuthLayout>
    )

    await waitFor(() => {
      expect(screen.getByTestId("page-content")).toBeInTheDocument()
    })

    // No auth-level pending visible
    expect(screen.queryByTestId("app-warmup-loader:auth-layout-pending")).not.toBeInTheDocument()
  })

  it("未登录访问受保护页时会带着 returnTo 跳转到 canonical 登录页", async () => {
    mockUseAuth.mockReturnValue({
      data: { authenticated: false },
      isLoading: false,
    })

    const { AuthLayout } = await import("@/components/auth/auth-layout")

    render(
      <AuthLayout>
        <div data-testid="page-content">page</div>
      </AuthLayout>
    )

    await waitFor(() => {
      expect(mockRouterReplace).toHaveBeenCalledWith(
        "/login/?returnTo=%2Ftargets%2F%3Ftab%3Dsummary"
      )
    })

    // No protected shell, no page content — the global boot layer keeps ownership.
    expect(screen.queryByTestId("page-content")).not.toBeInTheDocument()
    expect(screen.queryByTestId("app-shell-warmup:auth-layout-warmup")).not.toBeInTheDocument()
    expect(screen.queryByTestId("app-shell-warmup:auth-layout-suspense-fallback")).not.toBeInTheDocument()
    expect(screen.getByTestId("auth-boot-handoff-blocker")).toHaveAttribute(
      "data-boot-handoff-pending",
      "true"
    )
    expect(screen.queryByTestId("app-warmup-loader:auth-layout-pending")).not.toBeInTheDocument()
  })
})
