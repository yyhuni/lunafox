import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "hooks/use-nuclei-pocs.ts"), "utf8")

describe("use-nuclei-pocs contract", () => {
  it("keeps polling owned by the mounted catalog page at one-second cadence", () => {
    expect(source).toContain("refetchInterval")
    expect(source).toContain("if (!taskName) return false")
    expect(source).not.toContain("open || !taskName")
    expect(source).toContain("return 1000")
    expect(source).toContain('status === 404 || status === 410')
    expect(source).toContain('TERMINAL_TASK_STATES = new Set(["SUCCEEDED", "FAILED"])')
  })

  it("invalidates source, list, tag options, and detail projections only after success", () => {
    expect(source).toContain('task.state !== "SUCCEEDED"')
    expect(source).toContain("nucleiPocKeys.source()")
    expect(source).toContain("nucleiPocKeys.lists()")
    expect(source).toContain('nucleiPocKeys.filterOptions("tags")')
    expect(source).toContain("nucleiPocKeys.details()")
    expect(source).toContain("UpdateNucleiPocRequest")
    expect(source).not.toContain("cancelNucleiPoc")
  })

  it("exposes the complete-catalog tag options as a dedicated query", () => {
    expect(source).toContain("NucleiPocFilterOptionField")
    expect(source).toContain("filterOptions: (field: NucleiPocFilterOptionField)")
    expect(source).toContain("export function useNucleiPocFilterOptions")
    expect(source).toContain("getNucleiPocFilterOptions(field)")
  })
})
