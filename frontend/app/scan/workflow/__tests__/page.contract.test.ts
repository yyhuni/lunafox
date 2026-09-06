import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "app/scan/workflow/page.tsx"), "utf8")

describe("page contract", () => {
  it("redirects the legacy workflow address to the canonical scan configuration route", () => {
    expect(source).toContain("export default function ScanWorkflowPage")
    expect(source).toContain('from "next/navigation"')
    expect(source).toContain('redirect("/scan/config/workflows/")')
  })

  it("does not retain a second workflow management entry point", () => {
    expect(source).not.toContain("ScanWorkflowPageContent")
    expect(source).not.toContain("next/dynamic")
    expect(source).not.toContain("ContentHandoff")
  })
})
