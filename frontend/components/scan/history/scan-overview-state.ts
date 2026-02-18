import React from "react"
import { Globe, Network, Server, Link2, FolderOpen, Camera } from "@/components/icons"
import { useScan } from "@/hooks/use-scans"
import { useScanLogs } from "@/hooks/use-scan-logs"
import { useStarNudge } from "@/hooks/use-star-nudge"
import type { ScanRecord } from "@/types/scan.types"

type UseScanOverviewStateProps = {
  scanId: number
  t: (key: string, params?: Record<string, string | number | Date>) => string
}

type LegacyVulnerabilitySummary = {
  total?: number
  critical?: number
  high?: number
  medium?: number
  low?: number
}

type LegacyScanSummary = {
  subdomains?: number
  websites?: number
  endpoints?: number
  ips?: number
  directories?: number
  screenshots?: number
  vulnerabilities?: LegacyVulnerabilitySummary
}

type ScanRecordWithLegacy = ScanRecord & {
  summary?: LegacyScanSummary
  startedAt?: string
  completedAt?: string
}

export function useScanOverviewState({ scanId, t }: UseScanOverviewStateProps) {
  const { data: scan, isLoading, error } = useScan(scanId)
  const { trigger: triggerStarNudge } = useStarNudge({ probability: 1 })

  React.useEffect(() => {
    if (scan?.status === "completed") {
      triggerStarNudge()
    }
  }, [scan?.status, triggerStarNudge])

  const scanWithLegacy = scan as ScanRecordWithLegacy | undefined

  const isRunning = React.useMemo(
    () => scan?.status === "running" || scan?.status === "pending",
    [scan?.status]
  )

  const [autoRefresh, setAutoRefresh] = React.useState(true)
  const [activeTab, setActiveTab] = React.useState<"logs" | "config">("logs")

  const { logs, loading: logsLoading } = useScanLogs({
    scanId,
    enabled: Boolean(scan && activeTab === "logs"),
    pollingInterval: isRunning && autoRefresh && activeTab === "logs" ? 3000 : 0,
  })

  const summary = React.useMemo(() => {
    const stats = scan?.cachedStats
    const legacy = scanWithLegacy?.summary
    return {
      subdomains: stats?.subdomainsCount ?? legacy?.subdomains ?? 0,
      websites: stats?.websitesCount ?? legacy?.websites ?? 0,
      endpoints: stats?.endpointsCount ?? legacy?.endpoints ?? 0,
      ips: stats?.ipsCount ?? legacy?.ips ?? 0,
      directories: stats?.directoriesCount ?? legacy?.directories ?? 0,
      screenshots: stats?.screenshotsCount ?? legacy?.screenshots ?? 0,
    }
  }, [scan, scanWithLegacy])

  const vulnSummary = React.useMemo(() => {
    const stats = scan?.cachedStats
    const legacy = scanWithLegacy?.summary
    return (
      legacy?.vulnerabilities || {
        total: stats?.vulnsTotal ?? 0,
        critical: stats?.vulnsCritical ?? 0,
        high: stats?.vulnsHigh ?? 0,
        medium: stats?.vulnsMedium ?? 0,
        low: stats?.vulnsLow ?? 0,
      }
    )
  }, [scan, scanWithLegacy])

  const startedAt = React.useMemo(
    () => scanWithLegacy?.startedAt || scan?.createdAt,
    [scan, scanWithLegacy]
  )
  const completedAt = React.useMemo(() => scanWithLegacy?.completedAt, [scanWithLegacy])

  const assetCards = React.useMemo(
    () => [
      {
        title: t("cards.websites"),
        value: summary.websites || 0,
        icon: Globe,
        code: "DAT-WEB",
        href: `/scan/history/${scanId}/websites/`,
      },
      {
        title: t("cards.subdomains"),
        value: summary.subdomains || 0,
        icon: Network,
        code: "DAT-SUB",
        href: `/scan/history/${scanId}/subdomain/`,
      },
      {
        title: t("cards.ips"),
        value: summary.ips || 0,
        icon: Server,
        code: "DAT-IP",
        href: `/scan/history/${scanId}/ip-addresses/`,
      },
      {
        title: t("cards.urls"),
        value: summary.endpoints || 0,
        icon: Link2,
        code: "DAT-URL",
        href: `/scan/history/${scanId}/endpoints/`,
      },
      {
        title: t("cards.directories"),
        value: summary.directories || 0,
        icon: FolderOpen,
        code: "DAT-DIR",
        href: `/scan/history/${scanId}/directories/`,
      },
      {
        title: t("cards.screenshots"),
        value: summary.screenshots || 0,
        icon: Camera,
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
