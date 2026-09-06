import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/scan/workflow/scan-workflow-management-columns.tsx"), "utf8")

describe("scan-workflow-management-columns contract", () => {
  it("shares the compact workflow management table columns across content and loading", () => {
    expect(source).toContain("export function createWorkflowManagementColumns")
    expect(source).toContain("WorkflowManagementColumnRow")
    expect(source).not.toContain('id: "select"')
    expect(source).toContain('accessorKey: "scanWorkflowId"')
    expect(source).toContain('accessorKey: "title"')
    expect(source).toContain('accessorKey: "description"')
    expect(source).toContain('accessorKey: "engineCount"')
    expect(source).toContain("scanWorkflowId: string")
    expect(source).toContain('tWorkflow("management.columns.id")')
    expect(source).toContain("size: 180")
    expect(source).toContain("size: 240")
    expect(source).toContain("size: 280")
    expect(source).toContain("size: 76")
    expect(source).not.toContain('workflow.name}</div>')
  })

  it("keeps secondary status and structure columns out while offering view or edit only", () => {
    expect(source).not.toContain('accessorKey: "status"')
    expect(source).not.toContain('id: "structure"')
    expect(source).not.toContain('accessorKey: "stageCount"')
    expect(source).not.toContain('accessorKey: "lastRunStatus"')
    expect(source).not.toContain('accessorKey: "updatedAt"')
    expect(source).toContain('id: "actions"')
    expect(source).toContain("onEdit?: (workflow: TData) => void")
    expect(source).not.toContain("onDelete?: (workflow: TData) => void")
    expect(source).toContain("semanticIcons.action.edit")
    expect(source).toContain("DenseRowActionMenu")
    expect(source).toContain("row.original.isBuiltin ? viewLabel : editLabel")
    expect(source).toContain("builtinLabel")
    expect(source).toContain("unavailableLabel")
    expect(source).not.toContain("DropdownMenuSeparator")
    expect(source).not.toContain('variant="destructive"')
    expect(source).not.toContain("leadingActions=")
    expect(source).toContain("size: 64")
    expect(source).not.toContain('data-badge-type="workflow"')
  })
})
