import type { ScanWorkflow } from "@/types/scan-workflow.types"

export function normalizeScanWorkflowName(scanWorkflow?: string | null): string {
  return (scanWorkflow || "").replace(/^scanWorkflows\//, "").trim()
}

export function getScanWorkflowDisplayName(
  scanWorkflow: string | null | undefined,
  workflows: readonly Pick<ScanWorkflow, "name" | "displayName">[]
): string | undefined {
  const workflowName = normalizeScanWorkflowName(scanWorkflow)
  if (!workflowName) return undefined

  const workflow = workflows.find((item) => (
    normalizeScanWorkflowName(item.name) === workflowName
  ))

  return workflow?.displayName || workflow?.name
}

export function formatScanWorkflowDisplayName(scanWorkflow?: string | null): string {
  const normalized = normalizeScanWorkflowName(scanWorkflow)
  if (!normalized) return "-"

  return normalized
    .split("_")
    .filter(Boolean)
    .map((segment) => segment.charAt(0).toUpperCase() + segment.slice(1))
    .join(" ")
}
