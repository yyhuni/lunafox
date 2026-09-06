import type {
  CanonicalFingerprintName,
  FingerprintFilterOptionField,
  FingerprintLibrary,
  FingerprintListParams,
} from "@/types/fingerprint.types"

function createLibraryKeys(library: FingerprintLibrary) {
  const all = () => ["fingerprintLibraries", library] as const
  return {
    all,
    list: (params: FingerprintListParams) => [...all(), "list", params] as const,
    detail: (name: CanonicalFingerprintName) => [...all(), "detail", name] as const,
    filterOptions: (field: FingerprintFilterOptionField) => [...all(), "filterOptions", field] as const,
  }
}

export const fingerprintKeys = {
  all: ["fingerprintLibraries"] as const,
  stats: () => [...fingerprintKeys.all, "statistics"] as const,
	  fingerprinthub: createLibraryKeys("fingerprinthub"),
}
