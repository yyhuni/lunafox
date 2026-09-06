import { fireEvent, render, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { SharedCompactPagination } from "@/components/shared/data-table/pagination"

describe("SharedCompactPagination cursor mode", () => {
  it("keeps the result summary and exposes only enabled adjacent navigation", () => {
    const onPreviousPage = vi.fn()
    const onNextPage = vi.fn()
    const onFirstPage = vi.fn()

    const { container, rerender } = render(
      <SharedCompactPagination
        mode="cursor"
        pageSize={20}
        canFirstPage={false}
        canPreviousPage={false}
        canNextPage
        onFirstPage={onFirstPage}
        onPreviousPage={onPreviousPage}
        onNextPage={onNextPage}
        onPageSizeChange={vi.fn()}
        summary="42 results"
      />,
    )

    expect(screen.getByText("42 results")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "previous" })).toBeDisabled()
    expect(screen.getByRole("button", { name: "next" })).toBeEnabled()
    expect(screen.getByRole("button", { name: "first" })).toBeDisabled()
    expect(screen.queryByRole("button", { name: "last" })).not.toBeInTheDocument()
    expect(container.querySelector('[aria-current="page"]')).toBeNull()

    fireEvent.click(screen.getByRole("button", { name: "previous" }))
    fireEvent.click(screen.getByRole("button", { name: "next" }))
    fireEvent.click(screen.getByRole("button", { name: "first" }))

    expect(onFirstPage).not.toHaveBeenCalled()
    expect(onPreviousPage).not.toHaveBeenCalled()
    expect(onNextPage).toHaveBeenCalledTimes(1)

    rerender(
      <SharedCompactPagination
        mode="cursor"
        pageSize={20}
        canFirstPage
        canPreviousPage
        canNextPage={false}
        onFirstPage={onFirstPage}
        onPreviousPage={onPreviousPage}
        onNextPage={onNextPage}
        onPageSizeChange={vi.fn()}
        summary="42 results"
      />,
    )

    expect(screen.getByRole("button", { name: "previous" })).toBeEnabled()
    expect(screen.getByRole("button", { name: "next" })).toBeDisabled()
    expect(screen.getByRole("button", { name: "first" })).toBeEnabled()

    fireEvent.click(screen.getByRole("button", { name: "previous" }))
    fireEvent.click(screen.getByRole("button", { name: "next" }))
    fireEvent.click(screen.getByRole("button", { name: "first" }))

    expect(onFirstPage).toHaveBeenCalledTimes(1)
    expect(onPreviousPage).toHaveBeenCalledTimes(1)
    expect(onNextPage).toHaveBeenCalledTimes(1)
  })
})
