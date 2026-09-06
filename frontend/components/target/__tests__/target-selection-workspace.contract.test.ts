import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(
  path.resolve(process.cwd(), "components/target/target-selection-workspace.tsx"),
  "utf8"
)

describe("target-selection-workspace contract", () => {
  it("owns the target-domain card workspace presentation", () => {
    expect(source).toContain("export function TargetSelectionWorkspace")
    expect(source).toContain('data-selection-entity="target"')
    expect(source).toContain("TargetWorkspacePagination")
    expect(source).toContain("canPreviousPage: boolean")
    expect(source).toContain("canNextPage: boolean")
    expect(source).toContain("onPreviousPage: () => void")
    expect(source).toContain("onNextPage: () => void")
    expect(source).toContain('toolbarDensity="compact"')
    expect(source).toContain("sm:grid-cols-2")
    expect(source).toContain('className="border-t bg-muted/30 sm:h-18"')
    expect(source).toContain('<Checkbox className="mt-2"')
    expect(source).not.toContain('variant="count"')
    expect(source).not.toContain('t("selectedCount"')
    expect(source).not.toContain("Command")
  })

  it("stays controlled and keeps query and payload ownership outside", () => {
    expect(source).toContain("selectedTargetId: number | null")
    expect(source).toContain("onSearchQueryChange")
    expect(source).toContain("onPreviousPage")
    expect(source).toContain("onNextPage")
    expect(source).toContain("onPageSizeChange")
    expect(source).toContain("onToggleTarget")
    expect(source).toContain("onClearTarget")
    expect(source).not.toContain("useTargets")
    expect(source).not.toContain("useQuery")
    expect(source).not.toContain("targetId:")
  })

  it("keeps target cards on the explicit subtype icon and localized label mapping", () => {
    expect(source).toContain("semanticIcons.concept[target.type]")
    expect(source).toContain("targetTypeLabels[target.type]")
  })
})
