import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "hooks/use-fingerprints/keys.ts"), "utf8")

describe("keys contract", () => {
  it("owns canonical library collection and detail keys without hook coupling", () => {
    expect(source).toContain("FingerprintListParams")
    expect(source).toContain('const all = () => ["fingerprintLibraries", library] as const')
    expect(source).toContain('detail: (name: CanonicalFingerprintName) => [...all(), "detail", name] as const')
    expect(source).toContain('filterOptions: (field: FingerprintFilterOptionField) => [...all(), "filterOptions", field] as const')
    expect(source).toContain('stats: () => [...fingerprintKeys.all, "statistics"] as const')
    expect(source).not.toContain("@/hooks/_shared/fingerprint-hooks")
  })
})
