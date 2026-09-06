import type { FingerprintLibrary, JsonValue } from "@/types/fingerprint.types"

type TextListPresentation = { separator: string; monospace: boolean }

const textListPresentations: Partial<Record<FingerprintLibrary, Partial<Record<string, TextListPresentation>>>> = {
  fingerprinthub: {
    author: { separator: ", ", monospace: false },
    tags: { separator: ", ", monospace: false },
  },
}

function isPrimitiveTextList(value: JsonValue): value is Array<string | number> {
  return Array.isArray(value) && value.every((entry) => typeof entry === "string" || typeof entry === "number")
}

export function getFingerprintTextListPresentation(library: FingerprintLibrary, field: string): TextListPresentation | undefined {
  return textListPresentations[library]?.[field]
}

/**
 * Only audited semantic list fields become readable text. Their original array
 * remains intact in the API, storage, and export paths; all other JSON values
 * retain their structured representation.
 */
export function formatFingerprintFieldValue(library: FingerprintLibrary, field: string, value: JsonValue | null | undefined): JsonValue | string | null | undefined {
  const presentation = getFingerprintTextListPresentation(library, field)
  if (presentation && value !== null && value !== undefined && isPrimitiveTextList(value)) {
    return value.join(presentation.separator)
  }

  return value
}
