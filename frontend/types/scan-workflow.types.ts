/**
 * Scan workflow catalog type definitions
 */

export type ScanWorkflowConfiguration = Record<string, unknown>
export type ScanWorkflowConfigurationValue = ScanWorkflowConfiguration | string

export interface ScanWorkflowStepView {
  stageId: string
  stepId: string
  engineId: string
  /** Initial outer Step enablement for a newly materialized Workflow Profile. */
  profileDefaultEnabled: boolean
}

export interface ScanWorkflowStageView {
  stageId: string
  steps: ScanWorkflowStepView[]
}

export interface ScanWorkflow {
  name: string
  displayName: string
  description: string
  stages: ScanWorkflowStageView[]
  /** Flattened render projection of stages; it carries no configuration. */
  steps: ScanWorkflowStepView[]
  isBuiltin: boolean
  isExecutable: boolean
  etag: string
  createTime: string
  updateTime: string
}

export interface ScanWorkflowProfile {
  name: string
  scanWorkflow: string
  configuration: ScanWorkflowConfiguration
}

export interface ScanWorkflowListResponse {
  scanWorkflows: ScanWorkflow[]
  nextPageToken: string
  totalSize: number
}

export interface CreateScanWorkflowInput {
  scanWorkflow: Required<Pick<ScanWorkflow, "displayName" | "description" | "stages">>
  requestId: string
  scanWorkflowId?: string
}

export interface UpdateScanWorkflowInput {
  scanWorkflow: Required<Pick<ScanWorkflow, "name" | "displayName" | "description" | "stages" | "etag">>
  updateMask: string[]
}
