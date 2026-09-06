import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/fingerprints/fingerprinthub-fingerprint-columns.tsx"), "utf8")

describe("fingerprinthub-fingerprint-columns contract", () => {
  it("defaults every fixed FingerPrintHub field to a collection column", () => {
    expect(source).toContain("export function createFingerPrintHubFingerprintColumns")
    expect(source).toContain("ColumnDef<FingerPrintHubFingerprint>")
    expect(source).toContain('accessorKey: "displayName"')
    expect(source).toContain('accessorKey: "severity"')
    expect(source).toContain('accessorKey: "createdAt"')
    for (const field of ["fingerprintId", "author", "tags", "metadata", "http", "sourceFile"]) {
      expect(source).toContain(`"${field}"`)
    }
    expect(source).toContain("fingerprintValueColumn<FingerPrintHubFingerprint>")
  })
})
