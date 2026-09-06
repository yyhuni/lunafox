import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "services/api-key-settings.service.ts"), "utf8")

describe("api-key-settings.service contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("from '@/lib/api-client'")
    expect(source).toContain("api.get<ApiKeySettings>('/settings/apiKeys/')")
    expect(source).toContain("api.patch<ApiKeySettings>('/settings/apiKeys/'")
  })
})
