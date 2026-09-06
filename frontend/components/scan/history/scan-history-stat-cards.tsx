"use client"

import { useTranslations } from "next-intl"
import {
  IconClock,
  semanticIcons,
} from "@/components/icons"
import { StatMetricRow, type StatMetricTone } from "@/components/shared/metrics/stat-metric-row"
import { useScanStatistics } from "@/hooks/use-scans"
import { getScanStatusMetricTone } from "@/lib/status-config"

type ScanHistoryStatStatus = "pending" | "running" | "succeeded" | "failed" | "cancelled"

const statTone: Record<ScanHistoryStatStatus, StatMetricTone> = {
  running: getScanStatusMetricTone("running"),
  pending: getScanStatusMetricTone("pending"),
  failed: getScanStatusMetricTone("failed"),
  cancelled: getScanStatusMetricTone("cancelled"),
  succeeded: getScanStatusMetricTone("succeeded"),
}

export function ScanHistoryStatCards() {
  const { data, isLoading } = useScanStatistics()

  return <ScanHistoryStatCardsContent data={data} isLoading={isLoading} />
}

export function ScanHistoryStatCardsLoadingState() {
  return <ScanHistoryStatCardsContent isLoading />
}

function ScanHistoryStatCardsContent({
  data,
  isLoading,
}: {
  data?: ReturnType<typeof useScanStatistics>["data"]
  isLoading: boolean
}) {
  const t = useTranslations("scan.history.stats")
  const tCommon = useTranslations("common.status")

  return (
    <StatMetricRow
      dataSlot={isLoading ? "scan-history-stats-skeleton" : undefined}
      featuredKey="running"
      className="scan-history-stats-row"
      items={[
        {
          key: "running",
          tone: statTone.running,
          title: tCommon("running"),
          value: data?.running ?? 0,
          footer: t("runningScans"),
          loading: isLoading,
        },
        {
          key: "pending",
          tone: statTone.pending,
          title: tCommon("pending"),
          value: data?.pending ?? 0,
          icon: <IconClock />,
          footer: t("pendingShort"),
          loading: isLoading,
        },
        {
          key: "failed",
          tone: statTone.failed,
          title: tCommon("failed"),
          value: data?.failed ?? 0,
          icon: <semanticIcons.status.failed />,
          footer: t("failedNeedsAttention"),
          loading: isLoading,
        },
        {
          key: "cancelled",
          tone: statTone.cancelled,
          title: tCommon("cancelled"),
          value: data?.cancelled ?? 0,
          icon: <semanticIcons.status.cancelled />,
          footer: t("cancelledShort"),
          loading: isLoading,
        },
        {
          key: "succeeded",
          tone: statTone.succeeded,
          title: tCommon("succeeded"),
          value: data?.succeeded ?? 0,
          icon: <semanticIcons.status.success />,
          footer: t("completedAll"),
          loading: isLoading,
        },
      ]}
    />
  )
}
