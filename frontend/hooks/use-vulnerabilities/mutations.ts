import { useResourceMutation } from "@/hooks/_shared/create-resource-mutation"
import { VulnerabilityService } from "@/services/vulnerability.service"
import { vulnerabilityKeys } from "./keys"

/** Mark a single vulnerability as reviewed */
export function useMarkAsReviewed() {
  return useResourceMutation({
    mutationFn: (id: number) => VulnerabilityService.markAsReviewed(id),
    loadingToast: {
      key: "common.status.updating",
      params: {},
      id: (id) => `mark-vulnerability-reviewed-${id}`,
    },
    invalidate: [{ queryKey: vulnerabilityKeys.all }],
    onSuccess: ({ toast }) => {
      toast.success("vulnerabilities.reviewSuccess")
    },
    errorFallbackKey: "vulnerabilities.reviewError",
  })
}

/** Mark a single vulnerability as pending (unreview) */
export function useMarkAsUnreviewed() {
  return useResourceMutation({
    mutationFn: (id: number) => VulnerabilityService.markAsUnreviewed(id),
    loadingToast: {
      key: "common.status.updating",
      params: {},
      id: (id) => `mark-vulnerability-unreviewed-${id}`,
    },
    invalidate: [{ queryKey: vulnerabilityKeys.all }],
    onSuccess: ({ toast }) => {
      toast.success("vulnerabilities.unreviewSuccess")
    },
    errorFallbackKey: "vulnerabilities.unreviewError",
  })
}

/** Bulk mark vulnerabilities as reviewed */
export function useBulkMarkAsReviewed() {
  return useResourceMutation({
    mutationFn: (ids: number[]) => VulnerabilityService.bulkMarkAsReviewed(ids),
    loadingToast: {
      key: "vulnerabilities.bulkReviewLoading",
      params: (ids) => ({ count: ids.length }),
      id: "vulnerabilities-bulk-review",
    },
    invalidate: [{ queryKey: vulnerabilityKeys.all }],
    onSuccess: ({ data, toast }) => {
      toast.success("vulnerabilities.bulkReviewSuccess", { count: data.updatedCount })
    },
    errorFallbackKey: "vulnerabilities.bulkReviewError",
  })
}

/** Bulk mark vulnerabilities as pending (unreview) */
export function useBulkMarkAsUnreviewed() {
  return useResourceMutation({
    mutationFn: (ids: number[]) => VulnerabilityService.bulkMarkAsUnreviewed(ids),
    loadingToast: {
      key: "vulnerabilities.bulkUnreviewLoading",
      params: (ids) => ({ count: ids.length }),
      id: "vulnerabilities-bulk-unreview",
    },
    invalidate: [{ queryKey: vulnerabilityKeys.all }],
    onSuccess: ({ data, toast }) => {
      toast.success("vulnerabilities.bulkUnreviewSuccess", { count: data.updatedCount })
    },
    errorFallbackKey: "vulnerabilities.bulkUnreviewError",
  })
}

/** Bulk delete vulnerabilities */
export function useBulkDeleteVulnerabilities() {
  return useResourceMutation({
    mutationFn: (ids: number[]) => VulnerabilityService.bulkDelete(ids),
    loadingToast: {
      key: "vulnerabilities.bulkDeleteLoading",
      params: (ids) => ({ count: ids.length }),
      id: "vulnerabilities-bulk-delete",
    },
    invalidate: [{ queryKey: vulnerabilityKeys.all }],
    onSuccess: ({ data, toast }) => {
      toast.success("vulnerabilities.bulkDeleteSuccess", { count: data.deletedCount })
    },
    errorFallbackKey: "vulnerabilities.bulkDeleteError",
  })
}
