import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/feedback/status.tsx"), "utf8")

describe("index contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from 'react'")
  })

  it("consumes shared status config instead of local raw var utility maps", () => {
    expect(source).toContain("from '@/lib/status-config'")
    expect(source).not.toContain("bg-[var(--success)]")
    expect(source).not.toContain("bg-[var(--error)]")
  })
})
