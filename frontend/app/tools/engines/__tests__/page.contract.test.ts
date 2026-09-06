import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "app/tools/engines/page.tsx"), "utf8")

describe("legacy engines page contract", () => {
  it("redirects the legacy engines address to the canonical scan configuration route", () => {
    expect(source).toContain("export default function EnginesPage")
    expect(source).toContain('from "next/navigation"')
    expect(source).toContain('redirect("/scan/config/engines/")')
    expect(source).not.toContain("EngineInstallationPage")
  })
})
