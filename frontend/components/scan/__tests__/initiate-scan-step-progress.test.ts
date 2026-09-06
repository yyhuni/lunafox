import { describe, expect, it } from "vitest"

import { getScanStepProgress } from "@/components/scan/initiate-scan-dialog-sections"

describe("getScanStepProgress", () => {
  it("maps the normal scan step position to its two-step progress", () => {
    expect(getScanStepProgress(1, 2)).toBe(50)
    expect(getScanStepProgress(2, 2)).toBe(100)
  })

  it("maps the quick scan step position to its three-step progress", () => {
    expect(getScanStepProgress(1, 3)).toBe(33)
    expect(getScanStepProgress(2, 3)).toBe(67)
    expect(getScanStepProgress(3, 3)).toBe(100)
  })
})
