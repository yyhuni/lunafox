import { createFingerprintHooks } from "@/hooks/_shared/fingerprint-hooks"
import { FingerprintService } from "@/services/fingerprint.service"
import type {
  BatchCreateResponse,
  BulkDeleteResponse,
  WappalyzerFingerprint,
} from "@/types/fingerprint.types"
import { fingerprintKeys } from "./keys"

const wappalyzerHooks = createFingerprintHooks<WappalyzerFingerprint, BatchCreateResponse, BulkDeleteResponse>({
  keys: fingerprintKeys.wappalyzer,
  statsKey: fingerprintKeys.stats(),
  service: {
    list: FingerprintService.getWappalyzerFingerprints,
    detail: FingerprintService.getWappalyzerFingerprint,
    create: FingerprintService.createWappalyzerFingerprint,
    update: FingerprintService.updateWappalyzerFingerprint,
    remove: FingerprintService.deleteWappalyzerFingerprint,
    importFromFile: FingerprintService.importWappalyzerFingerprints,
    bulkDelete: FingerprintService.bulkDeleteWappalyzerFingerprints,
    deleteAll: FingerprintService.deleteAllWappalyzerFingerprints,
  },
})

export const useWappalyzerFingerprints = wappalyzerHooks.useList
export const useWappalyzerFingerprint = wappalyzerHooks.useDetail
export const useCreateWappalyzerFingerprint = wappalyzerHooks.useCreate
export const useUpdateWappalyzerFingerprint = wappalyzerHooks.useUpdate
export const useDeleteWappalyzerFingerprint = wappalyzerHooks.useDelete
export const useImportWappalyzerFingerprints = wappalyzerHooks.useImport
export const useBulkDeleteWappalyzerFingerprints = wappalyzerHooks.useBulkDelete
export const useDeleteAllWappalyzerFingerprints = wappalyzerHooks.useDeleteAll
