import { createFingerprintHooks } from "@/hooks/_shared/fingerprint-hooks"
import { FingerprintService } from "@/services/fingerprint.service"
import type {
  BatchCreateResponse,
  BulkDeleteResponse,
  GobyFingerprint,
} from "@/types/fingerprint.types"
import { fingerprintKeys } from "./keys"

const gobyHooks = createFingerprintHooks<GobyFingerprint, BatchCreateResponse, BulkDeleteResponse>({
  keys: fingerprintKeys.goby,
  statsKey: fingerprintKeys.stats(),
  service: {
    list: FingerprintService.getGobyFingerprints,
    detail: FingerprintService.getGobyFingerprint,
    create: FingerprintService.createGobyFingerprint,
    update: FingerprintService.updateGobyFingerprint,
    remove: FingerprintService.deleteGobyFingerprint,
    importFromFile: FingerprintService.importGobyFingerprints,
    bulkDelete: FingerprintService.bulkDeleteGobyFingerprints,
    deleteAll: FingerprintService.deleteAllGobyFingerprints,
  },
})

export const useGobyFingerprints = gobyHooks.useList
export const useGobyFingerprint = gobyHooks.useDetail
export const useCreateGobyFingerprint = gobyHooks.useCreate
export const useUpdateGobyFingerprint = gobyHooks.useUpdate
export const useDeleteGobyFingerprint = gobyHooks.useDelete
export const useImportGobyFingerprints = gobyHooks.useImport
export const useBulkDeleteGobyFingerprints = gobyHooks.useBulkDelete
export const useDeleteAllGobyFingerprints = gobyHooks.useDeleteAll
