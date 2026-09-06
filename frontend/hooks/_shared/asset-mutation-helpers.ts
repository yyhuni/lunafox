import type { ToastParams } from "@/lib/toast-helpers"

export type AssetBulkCreateToastKeys = {
  success: string
  partial: string
}

export type AssetBulkCreateToast = {
  variant: "success" | "warning"
  key: string
  params: ToastParams
}

export const resolveAssetBulkCreateToast = (
  createdCount: number | undefined,
  keys: AssetBulkCreateToastKeys
): AssetBulkCreateToast => {
  const count = createdCount ?? 0
  if (count > 0) {
    return {
      variant: "success",
      key: keys.success,
      params: { count },
    }
  }

  return {
    variant: "warning",
    key: keys.partial,
    params: { success: 0, skipped: 0 },
  }
}

export const getAssetDeletedCount = (data: { deletedCount?: number }): number =>
  data.deletedCount ?? 0
