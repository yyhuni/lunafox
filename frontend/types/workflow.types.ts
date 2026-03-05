/**
 * Workflow catalog type definitions
 */

// Read-only workflow capability metadata (from /api/workflows)
export interface ScanWorkflow {
  id: number
  name: string
  title?: string
  description?: string
  version?: string
  // Legacy compatibility fields (deprecated).
  configuration?: string
  isValid?: boolean
  createdAt?: string
  updatedAt?: string
}

// Workflow profile (from /api/workflows/profiles)
export interface WorkflowProfile {
  id: string
  name: string
  description?: string
  workflowNames?: string[]
  configuration: string
}
