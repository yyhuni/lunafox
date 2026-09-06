import { getErrorResponseData } from "@/lib/response-parser"
import {
  FINGERPRINT_LIBRARIES,
  type FingerprintImportDiagnostic,
  type FingerprintImportDiagnosticKind,
  type FingerprintLibrary,
} from "@/types/fingerprint.types"

export const FINGERPRINT_IMPORT_DIAGNOSTIC_TYPE =
  "type.googleapis.com/lunafox.v1.FingerprintImportDiagnostic" as const

const ERROR_INFO_TYPE = "type.googleapis.com/google.rpc.ErrorInfo"
const fingerprintLibraries = new Set<string>(FINGERPRINT_LIBRARIES)
const diagnosticKinds = new Set<FingerprintImportDiagnosticKind>([
  "TRANSPORT",
  "ENCODING",
  "SYNTAX",
  "FORMAT",
  "RECORD",
])

export type FingerprintImportFailure = {
  diagnostic: FingerprintImportDiagnostic | null
  reason: string | null
}

const reasonTranslationKeys: Record<string, string> = {
  FINGERPRINT_IMPORT_FILE_REQUIRED: "import.diagnostic.reasons.fileRequired",
  FINGERPRINT_IMPORT_FILE_EMPTY: "import.diagnostic.reasons.fileEmpty",
  FINGERPRINT_IMPORT_FILE_TOO_LARGE: "import.diagnostic.reasons.fileTooLarge",
  FINGERPRINT_IMPORT_MIME_UNSUPPORTED: "import.diagnostic.reasons.mimeUnsupported",
  FINGERPRINT_IMPORT_UNSUPPORTED_MEDIA_TYPE: "import.diagnostic.reasons.mimeUnsupported",
  FINGERPRINT_IMPORT_INVALID: "import.diagnostic.reasons.requestInvalid",
  FINGERPRINT_IMPORT_REQUEST_TOO_LARGE: "import.diagnostic.reasons.requestTooLarge",
  FINGERPRINT_IMPORT_INVALID_ENCODING: "import.diagnostic.reasons.invalidEncoding",
  FINGERPRINT_IMPORT_FORMAT_INVALID: "import.diagnostic.reasons.formatInvalid",
  UNSUPPORTED_MEDIA_TYPE: "import.diagnostic.reasons.mimeUnsupported",
  MULTIPART_INVALID: "import.diagnostic.reasons.requestInvalid",
  REQUEST_TOO_LARGE: "import.diagnostic.reasons.requestTooLarge",
  FILE_PART_INVALID: "import.diagnostic.reasons.filePartInvalid",
  FILE_READ_FAILED: "import.diagnostic.reasons.fileReadFailed",
  FILE_TOO_LARGE: "import.diagnostic.reasons.fileTooLarge",
  FILE_REQUIRED: "import.diagnostic.reasons.fileRequired",
  PARSE_ERROR: "import.diagnostic.reasons.parseError",
  INVALID_SYNTAX: "import.diagnostic.reasons.parseError",
  INVALID_UTF8: "import.diagnostic.reasons.invalidEncoding",
  FORMAT_MISMATCH: "import.diagnostic.reasons.formatInvalid",
  REQUIRED_FIELD_MISSING: "import.diagnostic.reasons.requiredFieldMissing",
  INVALID_FIELD_TYPE: "import.diagnostic.reasons.invalidFieldType",
  INVALID_RULE_STRUCTURE: "import.diagnostic.reasons.invalidRuleStructure",
  RULE_DEPTH_EXCEEDED: "import.diagnostic.reasons.ruleDepthExceeded",
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null
}

function getDetails(error: unknown): unknown[] {
  const response = getErrorResponseData(error)
  if (!isRecord(response) || !isRecord(response.error) || !Array.isArray(response.error.details)) {
    return []
  }

  return response.error.details
}

function getPositiveInteger(value: unknown): number | undefined {
  return typeof value === "number" && Number.isInteger(value) && value > 0 ? value : undefined
}

function toFingerprintImportDiagnostic(detail: unknown): FingerprintImportDiagnostic | null {
  if (!isRecord(detail) || detail["@type"] !== FINGERPRINT_IMPORT_DIAGNOSTIC_TYPE) {
    return null
  }

  if (
    typeof detail.kind !== "string" ||
    !diagnosticKinds.has(detail.kind as FingerprintImportDiagnosticKind) ||
    typeof detail.library !== "string" ||
    !fingerprintLibraries.has(detail.library) ||
    typeof detail.reason !== "string"
  ) {
    return null
  }

  const diagnostic: FingerprintImportDiagnostic = {
    "@type": FINGERPRINT_IMPORT_DIAGNOSTIC_TYPE,
    kind: detail.kind as FingerprintImportDiagnosticKind,
    library: detail.library as FingerprintLibrary,
    reason: detail.reason,
  }

  const recordIndex = getPositiveInteger(detail.recordIndex)
  const line = getPositiveInteger(detail.line)
  const column = getPositiveInteger(detail.column)

  if (recordIndex !== undefined) diagnostic.recordIndex = recordIndex
  if (typeof detail.fieldPath === "string" && detail.fieldPath.length > 0) {
    diagnostic.fieldPath = detail.fieldPath
  }
  if (line !== undefined) diagnostic.line = line
  if (column !== undefined) diagnostic.column = column

  return diagnostic
}

function getErrorInfoReason(detail: unknown): string | null {
  if (
    !isRecord(detail) ||
    detail["@type"] !== ERROR_INFO_TYPE ||
    typeof detail.reason !== "string"
  ) {
    return null
  }

  return detail.reason
}

/**
 * AIP does not guarantee detail ordering. Resolve each typed detail by its
 * type URL so a future server serialization order cannot hide the location.
 */
export function getFingerprintImportFailure(error: unknown): FingerprintImportFailure {
  const details = getDetails(error)
  const diagnostic = details
    .map(toFingerprintImportDiagnostic)
    .find((detail): detail is FingerprintImportDiagnostic => detail !== null) ?? null
  const errorInfoReason = details
    .map(getErrorInfoReason)
    .find((reason): reason is string => reason !== null) ?? null

  return {
    diagnostic,
    reason: diagnostic?.reason ?? errorInfoReason,
  }
}

export function getFingerprintImportReasonTranslationKey(reason: string | null): string | null {
  return reason ? reasonTranslationKeys[reason] ?? null : null
}
