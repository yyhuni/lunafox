import type { FingerprintLibrary, FingerprintListParams } from "@/types/fingerprint.types"
import { getMockFingerprintList } from "../data/fingerprints"

/**
 * Builder kept for the shared mock scenario surface. Fingerprint mock data now
 * uses the same page-token envelope as the production `/v1` collection.
 */
export function buildFingerprintList<T extends FingerprintLibrary>(
  library: T,
  params?: FingerprintListParams
) {
  return getMockFingerprintList(library, params)
}
