import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "lib/auth-runtime.mjs"), "utf8")

describe("auth-runtime contract", () => {
  it("exports shared route and smoke auth helpers", () => {
    expect(source).toContain("export const SMOKE_AUTH_SESSION_KEY")
    expect(source).toContain("export const SMOKE_LOCALE_COOKIE_KEY")
    expect(source).toContain("export function isPublicPathname")
    expect(source).toContain("export function buildLoginPath")
    expect(source).toContain("export function buildReturnTo")
    expect(source).toContain("export function buildLoginRedirectPath")
    expect(source).toContain("export function buildLoginRedirectPathForLocation")
    expect(source).toContain("export function createSmokeLocaleCookie")
    expect(source).toContain("export async function primeSmokeLocaleCookie")
    expect(source).toContain("export function primeSmokeAuthSession")
  })
})
