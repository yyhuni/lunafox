import { createFingerprintHooks } from "@/hooks/_shared/fingerprint-hooks"
import { FingerprintService } from "@/services/fingerprint.service"
import type {
  ARLFingerprint,
  BatchCreateResponse,
  BulkDeleteResponse,
} from "@/types/fingerprint.types"
import { fingerprintKeys } from "./keys"

const arlHooks = createFingerprintHooks<ARLFingerprint, BatchCreateResponse, BulkDeleteResponse>({
  keys: fingerprintKeys.arl,
  statsKey: fingerprintKeys.stats(),
  service: {
    list: FingerprintService.getARLFingerprints,
    detail: FingerprintService.getARLFingerprint,
    create: FingerprintService.createARLFingerprint,
    update: FingerprintService.updateARLFingerprint,
    remove: FingerprintService.deleteARLFingerprint,
    importFromFile: FingerprintService.importARLFingerprints,
    bulkDelete: FingerprintService.bulkDeleteARLFingerprints,
    deleteAll: FingerprintService.deleteAllARLFingerprints,
  },
})

export const useARLFingerprints = arlHooks.useList
export const useARLFingerprint = arlHooks.useDetail
export const useCreateARLFingerprint = arlHooks.useCreate
export const useUpdateARLFingerprint = arlHooks.useUpdate
export const useDeleteARLFingerprint = arlHooks.useDelete
export const useImportARLFingerprints = arlHooks.useImport
export const useBulkDeleteARLFingerprints = arlHooks.useBulkDelete
export const useDeleteAllARLFingerprints = arlHooks.useDeleteAll
