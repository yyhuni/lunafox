import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/edge-panel-header/edge-panel-header.tsx"), "utf8")
const readme = readFileSync(path.resolve(process.cwd(), "components/shared/edge-panel-header/README.md"), "utf8")
const indexSource = readFileSync(path.resolve(process.cwd(), "components/shared/edge-panel-header/index.ts"), "utf8")

describe("edge-panel-header shared contract", () => {
  it("exports the explicit scenario variants", () => {
    expect(source).toContain('"workbench" | "form" | "detail" | "compact"')
    expect(source).toContain("export function EdgePanelHeader")
    expect(indexSource).toContain("EdgePanelHeader")
    expect(readme).toContain("`workbench`")
    expect(readme).toContain("`form`")
    expect(readme).toContain("`detail`")
    expect(readme).toContain("`compact`")
  })

  it("keeps semantic title and description nodes caller-owned", () => {
    expect(source).toContain('data-slot="edge-panel-title"')
    expect(source).toContain('data-slot="edge-panel-description"')
    expect(source).not.toContain("SheetTitle")
    expect(source).not.toContain("DrawerTitle")
    expect(source).not.toContain("SheetDescription")
    expect(source).not.toContain("DrawerDescription")
    expect(readme).toContain("MUST NOT create a second accessible title or description")
  })

  it("keeps the responsive action and content constraints shared", () => {
    expect(source).toContain('"min-w-0 flex-1"')
    expect(source).toContain('data-slot="edge-panel-actions"')
    expect(source).toContain("shrink-0")
    expect(source).toContain("bg-muted text-muted-foreground")
    expect(source).toContain("size-10")
    expect(source).toContain("[&>svg]:size-7")
  })

  it("aligns form headers to their visible title content", () => {
    expect(source).toContain('variant === "form"')
    expect(source).toContain('hasDescription ? "items-start" : "items-center"')
    expect(readme).toContain("without one, they align vertically to the title row")
  })

  it("keeps workbench progress as a named slot", () => {
    expect(source).toContain("progress?: ReactNode")
    expect(source).toContain('variant === "workbench" && "pt-4"')
    expect(readme).toContain("progress or step slot")
  })
})
