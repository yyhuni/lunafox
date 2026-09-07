import { describe, expect, it } from "vitest"
import { existsSync, readFileSync } from "node:fs"
import path from "node:path"

const filePath = path.resolve(process.cwd(), "components/shared/feedback/app-error-state.tsx")

describe("app-error-state contract", () => {
  it("provides one shared business error owner for the approved error kinds", () => {
    expect(existsSync(filePath)).toBe(true)

    const source = readFileSync(filePath, "utf8")

    expect(source).toContain("export function AppErrorState")
    expect(source).toContain("resource-not-found")
    expect(source).toContain("permission-denied")
    expect(source).toContain("rate-limited")
    expect(source).toContain("service-unavailable")
    expect(source).toContain("network-error")
    expect(source).toContain("unexpected-error")
    expect(source).toContain("onRetry")
    expect(source).toContain('variant = "page"')
    expect(source).toContain('data-app-error-variant={variant}')
    expect(source).toContain('role="alert"')
    expect(source).toContain("getStatusToneSurfaceClass")
    expect(source).toContain("getStatusToneTextClass")
    expect(source).toContain("textRole.panelTitle")
    expect(source).not.toContain("error.message")
  })
})
