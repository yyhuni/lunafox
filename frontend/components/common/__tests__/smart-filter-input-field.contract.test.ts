import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/common/smart-filter-input-field.tsx"), "utf8")

describe("smart-filter-input-field contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function SmartFilterInputField")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("keeps compact and standard input heights in the shared field owner", () => {
    expect(source).toContain("toolbarDensity?: DataTableToolbarDensity")
    expect(source).toContain("toolbarDensity = \"compact\"")
    expect(source).toContain('size={isStandardDensity ? "default" : "sm"}')
    expect(source).toContain("inputClassName?: string")
    expect(source).toContain("inputClassName")
    expect(source).not.toContain("className,\n          inputClassName")
    expect(source).toContain("showIcon?: boolean")
    expect(source).toContain("showIcon = true")
    expect(source).toContain('showIcon && "pl-9"')
    expect(source).toContain('showIcon ? "pl-9 pr-3" : "px-3"')
    expect(source).toContain('left-3 top-1/2 size-4')
    expect(source).toContain("onClick?: () => void")
    expect(source).toContain("onClick={onClick}")
    expect(source).toContain("ref?: React.Ref<HTMLDivElement>")
    expect(source).toContain("<div ref={ref}")
    expect(source).not.toContain("!isStandardDensity && \"h-8\"")
  })
})
