"use client"

import * as React from "react"
import { useLocale, useTranslations } from "next-intl"

import {
  useBulkDeleteFingerPrintHubFingerprints,
  useDeleteAllFingerPrintHubFingerprints,
  useExportFingerPrintHubFingerprints,
  useFingerPrintHubFingerprint,
  useFingerPrintHubFingerprints,
} from "@/hooks/use-fingerprints"
import { getDateLocale } from "@/lib/date-utils"
import type { FingerPrintHubFingerprint, FingerPrintHubFingerprintDetail } from "@/types/fingerprint.types"

import { createFingerPrintHubFingerprintColumns } from "./fingerprinthub-fingerprint-columns"
import { FingerPrintHubFingerprintDataTable } from "./fingerprinthub-fingerprint-data-table"
import { FingerprintDetailDrawer } from "./fingerprint-detail-drawer"
import { FingerprintLibraryWorkspace } from "./fingerprint-library-workspace"

function useFingerPrintHubFingerprintDateFormatter() {
  const locale = useLocale()

  return React.useCallback((dateString: string): string => {
    return new Date(dateString).toLocaleString(getDateLocale(locale), {
      year: "numeric",
      month: "numeric",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
      hour12: false,
    })
  }, [locale])
}

function useFingerPrintHubFingerprintColumns() {
  const tActions = useTranslations("common.actions")
  const tFingerprints = useTranslations("tools.fingerprints")
  const tFingerprintColumns = useTranslations("columns.fingerprint")
  const tFingerprintForm = useTranslations("tools.fingerprints.form")
  const formatDate = useFingerPrintHubFingerprintDateFormatter()

  return React.useMemo(
    () => createFingerPrintHubFingerprintColumns({
      formatDate,
      unsetLabel: tFingerprints("unset"),
      selectLabels: {
        selectAll: tActions("selectAll"),
        selectRow: tActions("selectRow"),
      },
      getColumnLabel: tFingerprintColumns,
      getFormLabel: tFingerprintForm,
    }),
    [formatDate, tActions, tFingerprints, tFingerprintColumns, tFingerprintForm]
  )
}

export function FingerPrintHubFingerprintView() {
  const columns = useFingerPrintHubFingerprintColumns()
  const formatDate = useFingerPrintHubFingerprintDateFormatter()

  return (
    <FingerprintLibraryWorkspace<FingerPrintHubFingerprint, FingerPrintHubFingerprintDetail>
      library="fingerprinthub"
      owner="fingerprinthub-fingerprint-view-content"
      columns={columns}
      formatDate={formatDate}
      useList={useFingerPrintHubFingerprints}
      useDetail={useFingerPrintHubFingerprint}
      useBulkDelete={useBulkDeleteFingerPrintHubFingerprints}
      useClear={useDeleteAllFingerPrintHubFingerprints}
      useExport={useExportFingerPrintHubFingerprints}
      renderTable={(props) => <FingerPrintHubFingerprintDataTable {...props} />}
      renderDrawer={({ fingerprint, ...props }) => (
        <FingerprintDetailDrawer inspection={{ source: "fingerprinthub", fingerprint }} {...props} />
      )}
    />
  )
}
