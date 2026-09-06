import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/data-table/row-actions.tsx"), "utf8")

describe("row-actions contract", () => {
  it("keeps shared tooltip ownership and dense action layout in the helper layer", () => {
    expect(source).toContain("export function DenseRowActionOwner")
    expect(source).toContain("TooltipProvider delay={300}")
    expect(source).not.toContain("TooltipProvider delayDuration={300}")
    expect(source).toContain('data-row-click-exempt="true"')
    expect(source).toContain("justify-end gap-1")
  })

  it("keeps quiet copy fallback and desktop hover reveal in one shared helper", () => {
    expect(source).toContain("export function QuietCopyButton")
    expect(source).toContain('from "@/components/shared/feedback/copy-button"')
    expect(source).toContain("<CopyButton")
    expect(source).toContain("DENSE_ROW_ACTION_HOVER_REVEAL_CLASSNAME")
    expect(source).toContain("hideUntilHover")
    expect(source).toContain("toastId={toastId}")
  })

  it("keeps DenseRowActionOwner focused on layout while shared menus live in menu-owners", () => {
    expect(source).toContain("export function DenseRowActionOwner")
    expect(source).not.toContain("DropdownMenuTrigger")
    expect(source).not.toContain("MoreHorizontal")
  })

  it("keeps tooltip-open row actions on the same visual state as hover", () => {
    expect(source).toContain(
      "data-[popup-open]:bg-primary/10 data-[popup-open]:text-primary dark:data-[popup-open]:bg-primary/20"
    )
  })
})
