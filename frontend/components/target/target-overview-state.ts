import React from "react"
import { useLocale, useTranslations } from "next-intl"

import { useTarget } from "@/hooks/use-targets"
import { useScheduledScans } from "@/hooks/use-scheduled-scans"

import type { TargetDetail } from "@/types/target.types"

interface TargetOverviewStateOptions {
  targetId: number
}

export function useTargetOverviewState({ targetId }: TargetOverviewStateOptions) {
  const t = useTranslations("pages.targetDetail.overview")
  const tInitiate = useTranslations("scan.initiate")
  const locale = useLocale()

  const [scanDialogOpen, setScanDialogOpen] = React.useState(false)

  const { data: target, isLoading, error } = useTarget(targetId)
  const { data: scheduledScansData, isLoading: isLoadingScans } = useScheduledScans({
    targetId,
    pageSize: 5,
  })

  const targetSummary = (target as TargetDetail | undefined)?.summary

  const summary = React.useMemo(
    () => ({
      subdomains: targetSummary?.subdomains ?? 0,
      websites: targetSummary?.websites ?? 0,
      endpoints: targetSummary?.endpoints ?? 0,
      ips: targetSummary?.ips ?? 0,
      directories: targetSummary?.directories ?? 0,
      screenshots: targetSummary?.screenshots ?? 0,
    }),
    [targetSummary]
  )

  const vulnSummary = React.useMemo(
    () => targetSummary?.vulnerabilities || { total: 0, critical: 0, high: 0, medium: 0, low: 0 },
    [targetSummary]
  )

  const assetCards = React.useMemo(
    () => [
      {
        title: t("cards.websites"),
        value: summary.websites || 0,
        iconKey: "websites",
        code: "DAT-WEB",
        href: `/target/${targetId}/websites/`,
      },
      {
        title: t("cards.subdomains"),
        value: summary.subdomains || 0,
        iconKey: "subdomains",
        code: "DAT-SUB",
        href: `/target/${targetId}/subdomain/`,
      },
      {
        title: t("cards.ips"),
        value: summary.ips || 0,
        iconKey: "ips",
        code: "DAT-IP",
        href: `/target/${targetId}/ip-addresses/`,
      },
      {
        title: t("cards.urls"),
        value: summary.endpoints || 0,
        iconKey: "urls",
        code: "DAT-URL",
        href: `/target/${targetId}/endpoints/`,
      },
      {
        title: t("cards.directories"),
        value: summary.directories || 0,
        iconKey: "directories",
        code: "DAT-DIR",
        href: `/target/${targetId}/directories/`,
      },
      {
        title: t("cards.screenshots"),
        value: summary.screenshots || 0,
        iconKey: "screenshots",
        code: "DAT-SCR",
        href: `/target/${targetId}/screenshots/`,
      },
    ],
    [summary, targetId, t]
  )

  const scheduledScans = React.useMemo(
    () => scheduledScansData?.results || [],
    [scheduledScansData?.results]
  )
  const totalScheduledScans = React.useMemo(
    () => scheduledScansData?.total || 0,
    [scheduledScansData?.total]
  )
  const enabledScans = React.useMemo(
    () => scheduledScans.filter((scan) => scan.isEnabled),
    [scheduledScans]
  )

  const nextExecution = React.useMemo(() => {
    const enabledWithNextRun = enabledScans.filter((scan) => scan.nextRunTime)
    if (enabledWithNextRun.length === 0) return null

    const sorted = enabledWithNextRun.toSorted(
      (a, b) => new Date(a.nextRunTime!).getTime() - new Date(b.nextRunTime!).getTime()
    )
    return sorted[0]
  }, [enabledScans])

  const formatDate = React.useCallback(
    (dateString: string | undefined): string => {
      if (!dateString) return "-"
      return new Date(dateString).toLocaleString(locale === "zh" ? "zh-CN" : "en-US", {
        year: "numeric",
        month: "short",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
      })
    },
    [locale]
  )

  const formatShortDate = React.useCallback(
    (dateString: string | undefined, todayText: string, tomorrowText: string): string => {
      if (!dateString) return "-"
      const date = new Date(dateString)
      const now = new Date()
      const tomorrow = new Date(now)
      tomorrow.setDate(tomorrow.getDate() + 1)

      const localeStr = locale === "zh" ? "zh-CN" : "en-US"

      if (date.toDateString() === now.toDateString()) {
        return `${todayText} ${date.toLocaleTimeString(localeStr, {
          hour: "2-digit",
          minute: "2-digit",
        })}`
      }

      if (date.toDateString() === tomorrow.toDateString()) {
        return `${tomorrowText} ${date.toLocaleTimeString(localeStr, {
          hour: "2-digit",
          minute: "2-digit",
        })}`
      }

      return date.toLocaleString(localeStr, {
        month: "short",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
      })
    },
    [locale]
  )

  return {
    t,
    tInitiate,
    locale,
    targetId,
    target,
    isLoading,
    error,
    isLoadingScans,
    scanDialogOpen,
    setScanDialogOpen,
    assetCards,
    scheduledScans,
    totalScheduledScans,
    enabledScans,
    nextExecution,
    vulnSummary,
    formatDate,
    formatShortDate,
  }
}

export type TargetOverviewState = ReturnType<typeof useTargetOverviewState>
