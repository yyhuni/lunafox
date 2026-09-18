import { act, render, screen, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

const navigationMocks = vi.hoisted(() => ({
  pathname: "/overview/",
  replace: vi.fn(),
}))
const hookMocks = vi.hoisted(() => ({
  operation: {
    operationId: "operation-id" as string | null,
    isResolving: false,
    isReconnecting: false,
    isError: false,
    isActive: true,
  },
}))

vi.mock("next/navigation", () => ({
  usePathname: () => navigationMocks.pathname,
  useRouter: () => ({ replace: navigationMocks.replace }),
}))

vi.mock("@/components/route-progress", () => ({
  replaceWithRouteProgress: (router: { replace: (path: string) => void }, path: string) => router.replace(path),
}))

vi.mock("@/hooks/use-version", () => ({
  useUpgradeOperation: () => hookMocks.operation,
}))

import { UpgradeRouteBoundary } from "@/components/auth/upgrade-route-boundary"

describe("UpgradeRouteBoundary", () => {
  beforeEach(() => {
    navigationMocks.pathname = "/overview/"
    navigationMocks.replace.mockReset()
    hookMocks.operation = {
      operationId: "operation-id",
      isResolving: false,
      isReconnecting: false,
      isError: false,
      isActive: true,
    }
  })

  it("does not mount ordinary route content and redirects to the status route for an active operation", async () => {
    render(
      <UpgradeRouteBoundary renderProtectedShell={(children) => <div data-testid="shell">{children}</div>}>
        <div data-testid="ordinary-content">ordinary</div>
      </UpgradeRouteBoundary>,
    )

    expect(screen.getByTestId("upgrade-route-boundary")).toBeInTheDocument()
    expect(screen.queryByTestId("ordinary-content")).not.toBeInTheDocument()
    await waitFor(() => expect(navigationMocks.replace).toHaveBeenCalledWith("/system-upgrade/"))
  })

  it("releases the status route without mounting the protected shell", () => {
    navigationMocks.pathname = "/system-upgrade/"
    const { rerender } = render(
      <UpgradeRouteBoundary renderProtectedShell={(children) => <div data-testid="shell">{children}</div>}>
        <div data-testid="status-content">status</div>
      </UpgradeRouteBoundary>,
    )

    expect(screen.getByTestId("status-content")).toBeInTheDocument()
    expect(screen.queryByTestId("shell")).not.toBeInTheDocument()

    act(() => {
      navigationMocks.pathname = "/overview/"
      hookMocks.operation = { operationId: null, isResolving: false, isReconnecting: false, isError: false, isActive: false }
      rerender(
        <UpgradeRouteBoundary renderProtectedShell={(children) => <div data-testid="shell">{children}</div>}>
          <div data-testid="ordinary-content">ordinary</div>
        </UpgradeRouteBoundary>,
      )
    })
    expect(screen.getByTestId("shell")).toBeInTheDocument()
  })

  it("fails closed and redirects when the active-operation query fails", async () => {
    hookMocks.operation = {
      operationId: null,
      isResolving: false,
      isReconnecting: false,
      isError: true,
      isActive: false,
    }

    render(
      <UpgradeRouteBoundary renderProtectedShell={(children) => <div data-testid="shell">{children}</div>}>
        <div data-testid="ordinary-content">ordinary</div>
      </UpgradeRouteBoundary>,
    )

    expect(screen.getByTestId("upgrade-route-boundary")).toBeInTheDocument()
    expect(screen.queryByTestId("ordinary-content")).not.toBeInTheDocument()
    await waitFor(() => expect(navigationMocks.replace).toHaveBeenCalledWith("/system-upgrade/"))
  })

  it("waits for the active-operation query before deciding whether to redirect", async () => {
    hookMocks.operation = {
      operationId: null,
      isResolving: true,
      isReconnecting: false,
      isError: false,
      isActive: false,
    }

    render(
      <UpgradeRouteBoundary renderProtectedShell={(children) => <div data-testid="shell">{children}</div>}>
        <div data-testid="ordinary-content">ordinary</div>
      </UpgradeRouteBoundary>,
    )

    expect(screen.getByTestId("upgrade-route-boundary")).toBeInTheDocument()
    await act(async () => {})
    expect(navigationMocks.replace).not.toHaveBeenCalled()
  })
})
