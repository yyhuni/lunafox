import { fireEvent, render, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { AppErrorState } from "@/components/shared/feedback/app-error-state"
import { createAppError } from "@/lib/errors/app-error"

const back = vi.fn()

vi.mock("next/navigation", () => ({
  useRouter: () => ({ back }),
}))

vi.mock("next-intl", () => ({
  useLocale: () => "zh",
}))

describe("AppErrorState", () => {
  it("renders the compact variant with localized semantic copy and retries through the query owner", () => {
    const onRetry = vi.fn()
    const rawTransportMessage = "Request failed with status code 503"

    render(
      <AppErrorState
        error={createAppError("service-unavailable", {
          cause: new Error(rawTransportMessage),
        })}
        variant="section"
        onRetry={onRetry}
      />
    )

    const alert = screen.getByRole("alert")
    expect(alert).toHaveAttribute("data-app-error-kind", "service-unavailable")
    expect(alert).toHaveAttribute("data-app-error-variant", "section")
    expect(screen.getByText("服务暂时不可用")).toBeInTheDocument()
    expect(screen.queryByText(rawTransportMessage)).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole("button", { name: "重试" }))
    expect(onRetry).toHaveBeenCalledTimes(1)
  })

  it("uses safe back navigation instead of retry for non-recoverable failures", () => {
    back.mockClear()

    render(
      <AppErrorState
        error={createAppError("permission-denied")}
      />
    )

    expect(screen.queryByRole("button", { name: "重试" })).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole("button", { name: "返回" }))
    expect(back).toHaveBeenCalledTimes(1)
  })
})
