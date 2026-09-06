"use client"

import { useTranslations } from "next-intl"

import { PageRefreshStatusButton } from "@/components/common/page-refresh-status-button"
import { useScanHistoryRefresh } from "@/hooks/use-scan-history-refresh"

export function ScanHistoryPageRefresh() {
  const t = useTranslations("scan.history")
  const { isRefreshing, lastRefreshedAt, refresh } = useScanHistoryRefresh()

  return (
    <PageRefreshStatusButton
      isRefreshing={isRefreshing}
      lastRefreshedAt={lastRefreshedAt}
      onRefresh={refresh}
      refreshLabel={t("refresh")}
      updatedAtLabel={t("updatedAtLabel")}
      updatedAtUnavailable={t("updatedAtUnavailable")}
    />
  )
}
