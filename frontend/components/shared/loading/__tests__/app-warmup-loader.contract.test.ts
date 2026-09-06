import { describe, expect, it } from "vitest"
import { existsSync, readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/loading/app-warmup-loader.tsx"), "utf8")

describe("app-warmup-loader contract", () => {
  it("keeps restrained app/auth warmup semantics under shared ownership", () => {
    expect(source).toContain("export function AppWarmupLoader")
    expect(source).toContain("owner: string")
    expect(source).toContain("intent: AppWarmupIntent")
    expect(source).toContain("getLoadingOwnerAttributes")
    expect(source).toContain("role=\"status\"")
    expect(source).toContain("aria-live=\"polite\"")
  })

  it("requires a non-empty owner", () => {
    expect(source).toContain("AppWarmupLoader requires a non-empty owner")
  })

  it("does not keep the removed visible auth warmup card entrypoint", () => {
    expect(existsSync(path.resolve(process.cwd(), "components/shared/loading/auth-warmup-shell.tsx"))).toBe(false)
  })

  it("does not keep removed standalone legacy loading wrapper entrypoints", () => {
    expect(existsSync(path.resolve(process.cwd(), "components/loading-spinner.tsx"))).toBe(false)
    expect(existsSync(path.resolve(process.cwd(), "components/shared/loading/shield-loader.tsx"))).toBe(false)
  })
})
