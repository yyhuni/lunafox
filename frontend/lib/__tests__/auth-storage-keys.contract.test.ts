import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "lib/auth-storage-keys.mjs"), "utf8")

describe("auth-storage-keys contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export const AUTH_PRIMARY_TOKEN_KEY = \"accessToken\"")
  })
})
