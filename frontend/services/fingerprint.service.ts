import { api } from "@/lib/api-client"
import type {
  AdditionalFingerprintField, CanonicalFingerprintName, FingerPrintHubFingerprint, FingerPrintHubFingerprintDetail,
  FingerprintBatchDeleteResponse, FingerprintClearResponse, FingerprintFilterOptionsResponse, FingerprintImportResponse,
  FingerprintListParams, FingerprintListResponse, FingerprintListTransportResponse, FingerprintStats, FingerprintTransport, JsonValue,
} from "@/types/fingerprint.types"

const library = "fingerprinthub"
const libraryPath = `/fingerprintLibraries/${library}`
const collectionPath = `${libraryPath}/fingerprints`
function invalidResponse(field: string): never { throw new Error(`Fingerprint API response is invalid at ${field}.`) }
function asObject(value: unknown, field: string): Record<string, unknown> { if (typeof value !== "object" || value === null || Array.isArray(value)) return invalidResponse(field); return value as Record<string, unknown> }
function requiredString(record: Record<string, unknown>, field: string) { const value = record[field]; if (typeof value !== "string" || value.length === 0) return invalidResponse(field); return value }
function asJsonValue(value: unknown, field: string): JsonValue {
  if (value === null || typeof value === "string" || typeof value === "boolean" || (typeof value === "number" && Number.isFinite(value))) return value
  if (Array.isArray(value)) return value.map((item, index) => asJsonValue(item, `${field}[${index}]`))
  if (typeof value === "object" && value !== null) return Object.fromEntries(Object.entries(value).map(([key, item]) => [key, asJsonValue(item, `${field}.${key}`)]))
  return invalidResponse(field)
}
function nullableJson(record: Record<string, unknown>, field: string): JsonValue | null { return !(field in record) || record[field] === null ? null : asJsonValue(record[field], field) }
function optionalString(record: Record<string, unknown>, field: string): string | null { const value = record[field]; if (value === undefined || value === null) return null; if (typeof value !== "string") return invalidResponse(field); return value }
function assertCanonicalName(name: string) { if (!/^fingerprintLibraries\/fingerprinthub\/fingerprints\/[0-9a-f-]{36}$/i.test(name)) invalidResponse("name") }
function mapRecord(value: unknown, detail: boolean): FingerPrintHubFingerprint | FingerPrintHubFingerprintDetail {
  const record = asObject(value, "fingerprint")
  for (const field of ["payload", "resourceId", "resource_id", "identityKey", "identity_key", "contentHash", "content_hash"]) if (field in record) invalidResponse(field)
  const name = requiredString(record, "name"); assertCanonicalName(name)
  const projected: FingerPrintHubFingerprint = {
    name, createdAt: requiredString(record, "createdAt"), fingerprintId: requiredString(record, "fingerprintId"), displayName: requiredString(record, "displayName"),
    author: nullableJson(record, "author"), tags: nullableJson(record, "tags"), severity: optionalString(record, "severity"), metadata: nullableJson(record, "metadata"), http: asJsonValue(record.http, "http"), sourceFile: nullableJson(record, "sourceFile"),
  }
  if (!detail) return projected
  if (!Array.isArray(record.additionalFields)) return invalidResponse("additionalFields")
  const additionalFields: AdditionalFingerprintField[] = record.additionalFields.map((candidate, index) => { const field = asObject(candidate, `additionalFields[${index}]`); const path = requiredString(field, "path"); if (!path.startsWith("/")) invalidResponse(`additionalFields[${index}].path`); return { path, value: asJsonValue(field.value, `additionalFields[${index}].value`) } })
  return { ...projected, updatedAt: requiredString(record, "updatedAt"), additionalFields }
}
function parseDownloadFilename(value: unknown): string { if (typeof value !== "string") return invalidResponse("content-disposition"); const match = /(?:^|;)\s*filename="?([^";]+)"?/i.exec(value); if (!match?.[1]) return invalidResponse("content-disposition.filename"); return match[1] }

export class FingerprintService {
  static async list(params?: FingerprintListParams): Promise<FingerprintListResponse<FingerPrintHubFingerprint>> {
    const response = await api.get<FingerprintListTransportResponse>(collectionPath, { params: { pageSize: params?.pageSize ?? 20, ...(params?.pageToken ? { pageToken: params.pageToken } : {}), ...(params?.filter ? { filter: params.filter } : {}), ...(params?.orderBy ? { orderBy: params.orderBy } : {}) } })
	const data = asObject(response.data, "response") as unknown as FingerprintListTransportResponse
    if (!Array.isArray(data.results) || typeof data.totalSize !== "number" || !Number.isInteger(data.totalSize) || data.totalSize < 0) return invalidResponse("results")
	return { results: data.results.map((record) => mapRecord(record, false) as FingerPrintHubFingerprint), totalSize: data.totalSize, ...(typeof data.nextPageToken === "string" && data.nextPageToken ? { nextPageToken: data.nextPageToken } : {}) }
  }
  static async get(name: CanonicalFingerprintName): Promise<FingerPrintHubFingerprintDetail> { assertCanonicalName(name); return mapRecord((await api.get<FingerprintTransport>(`/${name}`)).data, true) as FingerPrintHubFingerprintDetail }
  static async getFilterOptions(): Promise<FingerprintFilterOptionsResponse> { const response = await api.get<unknown>(`${collectionPath}/filterOptions`, { params: { field: "severity" } }); const data = asObject(response.data, "filterOptions"); if (!Array.isArray(data.results)) return invalidResponse("filterOptions.results"); return { results: data.results.map((item, index) => { const row = asObject(item, `filterOptions.results[${index}]`); return { value: requiredString(row, "value"), label: requiredString(row, "label"), ...(row.count === undefined ? {} : { count: row.count as number }) } }) } }
  static async import(file: File): Promise<FingerprintImportResponse> { const body = new FormData(); body.append("file", file); return (await api.post<FingerprintImportResponse>(`${libraryPath}:import`, body, { headers: { "Content-Type": undefined } })).data }
  static async batchDelete(names: CanonicalFingerprintName[]): Promise<FingerprintBatchDeleteResponse> { return (await api.post<FingerprintBatchDeleteResponse>(`${collectionPath}:batchDelete`, { names })).data }
  static async clear(): Promise<FingerprintClearResponse> { return (await api.post<FingerprintClearResponse>(`${libraryPath}:clear`)).data }
  static async export(): Promise<{ blob: Blob; filename: string }> { const response = await api.get<Blob>(`${libraryPath}/exportFiles/current`, { responseType: "blob" }); return { blob: response.data, filename: parseDownloadFilename(response.headers?.["content-disposition"] ?? response.headers?.["Content-Disposition"]) } }
  static async getStats(): Promise<FingerprintStats> { return (await api.get<FingerprintStats>("/fingerprintLibraryStatistics")).data }
}
