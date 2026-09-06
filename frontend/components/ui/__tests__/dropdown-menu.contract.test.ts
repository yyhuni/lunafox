import { describe, expect, it } from "vitest"
import { readdirSync, readFileSync, statSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ui/dropdown-menu.tsx"), "utf8")

function collectSourceFiles(directory: string): string[] {
  return readdirSync(directory).flatMap((entry) => {
    const fullPath = path.join(directory, entry)
    const stats = statSync(fullPath)

    if (stats.isDirectory()) {
      if (entry === "__tests__") return []
      return collectSourceFiles(fullPath)
    }

    return /\.(tsx|ts)$/.test(entry) ? [fullPath] : []
  })
}

describe("dropdown-menu contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("reuses shared overlay surface tokens for dropdown content shells", () => {
    expect(source).toContain("from \"@/lib/ui/overlay-styles\"")
    expect(source).toContain("floatingSurfaceClassName")
    expect(source).toContain("floatingContentMotionClassName")
  })

  it("owns a shared content-fit width mode for short action menus", () => {
    expect(source).toContain('width = "default"')
    expect(source).toContain('width?: "default" | "content-fit"')
    expect(source).toContain('width === "default" && "min-w-[8rem]"')
    expect(source).toContain('width === "content-fit" && "w-max min-w-(--anchor-width)"')
  })

  it("uses Base UI as the dropdown menu primitive backend", () => {
    expect(source).toContain('from "@base-ui/react/menu"')
    expect(source).not.toContain("@radix-ui/react-dropdown-menu")
    expect(source).toContain("DropdownMenuPrimitive.Positioner")
    expect(source).toContain("DropdownMenuPrimitive.Popup")
  })

  it("keeps the positioned dropdown layer above Sheet content", () => {
    expect(source).toContain('data-slot="dropdown-menu-positioner"')
    expect(source).toContain('className="z-50"')
  })

  it("uses Base UI render directly instead of Radix-style asChild compatibility", () => {
    expect(source).not.toContain("asChild")
    expect(source).not.toContain("render={asChild ? child : render}")
  })

  it("uses Base UI geometry variables instead of Radix dropdown variables", () => {
    expect(source).toContain("--available-height")
    expect(source).toContain("--transform-origin")
    expect(source).not.toContain("--radix-dropdown")
  })

  it("keeps item-like menu rows on the shared control radius", () => {
    for (const functionName of [
      "DropdownMenuItem",
      "DropdownMenuCheckboxItem",
      "DropdownMenuRadioItem",
      "DropdownMenuSubTrigger",
    ] as const) {
      expect(source).toMatch(
        new RegExp(
          `function ${functionName}\\([\\s\\S]*?className=\\{cn\\(\\s*"radius-control(?!-subtle)`
        )
      )
    }
  })

  it("hard-cuts Base UI checkbox and radio menu items to close on selection", () => {
    expect(source).toContain("type DropdownMenuCheckboxItemProps = Omit<")
    expect(source).toContain("type DropdownMenuRadioItemProps = Omit<")
    expect(source).toContain('"closeOnClick"')

    for (const functionName of ["DropdownMenuCheckboxItem", "DropdownMenuRadioItem"] as const) {
      expect(source).toMatch(
        new RegExp(
          `function ${functionName}\\([\\s\\S]*?<DropdownMenuPrimitive\\.(?:CheckboxItem|RadioItem)[\\s\\S]*?closeOnClick=\\{true\\}`
        )
      )

      expect(source).toMatch(
        new RegExp(
          `function ${functionName}\\([\\s\\S]*?<DropdownMenuPrimitive\\.(?:CheckboxItem|RadioItem)[\\s\\S]*?\\{\\.\\.\\.props\\}[\\s\\S]*?closeOnClick=\\{true\\}`
        )
      )
    }

    expect(source).not.toContain("closeOnClick = true")
    expect(source).not.toContain("closeOnClick={closeOnClick}")
  })

  it("hard-cuts dropdown menu callers to Base UI popup-open state attributes", () => {
    const files = collectSourceFiles(path.resolve(process.cwd(), "components"))
    const legacyCallers = files.flatMap((filePath) => {
      const fileSource = readFileSync(filePath, "utf8")
      return fileSource.includes("data-[state=open]") || fileSource.includes("group-data-[state=open]")
        ? [path.relative(process.cwd(), filePath)]
        : []
    })

    expect(legacyCallers).toEqual([])
  })
})
