import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/loading/spinner.tsx"), "utf8")

describe("spinner contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from \"@/components/icons\"")
  })

  it("exposes the shared loading accessibility and reduced-motion hook points", () => {
    expect(source).toContain("data-slot=\"spinner\"")
    expect(source).toContain("role=\"status\"")
    expect(source).toContain("aria-live=\"polite\"")
    expect(source).toContain("aria-label")
    expect(source).toContain("loading-spinner")
  })

  it("exposes a refresh-specific spinner for refresh actions that should not swap icons", () => {
    expect(source).toContain("function RefreshSpinner")
    expect(source).toContain("IconRefresh")
    expect(source).toContain('data-slot="refresh-spinner"')
    expect(source).toContain('aria-hidden="true"')
    expect(source).toContain("export { Spinner, RefreshSpinner }")
  })
})
