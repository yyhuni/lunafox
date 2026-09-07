import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "lib/browser-storage.ts"), "utf8")

describe("browser-storage contract", () => {
  it("exports shared browser storage helpers", () => {
    expect(source).toContain("export function isStorageAvailable")
    expect(source).toContain("export function readStorageValue")
    expect(source).toContain("export function writeStorageValue")
    expect(source).toContain("export function removeStorageValue")
    expect(source).toContain("export function readJsonStorage")
    expect(source).toContain("export function writeJsonStorage")
    expect(source).toContain("export function readSessionValue")
    expect(source).toContain("export function writeSessionValue")
  })
})
