import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ui/switch.tsx"), "utf8")

describe("switch contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
    expect(source).toContain("radius-round")
  })

  it("uses Base UI Switch as the shared primitive baseline", () => {
    expect(source).toContain('from "@base-ui/react/switch"')
    expect(source).toContain("SwitchPrimitives.Root")
    expect(source).toContain("SwitchPrimitives.Thumb")
    expect(source).toContain("data-[checked]:bg-interaction-accent")
    expect(source).toContain("data-[unchecked]:bg-input")
    expect(source).toContain("data-[checked]:translate-x-4")
    expect(source).toContain("data-[unchecked]:translate-x-0")
    expect(source).not.toContain("data-[state=checked]")
    expect(source).not.toContain("data-[state=unchecked]")
    expect(source).not.toContain("@radix-ui/react-switch")
  })

  it("renders as a native button so drawer gestures do not swallow switch clicks", () => {
    expect(source).toContain("nativeButton")
    expect(source).toContain('render={<button type="button" />}')
    expect(source).not.toContain("internalInputRef")
    expect(source).not.toContain("input.click()")
  })
})
