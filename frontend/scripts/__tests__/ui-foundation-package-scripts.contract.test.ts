import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const packageJson = JSON.parse(
  readFileSync(path.resolve(process.cwd(), "package.json"), "utf8")
) as { scripts: Record<string, string> }

describe("UI foundation package scripts", () => {
  it("keeps local dev scripts on Turbopack entrypoints", () => {
    expect(packageJson.scripts["dev"]).toBe("next dev --turbo")
    expect(packageJson.scripts["dev:mock"]).toBe(
      "NEXT_PUBLIC_USE_MOCK=true next dev --turbo"
    )
    expect(packageJson.scripts["dev:noauth"]).toBe(
      "NEXT_PUBLIC_SKIP_AUTH=true next dev --turbo"
    )
    expect(packageJson.scripts["dev:mock:noauth"]).toBe(
      "NEXT_PUBLIC_USE_MOCK=true NEXT_PUBLIC_SKIP_AUTH=true next dev --turbo"
    )
    expect(packageJson.scripts).not.toHaveProperty("dev:mock:noauth:turbo")
  })

  it("maps every foundation guardrail to verify mode", () => {
    expect(packageJson.scripts["check:ui-boundary"]).toBe(
      "node scripts/check-ui-boundary.mjs --mode verify"
    )
    expect(packageJson.scripts["check:typography-foundation"]).toBe(
      "node scripts/check-typography-foundation.mjs --mode verify"
    )
    expect(packageJson.scripts["check:color-foundation"]).toBe(
      "node scripts/check-color-foundation.mjs --mode verify"
    )
    expect(packageJson.scripts["check:spacing-foundation"]).toBe(
      "node scripts/check-spacing-foundation.mjs --mode verify"
    )
    expect(packageJson.scripts["check:button-foundation"]).toBe(
      "node scripts/check-button-foundation.mjs --mode verify"
    )
    expect(packageJson.scripts["check:component-foundation"]).toBe(
      "node scripts/check-component-foundation.mjs --mode verify"
    )
    expect(packageJson.scripts["check:a11y-foundation"]).toBe(
      "node scripts/check-a11y-foundation.mjs --mode verify"
    )
    expect(packageJson.scripts["check:motion-foundation"]).toBe(
      "node scripts/check-motion-foundation.mjs --mode verify"
    )
    expect(packageJson.scripts["check:loading-foundation"]).toBe(
      "node scripts/check-loading-foundation.mjs --mode verify"
    )
    expect(packageJson.scripts["check:ui-foundation"]).toBe(
      "node scripts/check-ui-foundation.mjs --mode verify"
    )
  })
})
