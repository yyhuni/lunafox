import { describe, expect, it } from "vitest"
import {
  getColumnWidthContract,
  readComponentSource,
} from "@/test/utils/business-list-width-contract"

const source = readComponentSource("components/organization/targets/targets-columns.tsx")

describe("targets-columns contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("keeps the organization-targets table on a primary-name plus bounded type-badge model", () => {
    const name = getColumnWidthContract(source, "name")
    const type = getColumnWidthContract(source, "type")

    expect(name.minSize).toBeGreaterThanOrEqual(250)
    expect(type.maxSize).toBeLessThanOrEqual(140)
  })

  it("renders the target-type icon beside the target name with selected-row feedback", () => {
    expect(source).toContain("semanticIcons } from \"@/components/icons\"")
    expect(source).toContain("semanticIcons.concept[targetType]")
    expect(source).toContain("targetType={row.original.type}")
    expect(source).toContain('role="img" aria-label={targetTypeLabel} title={targetTypeLabel}')
    expect(source).toContain("group-data-[state=selected]:text-foreground")
    expect(source).not.toContain("bg-muted")
    expect(source).not.toContain("radius-surface flex size-8")
  })

  it("matches the target page timestamp columns", () => {
    expect(source).toContain('import { TimestampCell } from "@/components/shared/data-table/timestamp-cell"')
    expect(source).toContain('accessorKey: "createdAt"')
    expect(source).toContain('accessorKey: "lastScannedAt"')
    expect(source).toContain("formatDate(createdAt)")
    expect(source).toContain("formatDate(lastScannedAt)")
    expect(source).toContain('value={lastScannedAt ? formatDate(lastScannedAt) : "-"}')
    expect(source).toContain('size: 176')
    expect(source).toContain('maxSize: 220')
  })
})
