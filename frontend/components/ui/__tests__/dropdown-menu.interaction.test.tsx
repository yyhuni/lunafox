import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

describe("DropdownMenu interactions", () => {
  it("closes radio-item menus after selecting an option", async () => {
    const handleValueChange = vi.fn()

    render(
      <DropdownMenu>
        <DropdownMenuTrigger>Theme</DropdownMenuTrigger>
        <DropdownMenuContent>
          <DropdownMenuRadioGroup value="default" onValueChange={handleValueChange}>
            <DropdownMenuRadioItem value="default">Default</DropdownMenuRadioItem>
            <DropdownMenuRadioItem value="blue">Modern blue</DropdownMenuRadioItem>
          </DropdownMenuRadioGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    )

    fireEvent.click(screen.getByRole("button", { name: "Theme" }))

    const blueOption = await screen.findByRole("menuitemradio", { name: "Modern blue" })
    fireEvent.click(blueOption)

    expect(handleValueChange).toHaveBeenCalledWith("blue", expect.any(Object))
    await waitFor(() => {
      expect(screen.queryByRole("menuitemradio", { name: "Modern blue" })).not.toBeInTheDocument()
    })
  })

  it("keeps radio-item menus closing even when a caller bypasses the type boundary", async () => {
    const unsafeRadioProps = { closeOnClick: false } as unknown as Parameters<typeof DropdownMenuRadioItem>[0]

    render(
      <DropdownMenu>
        <DropdownMenuTrigger>Unsafe theme</DropdownMenuTrigger>
        <DropdownMenuContent>
          <DropdownMenuRadioGroup value="default" onValueChange={vi.fn()}>
            <DropdownMenuRadioItem {...unsafeRadioProps} value="blue">Modern blue</DropdownMenuRadioItem>
          </DropdownMenuRadioGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    )

    fireEvent.click(screen.getByRole("button", { name: "Unsafe theme" }))

    const blueOption = await screen.findByRole("menuitemradio", { name: "Modern blue" })
    fireEvent.click(blueOption)

    await waitFor(() => {
      expect(screen.queryByRole("menuitemradio", { name: "Modern blue" })).not.toBeInTheDocument()
    })
  })

  it("closes checkbox-item menus after toggling an option", async () => {
    const handleCheckedChange = vi.fn()

    render(
      <DropdownMenu>
        <DropdownMenuTrigger>Columns</DropdownMenuTrigger>
        <DropdownMenuContent>
          <DropdownMenuCheckboxItem checked={false} onCheckedChange={handleCheckedChange}>
            Email
          </DropdownMenuCheckboxItem>
        </DropdownMenuContent>
      </DropdownMenu>
    )

    fireEvent.click(screen.getByRole("button", { name: "Columns" }))

    const emailOption = await screen.findByRole("menuitemcheckbox", { name: "Email" })
    fireEvent.click(emailOption)

    expect(handleCheckedChange).toHaveBeenCalledWith(true, expect.any(Object))
    await waitFor(() => {
      expect(screen.queryByRole("menuitemcheckbox", { name: "Email" })).not.toBeInTheDocument()
    })
  })

  it("keeps checkbox-item menus closing even when a caller bypasses the type boundary", async () => {
    const unsafeCheckboxProps = { closeOnClick: false } as unknown as Parameters<typeof DropdownMenuCheckboxItem>[0]

    render(
      <DropdownMenu>
        <DropdownMenuTrigger>Unsafe columns</DropdownMenuTrigger>
        <DropdownMenuContent>
          <DropdownMenuCheckboxItem {...unsafeCheckboxProps} checked={false} onCheckedChange={vi.fn()}>
            Email
          </DropdownMenuCheckboxItem>
        </DropdownMenuContent>
      </DropdownMenu>
    )

    fireEvent.click(screen.getByRole("button", { name: "Unsafe columns" }))

    const emailOption = await screen.findByRole("menuitemcheckbox", { name: "Email" })
    fireEvent.click(emailOption)

    await waitFor(() => {
      expect(screen.queryByRole("menuitemcheckbox", { name: "Email" })).not.toBeInTheDocument()
    })
  })
})
