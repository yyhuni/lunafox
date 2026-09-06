import { readFileSync } from "node:fs"
import path from "node:path"
import { describe, expect, it } from "vitest"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/scan-input-source-selector.tsx"), "utf8")

describe("scan input source selector contract", () => {
  it("renders localized source choices as an explanatory radio group", () => {
    expect(source).toContain('import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group"')
    expect(source).toContain('<RadioGroup\n        id={id}')
    expect(source).toContain("id={id}")
    expect(source).toContain('value: "scanSnapshot" as const')
    expect(source).toContain('value: "targetInventory" as const')
    expect(source).toContain("recommended: true")
    expect(source).toContain('t("scanSnapshotDescription")')
    expect(source).toContain('t("targetInventoryDescription")')
    expect(source).toContain('<Badge variant="outline">{t("recommended")}</Badge>')
    expect(source).toContain("<RadioGroupItem")
    expect(source).toContain('className="mt-1 translate-y-px"')
  })

  it("keeps a closed input-source value set before updating state", () => {
    expect(source).toContain("if (!isScanInputSource(nextValue))")
    expect(source).toContain("onValueChange(nextValue)")
  })
})
