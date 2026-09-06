import React from "react"
import { semanticIcons } from "@/components/icons"
import { useScan } from "@/hooks/use-scans"
import { useTaskProgressLogs } from "@/hooks/use-task-progress-logs"

export const SCAN_RUNTIME_DETAIL_REFRESH_MS = 3000

type UseScanOverviewStateProps = {
  scanId: number
  t: (key: string, params?: Record<string, string | number | Date>) => string
  logsEnabled?: boolean
  refreshEnabled?: boolean
}

export function useScanOverviewState({ scanId, t, logsEnabled, refreshEnabled = false }: UseScanOverviewStateProps) {
  const { data: scan, isLoading, isFetching, error, refetch } = useScan(scanId)

  const isRunning = React.useMemo(
    () => scan?.status === "running" || scan?.status === "pending",
    [scan?.status]
  )

  const shouldRefreshRuntimeDetail = Boolean(refreshEnabled && scanId > 0 && isRunning)

  React.useEffect(() => {
    if (!shouldRefreshRuntimeDetail) return

    // Scope periodic reads to the open runtime drawer so list and overview queries remain manual.
    const intervalId = window.setInterval(() => {
      if (isFetching) return
      void refetch()
    }, SCAN_RUNTIME_DETAIL_REFRESH_MS)

    return () => window.clearInterval(intervalId)
  }, [isFetching, refetch, shouldRefreshRuntimeDetail])

  const [autoRefresh, setAutoRefresh] = React.useState(true)
  const [activeTab, setActiveTab] = React.useState<"logs" | "config">("logs")
  const resolvedLogsEnabled = logsEnabled ?? activeTab === "logs"
  const hasRuntimeLogs = React.useMemo(
    () => Boolean(scan?.runtimeTasks?.some((task) => task.status !== "skipped" || task.startedAt || task.completedAt)),
    [scan?.runtimeTasks]
  )

  const { logs, loading: logsLoading } = useTaskProgressLogs({
    scanId,
    enabled: Boolean(scan && resolvedLogsEnabled && hasRuntimeLogs),
    pollingInterval: isRunning && autoRefresh && resolvedLogsEnabled ? SCAN_RUNTIME_DETAIL_REFRESH_MS : 0,
  })

  const summary = React.useMemo(() => {
    const stats = scan?.cachedStats
    return {
      subdomains: stats?.subdomainsCount ?? 0,
      websites: stats?.websitesCount ?? 0,
      endpoints: stats?.endpointsCount ?? 0,
      ips: stats?.ipsCount ?? 0,
      directories: stats?.directoriesCount ?? 0,
      screenshots: stats?.screenshotsCount ?? 0,
    }
  }, [scan])

  const vulnSummary = React.useMemo(() => {
    const stats = scan?.cachedStats
    return {
      total: stats?.vulnsTotal ?? 0,
      critical: stats?.vulnsCritical ?? 0,
      high: stats?.vulnsHigh ?? 0,
      medium: stats?.vulnsMedium ?? 0,
      low: stats?.vulnsLow ?? 0,
    }
  }, [scan])

  const startedAt = React.useMemo(() => scan?.createdAt, [scan])
  const completedAt = React.useMemo(() => scan?.stoppedAt, [scan])

  const assetCards = React.useMemo(
    () => [
      {
        title: t("cards.websites"),
        value: summary.websites || 0,
        icon: semanticIcons.concept.website,
        code: "DAT-WEB",
        href: `/scan/history/${scanId}/websites/`,
      },
      {
        title: t("cards.subdomains"),
        value: summary.subdomains || 0,
        icon: semanticIcons.concept.subdomain,
        code: "DAT-SUB",
        href: `/scan/history/${scanId}/subdomains/`,
      },
      {
        title: t("cards.ips"),
        value: summary.ips || 0,
        icon: semanticIcons.concept.ip,
        code: "DAT-IP",
        href: `/scan/history/${scanId}/ip-addresses/`,
      },
      {
        title: t("cards.urls"),
        value: summary.endpoints || 0,
        icon: semanticIcons.concept.endpoint,
        code: "DAT-URL",
        href: `/scan/history/${scanId}/endpoints/`,
      },
      {
        title: t("cards.directories"),
        value: summary.directories || 0,
        icon: semanticIcons.concept.directory,
        code: "DAT-DIR",
        href: `/scan/history/${scanId}/directories/`,
      },
      {
        title: t("cards.screenshots"),
        value: summary.screenshots || 0,
        icon: semanticIcons.concept.screenshot,
        code: "DAT-SCR",
        href: `/scan/history/${scanId}/screenshots/`,
      },
    ],
    [scanId, summary, t]
  )

  return {
    scan,
    isLoading,
    error,
    refetch,
    logs,
    logsLoading,
    activeTab,
    setActiveTab,
    autoRefresh,
    setAutoRefresh,
    isRunning,
    summary,
    vulnSummary,
    startedAt,
    completedAt,
    assetCards,
  }
}

export type ScanOverviewState = ReturnType<typeof useScanOverviewState>
