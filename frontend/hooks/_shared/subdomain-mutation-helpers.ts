import type { ToastParams } from "@/lib/toast-helpers"

export type SubdomainCreateCounts = {
  createdCount: number
  existedCount?: number
  skippedCount?: number
}

export type SubdomainCreateToast = {
  variant: "success" | "warning"
  key: string
  params: ToastParams
}

export const resolveSubdomainCreateToast = (
  data: SubdomainCreateCounts
): SubdomainCreateToast => {
  const skippedCount = (data.existedCount ?? 0) + (data.skippedCount ?? 0)
  if (skippedCount > 0) {
    return {
      variant: "warning",
      key: "toast.asset.subdomain.create.partialSuccess",
      params: {
        success: data.createdCount,
        skipped: skippedCount,
      },
    }
  }

  return {
    variant: "success",
    key: "toast.asset.subdomain.create.success",
    params: {
      count: data.createdCount,
    },
  }
}

export const getSubdomainBatchDeleteCount = (data: {
  deletedCount?: number
}): number => data.deletedCount ?? 0

export const getSubdomainBatchDeleteFromOrgCount = (data: {
  successCount?: number
}): number => data.successCount ?? 0
