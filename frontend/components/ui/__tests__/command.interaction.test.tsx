import { describe, expect, it, vi } from "vitest"
import { fireEvent, render, screen } from "@testing-library/react"

import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "../command"

describe("Command", () => {
  it("calls onSelect with the selected item value", () => {
    const onSelect = vi.fn()

    render(
      <Command>
        <CommandList>
          <CommandGroup>
            <CommandItem value="alpha" onSelect={onSelect}>Alpha</CommandItem>
          </CommandGroup>
        </CommandList>
      </Command>
    )

    fireEvent.click(screen.getByText("Alpha"))

    expect(onSelect).toHaveBeenCalledWith("alpha")
  })

  it("filters items through CommandInput and shows the empty state when none remain", () => {
    render(
      <Command>
        <CommandInput aria-label="Search commands" />
        <CommandList>
          <CommandEmpty>No results</CommandEmpty>
          <CommandGroup>
            <CommandItem value="alpha">Alpha</CommandItem>
            <CommandItem value="beta">Beta</CommandItem>
          </CommandGroup>
        </CommandList>
      </Command>
    )

    fireEvent.change(screen.getByLabelText("Search commands"), { target: { value: "bet" } })

    expect(screen.getByText("Alpha")).not.toBeVisible()
    expect(screen.getByText("Beta")).toBeVisible()
    expect(screen.getByText("No results")).not.toBeVisible()

    fireEvent.change(screen.getByLabelText("Search commands"), { target: { value: "zzz" } })

    expect(screen.getByText("Alpha")).not.toBeVisible()
    expect(screen.getByText("Beta")).not.toBeVisible()
    expect(screen.getByText("No results")).toBeVisible()
  })

  it("selects the active item with keyboard navigation", () => {
    const onAlpha = vi.fn()
    const onBeta = vi.fn()

    render(
      <Command>
        <CommandList>
          <CommandGroup>
            <CommandItem value="alpha" onSelect={onAlpha}>Alpha</CommandItem>
            <CommandItem value="beta" onSelect={onBeta}>Beta</CommandItem>
          </CommandGroup>
        </CommandList>
      </Command>
    )

    const root = screen.getByRole("combobox")
    fireEvent.keyDown(root, { key: "ArrowDown" })
    fireEvent.keyDown(root, { key: "ArrowDown" })
    fireEvent.keyDown(root, { key: "Enter" })

    expect(onAlpha).not.toHaveBeenCalled()
    expect(onBeta).toHaveBeenCalledWith("beta")
  })
})
