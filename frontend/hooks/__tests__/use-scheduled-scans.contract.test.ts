import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "hooks/use-scheduled-scans.ts"), "utf8")

describe("use-scheduled-scans contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useScheduledScans")
    expect(source).toContain("from '@tanstack/react-query'")
  })
})

describe("use-scheduled-scans auto-refresh contract", () => {
  it("polls the list and overview queries on the shared 15-second focus-gated cadence", () => {
    expect(source).toContain("export const SCHEDULED_SCAN_POLL_INTERVAL_MS = 15000")
    expect(source).toContain("refetchInterval: SCHEDULED_SCAN_POLL_INTERVAL_MS")
    // Both the list and the overview summary must declare the interval.
    expect(source.match(/refetchInterval: SCHEDULED_SCAN_POLL_INTERVAL_MS/g)).toHaveLength(2)
  })
})
