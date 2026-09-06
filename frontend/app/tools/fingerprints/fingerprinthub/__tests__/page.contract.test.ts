import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "app/tools/fingerprints/fingerprinthub/page.tsx"), "utf8")

describe("page contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export default function FingerPrintHubFingerprintPage")
    expect(source).toContain("className")
    expect(source).toContain("from \"@/components/fingerprints/fingerprinthub-fingerprint-view\"")
    expect(source).not.toContain('"use client"')
  })

  it("directly imports the route-critical fingerprint workspace instead of hiding it behind a null chunk fallback", () => {
    expect(source).not.toContain("next/dynamic")
    expect(source).not.toContain("loading: () => null")
  })
})
