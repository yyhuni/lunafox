/**
 * Workflow catalog type definitions
 */

export type WorkflowConfiguration = Record<string, unknown>
export type WorkflowConfigurationValue = WorkflowConfiguration | string

export interface ScanWorkflow {
  id: number
  name: string
  title?: string
  description?: string
  version?: string
  configuration?: WorkflowConfigurationValue
  isValid?: boolean
  createdAt?: string
  updatedAt?: string
}

export interface WorkflowProfile {
  id: string
  name: string
  description?: string
  workflowIds?: string[]
  workflowNames?: string[]
  configuration: WorkflowConfigurationValue
}
