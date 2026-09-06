import { createFingerprintHooks } from "@/hooks/_shared/fingerprint-hooks"
import { FingerprintService } from "@/services/fingerprint.service"
import type { FingerPrintHubFingerprint, FingerPrintHubFingerprintDetail } from "@/types/fingerprint.types"
import { fingerprintKeys } from "./keys"

const fingerPrintHubHooks = createFingerprintHooks<FingerPrintHubFingerprint, FingerPrintHubFingerprintDetail>({
  keys: fingerprintKeys.fingerprinthub,
  statsKey: fingerprintKeys.stats(),
  service: {
    list: FingerprintService.list,
    detail: FingerprintService.get,
    importFromFile: FingerprintService.import,
    batchDelete: FingerprintService.batchDelete,
    clear: FingerprintService.clear,
    exportAll: FingerprintService.export,
  },
})

export const useFingerPrintHubFingerprints = fingerPrintHubHooks.useList
export const useFingerPrintHubFingerprint = fingerPrintHubHooks.useDetail
export const useImportFingerPrintHubFingerprints = fingerPrintHubHooks.useImport
export const useBulkDeleteFingerPrintHubFingerprints = fingerPrintHubHooks.useBulkDelete
export const useDeleteAllFingerPrintHubFingerprints = fingerPrintHubHooks.useDeleteAll
export const useExportFingerPrintHubFingerprints = fingerPrintHubHooks.useExport
