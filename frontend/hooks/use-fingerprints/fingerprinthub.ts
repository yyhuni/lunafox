import { createFingerprintHooks } from "@/hooks/_shared/fingerprint-hooks"
import { FingerprintService } from "@/services/fingerprint.service"
import type {
  BatchCreateResponse,
  BulkDeleteResponse,
  FingerPrintHubFingerprint,
} from "@/types/fingerprint.types"
import { fingerprintKeys } from "./keys"

const fingerPrintHubHooks = createFingerprintHooks<FingerPrintHubFingerprint, BatchCreateResponse, BulkDeleteResponse>({
  keys: fingerprintKeys.fingerprinthub,
  statsKey: fingerprintKeys.stats(),
  service: {
    list: FingerprintService.getFingerPrintHubFingerprints,
    detail: FingerprintService.getFingerPrintHubFingerprint,
    create: FingerprintService.createFingerPrintHubFingerprint,
    update: FingerprintService.updateFingerPrintHubFingerprint,
    remove: FingerprintService.deleteFingerPrintHubFingerprint,
    importFromFile: FingerprintService.importFingerPrintHubFingerprints,
    bulkDelete: FingerprintService.bulkDeleteFingerPrintHubFingerprints,
    deleteAll: FingerprintService.deleteAllFingerPrintHubFingerprints,
  },
})

export const useFingerPrintHubFingerprints = fingerPrintHubHooks.useList
export const useFingerPrintHubFingerprint = fingerPrintHubHooks.useDetail
export const useCreateFingerPrintHubFingerprint = fingerPrintHubHooks.useCreate
export const useUpdateFingerPrintHubFingerprint = fingerPrintHubHooks.useUpdate
export const useDeleteFingerPrintHubFingerprint = fingerPrintHubHooks.useDelete
export const useImportFingerPrintHubFingerprints = fingerPrintHubHooks.useImport
export const useBulkDeleteFingerPrintHubFingerprints = fingerPrintHubHooks.useBulkDelete
export const useDeleteAllFingerPrintHubFingerprints = fingerPrintHubHooks.useDeleteAll
