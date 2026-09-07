/** FingerprintHub contracts consumed by the Observer Ward-compatible workspace. */

export const FINGERPRINT_LIBRARIES = ["fingerprinthub"] as const
export type FingerprintLibrary = (typeof FINGERPRINT_LIBRARIES)[number]
export type CanonicalFingerprintName = string
export type JsonPrimitive = string | number | boolean | null
export type JsonValue = JsonPrimitive | JsonValue[] | { [key: string]: JsonValue }

export type FingerprintFilterOptionField = "severity"
export type FingerprintFilterOptionFieldByLibrary = { fingerprinthub: FingerprintFilterOptionField }
export interface FingerprintFilterOption { value: string; label: string; count?: number }
export interface FingerprintFilterOptionsResponse { results: FingerprintFilterOption[] }
export interface AdditionalFingerprintField { path: string; value: JsonValue }
export interface FingerprintResource { name: CanonicalFingerprintName; createdAt: string }
export interface FingerprintDetailResource extends FingerprintResource { updatedAt: string; additionalFields: AdditionalFingerprintField[] }
export interface FingerPrintHubFingerprint extends FingerprintResource {
  fingerprintId: string; displayName: string; author: JsonValue | null; tags: JsonValue | null; severity: string | null
  metadata: JsonValue | null; http: JsonValue; sourceFile: JsonValue | null
}
export type FingerPrintHubFingerprintDetail = FingerPrintHubFingerprint & FingerprintDetailResource
export type FingerprintListByLibrary = { fingerprinthub: FingerPrintHubFingerprint }
export type FingerprintDetailByLibrary = { fingerprinthub: FingerPrintHubFingerprintDetail }
export type FingerprintByLibrary = FingerprintListByLibrary

export interface FingerprintTransport { name: unknown; createdAt: unknown; updatedAt?: unknown; additionalFields?: unknown; [key: string]: unknown }
export type FingerprintDetailTransport = FingerprintTransport
export interface FingerprintListTransportResponse { results: unknown; totalSize: unknown; nextPageToken?: unknown }
export interface FingerprintListParams { pageSize?: number; pageToken?: string; filter?: string; orderBy?: string }
export interface FingerprintListResponse<T> { results: T[]; totalSize: number; nextPageToken?: string }
export interface FingerprintImportResponse { createdCount: number; updatedCount: number; unchangedCount: number }
export interface FingerprintBatchDeleteResponse { deletedCount: number }
export interface FingerprintClearResponse { deletedCount: number }
export interface FingerprintStats { fingerprinthub: number }
export type FingerprintImportDiagnosticKind = "TRANSPORT" | "ENCODING" | "SYNTAX" | "FORMAT" | "RECORD"
export interface FingerprintImportDiagnostic {
  "@type": "type.googleapis.com/lunafox.v1.FingerprintImportDiagnostic"
  kind: FingerprintImportDiagnosticKind; library: FingerprintLibrary; reason: string
  recordIndex?: number; fieldPath?: string; line?: number; column?: number
}
export interface FingerprintImportErrorInfo { "@type": "type.googleapis.com/google.rpc.ErrorInfo"; reason: string; domain: string; metadata?: Record<string, string> }
export interface AipErrorResponse { error?: { code?: number; message?: string; status?: string; details?: unknown[] } }
