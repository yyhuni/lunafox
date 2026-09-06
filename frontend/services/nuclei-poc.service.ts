import axios from "axios"

import apiClient from "@/lib/api-client"
import type {
  NucleiPocDetail,
  NucleiPocFilterOption,
  NucleiPocFilterOptionField,
  NucleiPocFilterOptionsResponse,
  NucleiPocListQuery,
  NucleiPocListItem,
  NucleiPocListResponse,
  NucleiPocRpcErrorBody,
  NucleiPocSource,
  NucleiPocSyncTask,
  SyncNucleiPocSourceRequest,
  SetNucleiPocActivationRequest,
  SetNucleiPocActivationResponse,
  UpdateNucleiPocRequest,
} from "@/types/nuclei-poc.types"

const SOURCE_PATH = "/nucleiPocSources/current"
const SYNC_PATH = "/nucleiPocSources:sync"
const TASK_PATH = "/nucleiPocSyncTasks"
const POC_PATH = "/nucleiPocs"
const FILTER_OPTIONS_PATH = "/nucleiPocs/filterOptions"
const ACTIVATION_PATH = "/nucleiPocs:setActivation"

export async function getNucleiPocSource(): Promise<NucleiPocSource | null> {
  try {
    const response = await apiClient.get<NucleiPocSource>(SOURCE_PATH)
    return response.data
  } catch (error) {
    if (axios.isAxiosError(error) && error.response?.status === 404) {
      return null
    }
    throw error
  }
}

export async function syncNucleiPocSource(
  request: SyncNucleiPocSourceRequest,
): Promise<NucleiPocSyncTask> {
  const response = await apiClient.post<NucleiPocSyncTask>(SYNC_PATH, request)
  return response.data
}

export async function getNucleiPocSyncTask(taskName: string): Promise<NucleiPocSyncTask> {
  const taskId = parseTaskId(taskName)
  const response = await apiClient.get<NucleiPocSyncTask>(`${TASK_PATH}/${encodeURIComponent(taskId)}`)
  return response.data
}

export async function listNucleiPocs(
  query: NucleiPocListQuery = {},
): Promise<NucleiPocListResponse> {
  const response = await apiClient.get<NucleiPocListResponse>(POC_PATH, {
    params: compactQuery(query),
  })
  return response.data
}

export async function getNucleiPocFilterOptions(
  field: NucleiPocFilterOptionField,
): Promise<NucleiPocFilterOptionsResponse> {
  if (field !== "tags") {
    throw new Error("Nuclei POC filter options support only the tags field")
  }
  const response = await apiClient.get<unknown>(FILTER_OPTIONS_PATH, { params: { field } })
  return parseNucleiPocFilterOptionsResponse(response.data)
}

export async function getNucleiPoc(name: string): Promise<NucleiPocDetail> {
  const templateId = parsePocTemplateId(name)
  const response = await apiClient.get<unknown>(`${POC_PATH}/${encodeURIComponent(templateId)}`)
  return parseNucleiPocDetail(response.data)
}

export async function updateNucleiPoc(
	request: UpdateNucleiPocRequest,
	): Promise<NucleiPocListItem> {
	const templateId = parsePocTemplateId(request.name)
	const response = await apiClient.patch<NucleiPocListItem>(`${POC_PATH}/${encodeURIComponent(templateId)}`, request)
  return response.data
}

export async function setNucleiPocActivation(
  request: SetNucleiPocActivationRequest,
): Promise<SetNucleiPocActivationResponse> {
  const hasNames = Boolean(request && typeof request === "object" && Object.prototype.hasOwnProperty.call(request, "names"))
  if (
    !request ||
    typeof request !== "object" ||
    Array.isArray(request) ||
    Object.keys(request).some((field) => field !== "enabled" && field !== "names") ||
    typeof request.enabled !== "boolean"
  ) {
    throw new Error("Nuclei POC activation requires an explicit enabled target")
  }
  let names: `nucleiPocs/${string}`[] | undefined
  if (hasNames) {
    if (!Array.isArray(request.names) || request.names.length < 1 || request.names.length > 1000) {
      throw new Error("Nuclei POC activation names must contain between 1 and 1000 entries")
    }
    const seen = new Set<string>()
    names = request.names.map((name, index) => {
      if (typeof name !== "string") {
        throw new Error(`Nuclei POC activation name at names[${index}] must be a string`)
      }
      const templateId = parsePocTemplateId(name)
      if (name !== `nucleiPocs/${templateId}`) {
        throw new Error(`Nuclei POC activation name at names[${index}] must be canonical`)
      }
      if (seen.has(name)) {
        throw new Error(`Nuclei POC activation names contains a duplicate at names[${index}]`)
      }
      seen.add(name)
      return name as `nucleiPocs/${string}`
    })
  }
  const payload = names === undefined
    ? { enabled: request.enabled }
    : { enabled: request.enabled, names }
  const response = await apiClient.post<unknown>(ACTIVATION_PATH, payload)
  return parseNucleiPocActivationResponse(response.data, request.enabled)
}

function parseNucleiPocActivationResponse(value: unknown, requestedEnabled: boolean): SetNucleiPocActivationResponse {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    throw new Error("Nuclei POC activation response must be an object")
  }
  const record = value as Record<string, unknown>
  for (const field of Object.keys(record)) {
    if (field !== "enabled" && field !== "affectedCount") {
      throw new Error(`Nuclei POC activation response has an unsupported field: ${field}`)
    }
  }
  if (typeof record.enabled !== "boolean") {
    throw new Error("Nuclei POC activation response has an invalid enabled target")
  }
  if (record.enabled !== requestedEnabled) {
    throw new Error("Nuclei POC activation response target does not match the request")
  }
  if (typeof record.affectedCount !== "number" || !Number.isSafeInteger(record.affectedCount) || record.affectedCount < 0) {
    throw new Error("Nuclei POC activation response has an invalid affectedCount")
  }
  return { enabled: record.enabled, affectedCount: record.affectedCount }
}

