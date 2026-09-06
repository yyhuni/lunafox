import apiClient from "@/lib/api-client"
import { scanWorkflowName } from "@/lib/resource-name"
import type {
  CreateScanWorkflowInput,
  ScanWorkflow,
  ScanWorkflowListResponse,
  ScanWorkflowProfile,
  ScanWorkflowStageView,
  ScanWorkflowStepView,
  UpdateScanWorkflowInput,
} from "@/types/scan-workflow.types"

export type ListScanWorkflowsInput = {
  pageSize?: number
  pageToken?: string
  filter?: string
}

export async function listScanWorkflows(input: ListScanWorkflowsInput = {}): Promise<ScanWorkflowListResponse> {
  const response = await apiClient.get("/scanWorkflows", { params: input })
  const payload = response.data ?? {}
  return {
    scanWorkflows: Array.isArray(payload.results)
      ? payload.results.map(normalizeWorkflow).filter(isWorkflow)
      : [],
    nextPageToken: typeof payload.nextPageToken === "string" ? payload.nextPageToken : "",
    totalSize: typeof payload.totalSize === "number" ? payload.totalSize : 0,
  }
}

export async function getScanWorkflow(id: string): Promise<ScanWorkflow> {
  const response = await apiClient.get(`/${scanWorkflowName(id)}`)
  return requireWorkflow(response.data)
}

export async function getScanWorkflowProfile(id: string): Promise<ScanWorkflowProfile> {
  const response = await apiClient.get(`/${scanWorkflowName(id)}/profile`)
  const payload = response.data ?? {}
  if (typeof payload.name !== "string" || typeof payload.scanWorkflow !== "string" || !isRecord(payload.configuration)) {
    throw new Error("Scan workflow Profile is missing its parent resource or complete configuration")
  }
  return {
    name: payload.name,
    scanWorkflow: payload.scanWorkflow,
    configuration: payload.configuration,
  }
}

export async function createScanWorkflow(input: CreateScanWorkflowInput): Promise<ScanWorkflow> {
  const response = await apiClient.post("/scanWorkflows", input)
  return requireWorkflow(response.data)
}

export async function updateScanWorkflow(id: string, input: UpdateScanWorkflowInput): Promise<ScanWorkflow> {
  const response = await apiClient.patch(`/${scanWorkflowName(id)}`, input)
  return requireWorkflow(response.data)
}

function requireWorkflow(payload: unknown): ScanWorkflow {
  const workflow = normalizeWorkflow(payload)
  if (!workflow) throw new Error("Scan workflow is missing a canonical resource name or pure orchestration topology")
  return workflow
}

function normalizeWorkflow(payload: unknown): ScanWorkflow | null {
  if (!isRecord(payload) || typeof payload.name !== "string" || !Array.isArray(payload.stages)) return null
  const stages = normalizeStages(payload.stages)
  if (!stages || stages.length === 0) return null
  return {
    name: payload.name,
    displayName: typeof payload.displayName === "string" ? payload.displayName : "",
    description: typeof payload.description === "string" ? payload.description : "",
    stages,
    steps: stages.flatMap((stage) => stage.steps),
    isBuiltin: payload.isBuiltin === true,
    isExecutable: payload.isExecutable === true,
    etag: typeof payload.etag === "string" ? payload.etag : "",
    createTime: typeof payload.createTime === "string" ? payload.createTime : "",
    updateTime: typeof payload.updateTime === "string" ? payload.updateTime : "",
  }
}

function normalizeStages(payload: unknown[]): ScanWorkflowStageView[] | null {
  const stages: ScanWorkflowStageView[] = []
  for (const stage of payload) {
    if (!isRecord(stage) || typeof stage.stageId !== "string" || !Array.isArray(stage.steps)) return null
    const stageId = stage.stageId
    const steps: ScanWorkflowStepView[] = []
    for (const step of stage.steps) {
      if (
        !isRecord(step) ||
        typeof step.stepId !== "string" ||
        typeof step.engineId !== "string" ||
        typeof step.profileDefaultEnabled !== "boolean"
      ) return null
      steps.push({
        stageId,
        stepId: step.stepId,
        engineId: step.engineId,
        profileDefaultEnabled: step.profileDefaultEnabled,
      })
    }
    if (steps.length === 0) return null
    stages.push({ stageId, steps })
  }
  return stages
}

function isWorkflow(value: ScanWorkflow | null): value is ScanWorkflow {
  return value !== null
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value)
}
