import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/icons/index.tsx"), "utf8")
describe("index contract", () => {
  it("keeps the Tabler-backed shared source markers", () => {
    expect(source).toContain('from "@tabler/icons-react"')
    expect(source).toContain("semanticIcons")
    expect(source).not.toContain("@fluentui/react-icons")
    expect(source).not.toContain("@phosphor-icons/react")
    expect(source).not.toContain("@solar-icons/react")
    expect(source).not.toContain("lucide-react")
    expect(source).not.toContain("@icon-park/react")
  })
})
