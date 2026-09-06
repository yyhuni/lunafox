import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "lib/http-status-config.ts"), "utf8")

describe("http-status-config contract", () => {
  it("exports shared HTTP status ownership helpers", () => {
    expect(source).toContain("export function getHttpStatusTone")
    expect(source).toContain("export function getHttpStatusBadgeVariant")
    expect(source).toContain("export function getHttpStatusBadgeClassName")
    expect(source).toContain("export function getHttpStatusTextClassName")
  })
})
