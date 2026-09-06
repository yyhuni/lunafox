import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "hooks/use-api-key-settings.ts"), "utf8")

describe("use-api-key-settings contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useApiKeySettings")
    expect(source).toContain("from '@tanstack/react-query'")
  })
})
