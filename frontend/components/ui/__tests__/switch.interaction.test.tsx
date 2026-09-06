import { fireEvent, render, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { Switch } from "../switch"

describe("switch interactions", () => {
  it("calls onCheckedChange when the shared switch is clicked", () => {
    const handleCheckedChange = vi.fn()

    render(
      <Switch
        aria-label="Advanced mode"
        checked={false}
        onCheckedChange={handleCheckedChange}
      />
    )

    fireEvent.click(screen.getByRole("switch", { name: "Advanced mode" }))

    expect(handleCheckedChange).toHaveBeenCalledWith(true, expect.any(Object))
  })
})
