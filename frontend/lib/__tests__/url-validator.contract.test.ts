import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"
import { URLValidator } from "../url-validator"

const source = readFileSync(path.resolve(process.cwd(), "lib/url-validator.ts"), "utf8")

describe("url-validator contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("/**")
  })

  it("derives an authority host without using a general URL parser", () => {
    expect(URLValidator.validate("http://bad-url").isValid).toBe(false)
    expect(URLValidator.validate("https://example.com").isValid).toBe(true)
    expect(URLValidator.validate("https://192.168.1.1").isValid).toBe(true)
    expect(URLValidator.validate("https://example.com/%zz?x=%00#fragment ")).toMatchObject({
      isValid: true,
      url: "https://example.com/%zz?x=%00#fragment ",
    })
  })

  it("preserves raw non-empty lines and accepts Windows record delimiters", () => {
    const rawURL = "HTTPS://Example.com:443/path?payload=%00%zz#fragment "
    expect(URLValidator.parseLines(`${rawURL}\r\n\n`)).toEqual([
      { url: rawURL, lineNumber: 1 },
    ])

    const result = URLValidator.validateBatch([rawURL], "example.com", "domain")
    expect(result.urls).toEqual([rawURL])
    expect(result.mismatchedCount).toBe(0)
  })

  it("rejects actual controls and values beyond the shared byte ceiling", () => {
    const prefix = "https://example.com/"
    const exactLimit = `${prefix}${"a".repeat(2000 - new TextEncoder().encode(prefix).byteLength)}`
    const beyondLimit = `${exactLimit}a`

    expect(URLValidator.validate("https://example.com/\rInjected")).toMatchObject({
      isValid: false,
      errorCode: "control",
    })
    expect(URLValidator.validate(exactLimit).isValid).toBe(true)
    expect(URLValidator.validate(beyondLimit)).toMatchObject({
      isValid: false,
      errorCode: "too_long",
    })
  })

  it("preserves original input line numbers for batch validation feedback", () => {
    const parsed = URLValidator.parseLines("https://example.com\n\nftp://example.com")
    const result = URLValidator.validateBatch(parsed)

    expect(result.invalidItems[0]).toMatchObject({
      lineNumber: 3,
      errorCode: "protocol",
    })
  })

  it("classifies duplicates as advisory items and mismatches as blocking items", () => {
    const parsed = URLValidator.parseLines([
      "https://example.com",
      "https://www.example.com",
      "https://example.com",
      "https://other-target.com",
    ].join("\n"))

    const result = URLValidator.validateBatch(parsed, "example.com", "domain")

    expect(result.duplicateItems).toEqual([
      expect.objectContaining({
        lineNumber: 3,
        duplicateOfLine: 1,
      }),
    ])
    expect(result.mismatchedItems).toEqual([
      expect.objectContaining({
        lineNumber: 4,
      }),
    ])
    expect(result.invalidCount).toBe(0)
    expect(result.duplicateCount).toBe(1)
    expect(result.mismatchedCount).toBe(1)
  })

  it("blocks entries beyond the supported batch size", { timeout: 15000 }, () => {
    const parsed = URLValidator.parseLines(Array.from({ length: 5001 }, (_, index) => `ftp://host-${index}.example.com`).join("\n"))
    const result = URLValidator.validateBatch(parsed)

    expect(result.invalidItems.at(-1)).toMatchObject({
      lineNumber: 5001,
      errorCode: "too_many_lines",
    })
  })
})
