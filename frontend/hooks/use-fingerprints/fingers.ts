import { createFingerprintHooks } from "@/hooks/_shared/fingerprint-hooks"
import { FingerprintService } from "@/services/fingerprint.service"
import type {
  BatchCreateResponse,
  BulkDeleteResponse,
  FingersFingerprint,
} from "@/types/fingerprint.types"
import { fingerprintKeys } from "./keys"

const fingersHooks = createFingerprintHooks<FingersFingerprint, BatchCreateResponse, BulkDeleteResponse>({
  keys: fingerprintKeys.fingers,
  statsKey: fingerprintKeys.stats(),
  service: {
    list: FingerprintService.getFingersFingerprints,
    detail: FingerprintService.getFingersFingerprint,
    create: FingerprintService.createFingersFingerprint,
    update: FingerprintService.updateFingersFingerprint,
    remove: FingerprintService.deleteFingersFingerprint,
    importFromFile: FingerprintService.importFingersFingerprints,
    bulkDelete: FingerprintService.bulkDeleteFingersFingerprints,
    deleteAll: FingerprintService.deleteAllFingersFingerprints,
  },
})

export const useFingersFingerprints = fingersHooks.useList
export const useFingersFingerprint = fingersHooks.useDetail
export const useCreateFingersFingerprint = fingersHooks.useCreate
export const useUpdateFingersFingerprint = fingersHooks.useUpdate
export const useDeleteFingersFingerprint = fingersHooks.useDelete
export const useImportFingersFingerprints = fingersHooks.useImport
export const useBulkDeleteFingersFingerprints = fingersHooks.useBulkDelete
export const useDeleteAllFingersFingerprints = fingersHooks.useDeleteAll
