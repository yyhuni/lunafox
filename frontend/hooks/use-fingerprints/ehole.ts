import { useResourceMutation } from "@/hooks/_shared/create-resource-mutation"
import {
  createFingerprintHooks,
  type FingerprintCreatePayload,
} from "@/hooks/_shared/fingerprint-hooks"
import { FingerprintService } from "@/services/fingerprint.service"
import type {
  BatchCreateResponse,
  BulkDeleteResponse,
  EholeFingerprint,
} from "@/types/fingerprint.types"
import { fingerprintKeys } from "./keys"

const eholeHooks = createFingerprintHooks<EholeFingerprint, BatchCreateResponse, BulkDeleteResponse>({
  keys: fingerprintKeys.ehole,
  statsKey: fingerprintKeys.stats(),
  service: {
    list: FingerprintService.getEholeFingerprints,
    detail: FingerprintService.getEholeFingerprint,
    create: FingerprintService.createEholeFingerprint,
    update: FingerprintService.updateEholeFingerprint,
    remove: FingerprintService.deleteEholeFingerprint,
    importFromFile: FingerprintService.importEholeFingerprints,
    bulkDelete: FingerprintService.bulkDeleteEholeFingerprints,
    deleteAll: FingerprintService.deleteAllEholeFingerprints,
  },
})

export const useEholeFingerprints = eholeHooks.useList
export const useEholeFingerprint = eholeHooks.useDetail
export const useCreateEholeFingerprint = eholeHooks.useCreate
export const useUpdateEholeFingerprint = eholeHooks.useUpdate
export const useDeleteEholeFingerprint = eholeHooks.useDelete
export const useImportEholeFingerprints = eholeHooks.useImport
export const useBulkDeleteEholeFingerprints = eholeHooks.useBulkDelete
export const useDeleteAllEholeFingerprints = eholeHooks.useDeleteAll

export function useBatchCreateEholeFingerprints() {
  return useResourceMutation<BatchCreateResponse, FingerprintCreatePayload<EholeFingerprint>[]>({
    mutationFn: (fingerprints: FingerprintCreatePayload<EholeFingerprint>[]) =>
      FingerprintService.batchCreateEholeFingerprints(fingerprints),
    invalidate: [
      { queryKey: fingerprintKeys.ehole.all() },
      { queryKey: fingerprintKeys.stats() },
    ],
    skipDefaultErrorHandler: true,
  })
}
