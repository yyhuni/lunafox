import { describe, expect, it } from "vitest"

import { MAX_TARGET_BATCH_SIZE, TargetValidator } from "@/lib/target-validator"

describe("TargetValidator", () => {
  it("exports the server-aligned batch maximum", () => {
    expect(MAX_TARGET_BATCH_SIZE).toBe(5000)
  })

  it("preserves original line numbers when blank lines precede targets", () => {
    const parsed = TargetValidator.parseLines("\n\nbad target\nexample.com")

    expect(parsed).toEqual([
      { target: "bad target", lineNumber: 3 },
      { target: "example.com", lineNumber: 4 },
    ])

    const results = TargetValidator.validateTargetBatch(parsed)

    expect(results[0]).toMatchObject({
      index: 0,
      lineNumber: 3,
      originalTarget: "bad target",
      isValid: false,
    })
    expect(results[1]).toMatchObject({
      index: 1,
      lineNumber: 4,
      originalTarget: "example.com",
      isValid: true,
    })
  })
})
