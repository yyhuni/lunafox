import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/scheduled/scheduled-scan-columns.tsx"), "utf8")

describe("scheduled-scan-columns contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("keeps selection and task names clean for the scheduled scan table", () => {
    expect(source).toContain("id: \"select\"")
    expect(source).toContain("Checkbox")
    expect(source).not.toContain("核心资产")
    expect(source).not.toContain("API 监控")
  })

  it("uses semantic icons for repeated scheduled scan table concepts", () => {
    expect(source).toContain('semanticIcons')
    expect(source).toContain("semanticIcons.action.edit")
    expect(source).toContain("semanticIcons.action.delete")
    expect(source).toContain("semanticIcons.concept.organization")
    expect(source).toContain("semanticIcons.concept.target")
  })

  it("routes the scheduled scan row action menu through the shared dense row owner", () => {
    expect(source).toContain("DenseRowActionMenu")
    expect(source).toContain("ariaLabel={t.actions.openMenu}")
  })

  it("keeps scheduled scan row actions inside the dense overflow menu", () => {
    const rowActionsSource = source.slice(
      source.indexOf("function ScheduledScanRowActions"),
      source.indexOf("/**\n * Create scheduled scan table column definitions")
    )

    expect(rowActionsSource).toContain("DropdownMenuItem onClick={onEdit}")
    expect(rowActionsSource).toContain("DropdownMenuItem")
    expect(rowActionsSource).toContain("onClick={onDelete}")
    expect(rowActionsSource).not.toContain("<Button")
  })

  it("reuses the scheduled scan table layout owner for real column geometry", () => {
    expect(source).toContain('from "./scheduled-scan-table-layout"')
    expect(source).toContain("scheduledScanTableColumnLayout")
    expect(source).toContain("size: scheduledScanTableColumnLayout.name.size")
    expect(source).not.toContain('accessorKey: "engineNames"')
    expect(source).not.toContain("scheduledScanTableColumnLayout.engineNames")
    expect(source).toContain("size: scheduledScanTableColumnLayout.actions.size")
    expect(source).not.toContain("size: 250")
    expect(source).not.toContain("size: 120")
  })

  it("does not expose a derived engine column in scheduled scan tables", () => {
    expect(source).not.toContain("engineNames: string")
    expect(source).not.toContain("t.columns.engineNames")
    expect(source).not.toContain("row.original.engineNames")
    expect(source).not.toContain('accessorKey: "workflowNames"')
  })

  it("uses one compact, non-interactive handoff result metric", () => {
    const handoffColumnSource = source.slice(
      source.indexOf('id: "handoffResults"'),
      source.indexOf('accessorKey: "lastRunTime"')
    )

    expect(handoffColumnSource).toContain("scheduledScanTableColumnLayout.handoffResults")
    expect(handoffColumnSource).toContain('getStatusToneTextClass("success")')
    expect(handoffColumnSource).toContain('getStatusToneTextClass("error")')
    expect(handoffColumnSource).toContain("row.original.successfulHandoffCount")
    expect(handoffColumnSource).toContain("row.original.failedHandoffCount")
    expect(handoffColumnSource).not.toContain('accessorKey: "runCount"')
    expect(handoffColumnSource).not.toContain("<Badge")
    expect(handoffColumnSource).not.toContain("onClick")
  })
})
