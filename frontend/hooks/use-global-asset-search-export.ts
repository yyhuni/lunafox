"use client"

import * as React from "react"

import { SearchExportService } from "@/services/search-export.service"
import { useToastMessages } from "@/lib/toast-helpers"
import type { AssetType } from "@/types/search.types"

type ExportAssetType = Extract<AssetType, "website" | "endpoint">

/** Owns the imperative grant/download action and its user-visible feedback. */
export function useGlobalAssetSearchExport(submittedQuery: string) {
  const toast = useToastMessages()

  const exportAll = React.useCallback(async (assetType: ExportAssetType) => {
    try {
      await SearchExportService.exportAll({ q: submittedQuery, assetType })
      toast.success("search.export.initiated")
    } catch {
      toast.error("search.export.failed")
    }
  }, [submittedQuery, toast])

  return { exportAll }
}
