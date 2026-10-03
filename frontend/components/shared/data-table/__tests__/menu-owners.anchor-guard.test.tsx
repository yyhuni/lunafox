import * as React from "react"
import { fireEvent, screen, waitFor } from "@testing-library/react"
import { describe, expect, it } from "vitest"

import { DenseRowActionMenu } from "@/components/shared/data-table/menu-owners"
import { DropdownMenuItem } from "@/components/ui/dropdown-menu"
import { renderWithProviders } from "@/test/utils/render-with-providers"

function renderRowMenu() {
  return renderWithProviders(
    <div data-testid="row">
      <DenseRowActionMenu ariaLabel="打开菜单">
        <DropdownMenuItem>编辑任务</DropdownMenuItem>
        <DropdownMenuItem>删除</DropdownMenuItem>
      </DenseRowActionMenu>
    </div>
  )
}

async function openMenu() {
  fireEvent.click(screen.getByRole("button", { name: "打开菜单" }))
  await screen.findByRole("menu")
}

describe("DenseRowActionMenu anchor self-heal", () => {
  it("stays open while the anchor row is visible", async () => {
    renderRowMenu()
    await openMenu()
    expect(await screen.findByText("编辑任务")).toBeVisible()
  })

  it("closes automatically when the anchor row is hidden via display:none", async () => {
    renderRowMenu()
    await openMenu()
    const row = screen.getByTestId("row")
    row.style.display = "none"
    window.dispatchEvent(new Event("resize"))
    await waitFor(() => {
      expect(screen.queryByRole("menu")).toBeNull()
    })
  })

  it("closes automatically when the anchor trigger is detached from the document", async () => {
    renderRowMenu()
    await openMenu()
    const row = screen.getByTestId("row")
    const parent = row.parentNode!
    const next = row.nextSibling
    row.remove()
    window.dispatchEvent(new Event("resize"))
    await waitFor(() => {
      expect(screen.queryByRole("menu")).toBeNull()
    })
    // Restore the React-owned node so cleanup unmounts a connected tree.
    parent.insertBefore(row, next)
  })
})