function parseNucleiPocFilterOptionsResponse(value: unknown): NucleiPocFilterOptionsResponse {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    throw new Error("Nuclei POC filter options response must be an object")
  }
  const record = value as Record<string, unknown>
  for (const field of Object.keys(record)) {
    if (field !== "results") {
      throw new Error(`Nuclei POC filter options response has an unsupported field: ${field}`)
    }
  }
  if (!Array.isArray(record.results)) {
    throw new Error("Nuclei POC filter options response has an invalid results collection")
  }
  return { results: record.results.map(parseNucleiPocFilterOption) }
}

function parseNucleiPocFilterOption(value: unknown, index: number): NucleiPocFilterOption {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    throw new Error(`Nuclei POC filter option at results[${index}] must be an object`)
  }
  const record = value as Record<string, unknown>
  for (const field of Object.keys(record)) {
    if (field !== "value" && field !== "label" && field !== "count") {
      throw new Error(`Nuclei POC filter option at results[${index}] has an unsupported field: ${field}`)
    }
  }
  if (typeof record.value !== "string" || record.value === "" || record.value !== record.value.trim() || record.value !== record.value.toLowerCase()) {
    throw new Error(`Nuclei POC filter option at results[${index}] has an invalid value`)
  }
  if (record.label !== record.value) {
    throw new Error(`Nuclei POC filter option at results[${index}] has an invalid label`)
  }
  if (typeof record.count !== "number" || !Number.isSafeInteger(record.count) || record.count < 1) {
    throw new Error(`Nuclei POC filter option at results[${index}] has an invalid count`)
  }
  return { value: record.value, label: record.label, count: record.count }
}

function parseNucleiPocDetail(value: unknown): NucleiPocDetail {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    throw new Error("Nuclei POC detail response must be an object")
  }
  const record = value as Record<string, unknown>
  for (const field of ["tags", "cve", "cwe", "references"] as const) {
    if (!Array.isArray(record[field]) || !record[field].every((item) => typeof item === "string")) {
      throw new Error(`Nuclei POC detail response has an invalid ${field}`)
    }
  }
  return record as unknown as NucleiPocDetail
}

export function parseTaskId(taskName: string): string {
  const value = taskName
  const match = /^nucleiPocSyncTasks\/([0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12})$/i.exec(value)
  if (!match || match[1] === "00000000-0000-0000-0000-000000000000") {
    throw new Error("Nuclei POC sync task name must be canonical")
  }
  return match[1]
}

export function parsePocTemplateId(name: string): string {
  const value = name
  const match = /^nucleiPocs\/([^/]+)$/.exec(value)
  if (!match || !match[1]) {
    throw new Error("Nuclei POC name must be canonical")
  }
  let templateId: string
  try {
    templateId = decodeURIComponent(match[1])
  } catch {
    throw new Error("Nuclei POC name must be canonical")
  }
  if (!templateId || templateId !== templateId.trim() || /[\\/\u0000-\u001f\u007f]/.test(templateId)) {
    throw new Error("Nuclei POC name must be canonical")
  }
  return templateId
}

export function getNucleiPocErrorBody(error: unknown): NucleiPocRpcErrorBody | null {
  if (!axios.isAxiosError(error)) return null
  const body = error.response?.data
  return body && typeof body === "object" ? body as NucleiPocRpcErrorBody : null
}

export function getNucleiPocErrorReason(error: unknown): string | null {
  const details = getNucleiPocErrorBody(error)?.error?.details
  if (!details) return null
  const detail = details.find(isNucleiPocErrorInfo)
  return detail && typeof detail.reason === "string" ? detail.reason : null
}

export function getNucleiPocActiveSyncTaskName(error: unknown): NucleiPocSyncTask["name"] | null {
  const body = getNucleiPocErrorBody(error)
  if (getNucleiPocHttpStatus(error) !== 409 || body?.error?.status !== "ABORTED") return null
  const detail = body.error.details?.find((candidate) => isNucleiPocErrorInfo(candidate) && candidate.reason === "SYNC_ALREADY_RUNNING")
  if (!detail) return null
  const metadata = detail.metadata
  if (!metadata || typeof metadata !== "object") return null
  const taskName = (metadata as Record<string, unknown>).task
  if (typeof taskName !== "string") return null
  try {
    parseTaskId(taskName)
    return taskName as NucleiPocSyncTask["name"]
  } catch {
    return null
  }
}

function isNucleiPocErrorInfo(candidate: unknown): candidate is Record<string, unknown> {
  return Boolean(
    candidate
    && typeof candidate === "object"
    && (candidate as Record<string, unknown>)["@type"] === "type.googleapis.com/google.rpc.ErrorInfo"
    && typeof (candidate as Record<string, unknown>).reason === "string",
  )
}

export function getNucleiPocHttpStatus(error: unknown): number | undefined {
  return axios.isAxiosError(error) ? error.response?.status : undefined
}

function compactQuery(query: NucleiPocListQuery): Record<string, string | number> {
  const output: Record<string, string | number> = {}
  if (query.pageSize !== undefined) output.pageSize = query.pageSize
  if (query.pageToken) output.pageToken = query.pageToken
  if (query.filter) output.filter = query.filter
  if (query.orderBy) output.orderBy = query.orderBy
  return output
}
