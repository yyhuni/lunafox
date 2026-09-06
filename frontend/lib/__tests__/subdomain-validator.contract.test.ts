import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"
import { SubdomainValidator } from "../subdomain-validator"

const source = readFileSync(path.resolve(process.cwd(), "lib/subdomain-validator.ts"), "utf8")

describe("subdomain-validator contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("/**")
  })

  it("preserves original input line numbers for batch validation feedback", () => {
    const parsed = SubdomainValidator.parseLines("api.example.com\n\n-bad.example.com")
    const result = SubdomainValidator.validateBatch(parsed)

    expect(result.invalidItems[0]).toMatchObject({
      lineNumber: 3,
      errorCode: "format",
    })
  })

  it("classifies duplicates as advisory items with original line ownership", () => {
    const parsed = SubdomainValidator.parseLines([
      "api.example.com",
      "www.example.com",
      "api.example.com",
    ].join("\n"))

    const result = SubdomainValidator.validateBatch(parsed)

    expect(result.duplicateItems).toEqual([
      expect.objectContaining({
        lineNumber: 3,
        duplicateOfLine: 1,
      }),
    ])
    expect(result.validCount).toBe(2)
    expect(result.duplicateCount).toBe(1)
    expect(result.invalidCount).toBe(0)
  })

  it("blocks entries beyond the supported batch size", () => {
    const parsed = SubdomainValidator.parseLines(Array.from({ length: 5001 }, (_, index) => `host-${index}.example.com`).join("\n"))
    const result = SubdomainValidator.validateBatch(parsed)

    expect(result.invalidItems.at(-1)).toMatchObject({
      lineNumber: 5001,
      errorCode: "too_many_lines",
    })
  })
})
