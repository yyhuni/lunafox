import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "hooks/use-scans/mutations.ts"), "utf8")

describe("mutations contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function useQuickScan")
    expect(source).toContain("from \"@/hooks/_shared/create-resource-mutation\"")
  })

  it("exposes a bulk initiate scan mutation with aggregate toast feedback", () => {
    expect(source).toContain("export function useBulkInitiateScan")
    expect(source).toContain("bulkInitiateScan,")
    expect(source).toContain("BulkInitiateScanRequest")
    expect(source).toContain("getBulkInitiateScanSuccessCount")
    expect(source).toContain("toast.scan.bulkInitiate.success")
    expect(source).toContain("toast.scan.bulkInitiate.error")
  })
})
