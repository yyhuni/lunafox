import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "app/tools/fingerprints/layout.tsx"), "utf8")

describe("layout contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export default function FingerprintsLayout")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("renders the singleton fingerprint workspace without retired route tabs", () => {
    expect(source).toContain("<PageHeader")
    expect(source).toContain("{children}")
    expect(source).not.toContain("TabsCountBadge")
    expect(source).not.toContain('from "@/components/ui/tabs"')
    expect(source).not.toContain("useFingerprintStats")
    expect(source).not.toContain("useRouter")
  })
})
