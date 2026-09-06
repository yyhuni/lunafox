import { describe, expect, it } from "vitest"
import { readdirSync, readFileSync, statSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ui/checkbox.tsx"), "utf8")

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

describe("checkbox contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from \"@/components/icons\"")
    expect(source).toContain("radius-control-subtle")
    expect(source).not.toContain("from \"@carbon/icons-react\"")
  })

  it("uses Base UI Checkbox as the shared primitive baseline", () => {
    expect(source).toContain('from "@base-ui/react/checkbox"')
    expect(source).toContain("CheckboxPrimitive.Root")
    expect(source).toContain("CheckboxPrimitive.Indicator")
    expect(source).toContain("indeterminate")
    expect(source).toContain("React.ComponentPropsWithoutRef<typeof CheckboxPrimitive.Root>")
    expect(source).toContain("data-[checked]:border-interaction-accent")
    expect(source).toContain("data-[checked]:bg-interaction-accent")
    expect(source).toContain("data-[checked]:text-interaction-accent-foreground")
    expect(source).toContain("data-[indeterminate]:border-interaction-accent")
    expect(source).toContain("data-[indeterminate]:bg-interaction-accent")
    expect(source).toContain("data-[indeterminate]:text-interaction-accent-foreground")
    expect(source).toContain("data-[unchecked]:hidden")
    expect(source).not.toContain('checked?: boolean | "indeterminate"')
    expect(source).not.toContain('checked === "indeterminate"')
    expect(source).not.toContain("data-[state=checked]")
    expect(source).not.toContain("data-[state=indeterminate]")
    expect(source).not.toContain("@radix-ui/react-checkbox")
  })

  it("requires callers to pass indeterminate separately instead of overloading checked", () => {
    const files = collectSourceFiles(path.resolve(process.cwd(), "components"))
    const legacyCallers = files.flatMap((filePath) => {
      const fileSource = readFileSync(filePath, "utf8")
      return fileSource.includes('&& "indeterminate"')
        ? [path.relative(process.cwd(), filePath)]
        : []
    })

    expect(legacyCallers).toEqual([])
  })
})
