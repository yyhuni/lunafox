import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/fingerprints/import-fingerprint-dialog.tsx"), "utf8")

describe("import-fingerprint-dialog contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function ImportFingerprintDialog")
    expect(source).toContain("FINGERPRINT_IMPORT_MAX_FILE_SIZE")
    expect(source).toContain("maxFiles={1}")
    expect(source).toContain("multiple={false}")
    expect(source).toContain("<Alert variant=\"destructive\"")
    expect(source).toContain("from \"react\"")
  })
})
