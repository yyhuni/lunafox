import { renderHook } from "@testing-library/react"
import { describe, expect, it } from "vitest"

import { useCursorPaginationScopeChange } from "@/components/shared/data-table/use-cursor-pagination-scope"

describe("useCursorPaginationScopeChange", () => {
  it("only reports the render where the cursor query scope changes", () => {
    const { result, rerender } = renderHook(
      ({ scopeKey }: { scopeKey: string }) => useCursorPaginationScopeChange(scopeKey),
      { initialProps: { scopeKey: "target:1" } },
    )

    expect(result.current).toBe(false)

    rerender({ scopeKey: "target:2" })
    expect(result.current).toBe(true)

    rerender({ scopeKey: "target:2" })
    expect(result.current).toBe(false)
  })
})
