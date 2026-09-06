import { render, screen } from "@testing-library/react"
import { AxiosError } from "axios"
import { describe, expect, it, vi } from "vitest"

import TargetLayout from "@/app/targets/[id]/layout"

const mockUseTarget = vi.fn()

vi.mock("next/navigation", () => ({
  useParams: () => ({ id: "999999" }),
  usePathname: () => "/targets/999999/overview/",
  useRouter: () => ({
    back: vi.fn(),
  }),
}))

vi.mock("next-intl", () => ({
  useLocale: () => "zh",
  useTranslations: (namespace: string) => {
    if (namespace !== "pages.targetDetail") {
      return (key: string) => key
    }

    return (key: string, values?: Record<string, string>) => {
      if (key === "error.title") return "加载失败"
      if (key === "error.message") return "获取目标数据时出现错误"
      if (key === "notFound.title") return "目标不存在"
      if (key === "notFound.message") return `未找到ID为 ${values?.id ?? ""} 的目标`
      if (key === "breadcrumb.targetDetail") return "目标详情"
      return key
    }
  },
}))

vi.mock("@/hooks/use-targets", () => ({
  useTarget: (...args: unknown[]) => mockUseTarget(...args),
}))

vi.mock("@/components/target/target-overview-sections", () => ({
  TargetOverviewLoadingState: () => <div data-slot="target-overview-loading-state" />,
}))

describe("TargetLayout error state", () => {
  it("在 metadata 未决时持续挂载同一个详情 owner，数据就绪后才挂载 resolved layout", () => {
    mockUseTarget.mockReturnValue({
      data: undefined,
      isLoading: true,
      error: null,
    })

    const { container, rerender } = render(
      <TargetLayout>
        <div>child workspace</div>
      </TargetLayout>
    )

    const initialOwner = container.querySelector('[data-loading-owner="target-detail-shell"]')
    expect(initialOwner).toHaveAttribute("data-loading-phase", "loading")
    expect(initialOwner).toHaveAttribute("data-loading-layer", "workspace")
    expect(container.querySelector('[data-boot-handoff-pending="true"]')).not.toBeInTheDocument()
    expect(screen.queryByText("child workspace")).not.toBeInTheDocument()

    mockUseTarget.mockReturnValue({
      data: { name: "example.com", summary: {} },
      isLoading: false,
      error: null,
    })

    rerender(
      <TargetLayout>
        <div>child workspace</div>
      </TargetLayout>
    )

    expect(container.querySelector('[data-loading-owner="target-detail-shell"]')).toBe(initialOwner)
    expect(screen.getByText("child workspace")).toBeInTheDocument()
  })

  it("在目标详情 404 时渲染资源不存在文案，而不是通用加载失败文案", async () => {
    mockUseTarget.mockReturnValue({
      data: undefined,
      isLoading: false,
      error: new AxiosError(
        "Request failed with status code 404",
        "ERR_BAD_REQUEST",
        undefined,
        undefined,
        {
          data: { error: "Target not found" },
          status: 404,
          statusText: "Not Found",
          headers: {},
          config: {} as never,
        }
      ),
    })

    render(
      <TargetLayout>
        <div>content</div>
      </TargetLayout>
    )

    expect(screen.getByText("目标不存在")).toBeInTheDocument()
    expect(screen.getByText("未找到ID为 999999 的目标")).toBeInTheDocument()
    expect(screen.queryByText("加载失败")).not.toBeInTheDocument()
    expect(screen.queryByText("获取目标数据时出现错误")).not.toBeInTheDocument()
  })

  it("在目标详情 503 时沿用共享服务不可用文案，而不是页面自定义加载失败文案", async () => {
    mockUseTarget.mockReturnValue({
      data: undefined,
      isLoading: false,
      error: new AxiosError(
        "Request failed with status code 503",
        "ERR_BAD_RESPONSE",
        undefined,
        undefined,
        {
          data: { error: "Service unavailable" },
          status: 503,
          statusText: "Service Unavailable",
          headers: {},
          config: {} as never,
        }
      ),
    })

    render(
      <TargetLayout>
        <div>content</div>
      </TargetLayout>
    )

    expect(screen.getByText("服务暂时不可用")).toBeInTheDocument()
    expect(screen.getByText("服务暂时不可用，请稍后重试。")).toBeInTheDocument()
    expect(screen.queryByText("加载失败")).not.toBeInTheDocument()
    expect(screen.queryByText("获取目标数据时出现错误")).not.toBeInTheDocument()
  })
})
