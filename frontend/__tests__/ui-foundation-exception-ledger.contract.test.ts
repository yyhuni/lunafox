import { describe, expect, it } from "vitest"
import { existsSync, readFileSync } from "node:fs"
import path from "node:path"

const ledgerPath = path.resolve(process.cwd(), "foundation-exceptions.json")
const inventoryPath = path.resolve(process.cwd(), "foundation-inventory.md")

const allowedStatuses = ["keep", "narrow", "remove", "deferred", "excluded"]
const allowedClasses = [
  "theme-token-owner",
  "typography-role-owner",
  "chart-rendering",
  "terminal-log",
  "canvas-webgl",
  "skeleton-geometry",
  "radix-dynamic-var",
  "runtime-calculated-geometry",
  "bounded-hover-motion",
  "brand-provider-color",
  "non-loading-domain-status-indicator",
  "visual-lab-excluded",
  "accessibility-only",
  "legacy-migration",
]

describe("UI foundation exception ledger", () => {
  it("exists as the unified exception ledger", () => {
    expect(existsSync(ledgerPath)).toBe(true)
  })

  it("requires reviewable metadata on every exception entry", () => {
    if (!existsSync(ledgerPath)) {
      expect.fail("foundation-exceptions.json is missing")
    }

    const ledger = JSON.parse(readFileSync(ledgerPath, "utf8")) as {
      exceptions: Array<Record<string, unknown>>
    }

    expect(Array.isArray(ledger.exceptions)).toBe(true)
    expect(ledger.exceptions.length).toBeGreaterThan(0)

    for (const entry of ledger.exceptions) {
      expect(typeof entry.id).toBe("string")
      expect(allowedStatuses).toContain(entry.status)
      expect(typeof entry.scope).toBe("string")
      expect(typeof entry.pattern).toBe("string")
      expect(allowedClasses).toContain(entry.class)
      expect(typeof entry.owner).toBe("string")
      expect(typeof entry.reason).toBe("string")
      expect(typeof entry.reviewTrigger).toBe("string")
      expect(typeof entry.recoveryPath).toBe("string")
    }
  })

  it("does not keep migration debt statuses after final convergence", () => {
    if (!existsSync(ledgerPath)) {
      expect.fail("foundation-exceptions.json is missing")
    }

    const ledger = JSON.parse(readFileSync(ledgerPath, "utf8")) as {
      exceptions: Array<Record<string, unknown>>
    }

    for (const entry of ledger.exceptions) {
      expect(["keep", "narrow", "excluded"]).toContain(entry.status)
    }
  })

  it("records the approved initial baseline classes", () => {
    if (!existsSync(ledgerPath)) {
      expect.fail("foundation-exceptions.json is missing")
    }

    const source = readFileSync(ledgerPath, "utf8")

    expect(source).toContain("theme-token-owner")
    expect(source).toContain("chart-rendering")
    expect(source).toContain("terminal-log")
    expect(source).toContain("skeleton-geometry")
    expect(source).toContain("radix-dynamic-var")
    expect(source).toContain("runtime-calculated-geometry")
    expect(source).toContain("visual-lab-excluded")
  })

  it("keeps a readable baseline inventory index next to the ledger", () => {
    expect(existsSync(inventoryPath)).toBe(true)

    const source = readFileSync(inventoryPath, "utf8")

    expect(source).toContain("foundation-exceptions.json")
    for (const status of allowedStatuses) {
      expect(source).toContain(status)
    }
  })
})
