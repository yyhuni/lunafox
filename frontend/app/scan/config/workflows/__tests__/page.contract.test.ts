import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(
  path.resolve(process.cwd(), "app/scan/config/workflows/page.tsx"),
  "utf8"
)

describe("scan configuration workflows page contract", () => {
  it("uses the canonical route to mount workflow management", () => {
    expect(source).toContain("export default function ScanConfigurationWorkflowsPage")
    expect(source).toContain('from "@/components/scan/workflow/scan-workflow-page"')
    expect(source).toContain("<ScanWorkflowPageContent />")
    expect(source).not.toContain("redirect(")
  })
})
