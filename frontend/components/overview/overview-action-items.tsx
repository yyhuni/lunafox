"use client"

import Link from "next/link"
import { useLocale, useTranslations } from "next-intl"
import { useAgents } from "@/hooks/use-agents"
import { useDatabaseHealth } from "@/hooks/use-database-health"
import { useAssetStatistics } from "@/hooks/use-overview"
import { useScanStatistics } from "@/hooks/use-scans"
import { Button } from "@/components/ui/button"
import { semanticIcons, type Icon } from "@/components/icons"
import { OVERVIEW_ACTION_ITEM_CLASS, OverviewSectionPanel } from "@/components/overview/overview-section-layouts"
import { AppErrorState } from "@/components/shared/feedback/app-error-state"
import { buildAgentSummary } from "@/components/overview/agent-globe-card-state"
import { normalizeError } from "@/lib/errors/normalize-error"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

const VulnerabilityIcon = semanticIcons.concept.vulnerability
const ScanIcon = semanticIcons.concept.scan
const AgentIcon = semanticIcons.concept.agent
const DatabaseIcon = semanticIcons.concept.database
const ViewIcon = semanticIcons.action.view
const SuccessIcon = semanticIcons.status.success

type PendingItem = {
  id: string
  Icon: Icon
  sourceUpdatedAt: string | number | null
  title: string
  description: string
  href: string
  actionLabel: string
}

function formatActionItemTime(
  value: string | number | null,
  locale: string,
  unavailableLabel: string,
) {
  if (!value) return unavailableLabel

  const date = new Date(value)

  if (Number.isNaN(date.getTime())) return unavailableLabel

  return new Intl.DateTimeFormat(locale === "zh" ? "zh-CN" : "en-US", {
    hour: "2-digit",
    hour12: false,
    minute: "2-digit",
  }).format(date)
}

// This adapter intentionally has no persistence or mutation semantics. It will be replaced by the server-owned action-items read model.
function buildPendingItems({
  criticalVulnerabilities,
  criticalVulnerabilitiesSourceUpdatedAt,
  failedScans,
  failedScansSourceUpdatedAt,
  agentsNeedingAttention,
  agentsNeedingAttentionSourceUpdatedAt,
  oldestPendingTaskAgeSec,
  databaseBacklogSourceUpdatedAt,
  t,
}: {
  criticalVulnerabilities: number
  criticalVulnerabilitiesSourceUpdatedAt: string | number | null
  failedScans: number
  failedScansSourceUpdatedAt: string | number | null
  agentsNeedingAttention: number
  agentsNeedingAttentionSourceUpdatedAt: string | number | null
  oldestPendingTaskAgeSec: number
  databaseBacklogSourceUpdatedAt: string | number | null
  t: ReturnType<typeof useTranslations<"overview.operational.pending">>
}): PendingItem[] {
  const items: PendingItem[] = []

  if (criticalVulnerabilities > 0) {
    items.push({
      id: "critical-vulnerabilities",
      Icon: VulnerabilityIcon,
      sourceUpdatedAt: criticalVulnerabilitiesSourceUpdatedAt,
      title: t("criticalVulnerabilities"),
      description: t("criticalVulnerabilitiesDescription", { count: criticalVulnerabilities }),
      href: "/vulnerabilities/",
      actionLabel: t("viewVulnerabilities"),
    })
  }

  if (failedScans > 0) {
    items.push({
      id: "scan-failures",
      Icon: ScanIcon,
      sourceUpdatedAt: failedScansSourceUpdatedAt,
      title: t("scanFailures"),
      description: t("scanFailuresDescription", { count: failedScans }),
      href: "/scan/history/",
      actionLabel: t("viewScans"),
    })
  }

  if (agentsNeedingAttention > 0) {
    items.push({
      id: "agent-health",
      Icon: AgentIcon,
      sourceUpdatedAt: agentsNeedingAttentionSourceUpdatedAt,
      title: t("agentHealth"),
      description: t("agentHealthDescription", { count: agentsNeedingAttention }),
      href: "/settings/agents/",
      actionLabel: t("viewAgents"),
    })
  }

  if (oldestPendingTaskAgeSec >= 120) {
    items.push({
      id: "database-backlog",
      Icon: DatabaseIcon,
      sourceUpdatedAt: databaseBacklogSourceUpdatedAt,
      title: t("databaseHealth"),
      description: t("databaseHealthDescription", { minutes: Math.ceil(oldestPendingTaskAgeSec / 60) }),
      href: "/settings/database-health/",
      actionLabel: t("viewDatabase"),
    })
  }

  return items.slice(0, 3)
}

export function OverviewActionItems() {
  const t = useTranslations("overview.operational.pending")
  const locale = useLocale()
  const assets = useAssetStatistics()
  const scans = useScanStatistics()
  const agents = useAgents({ pageSize: 100, orderBy: "createdAt desc" })
  const database = useDatabaseHealth()
  const agentSummary = buildAgentSummary(agents.data?.results ?? [])
  const sourceError = assets.error ?? scans.error ?? agents.error ?? database.error
  const items = buildPendingItems({
    criticalVulnerabilities: assets.data?.vulnBySeverity.critical ?? 0,
    criticalVulnerabilitiesSourceUpdatedAt: assets.data?.updatedAt ?? assets.dataUpdatedAt,
    failedScans: scans.data?.failed ?? 0,
    failedScansSourceUpdatedAt: scans.dataUpdatedAt,
    agentsNeedingAttention: agentSummary.highLoad + agentSummary.offline,
    agentsNeedingAttentionSourceUpdatedAt: agents.dataUpdatedAt,
    oldestPendingTaskAgeSec: database.data?.coreSignals.oldestPendingTaskAgeSec ?? 0,
    databaseBacklogSourceUpdatedAt: database.data?.observedAt ?? database.dataUpdatedAt,
    t,
  })

  return (
    <OverviewSectionPanel
      title={t("title")}
      className="h-full"
      contentClassName="pt-1"
    >
      {items.length > 0 ? (
        <div>
          {items.map((item) => {
            const { Icon } = item
            const timeLabel = formatActionItemTime(item.sourceUpdatedAt, locale, t("timeUnavailable"))

            return (
              <div key={item.id} className={OVERVIEW_ACTION_ITEM_CLASS}>
                <Icon aria-hidden="true" className="size-3.5 shrink-0 text-muted-foreground" />
                <time className={cn("w-12 shrink-0 tabular-nums", textRole.metadataLabel)}>{timeLabel}</time>
                <div className="min-w-0 flex-1">
                  <p className={cn("truncate", textRole.bodyStrong)}>{item.title}</p>
                  <p className={cn("mt-1 truncate", textRole.caption)}>{item.description}</p>
                </div>
                <div className="flex shrink-0 items-center">
                  <Button size="sm" variant="outline" render={<Link href={item.href} />}>
                    <ViewIcon aria-hidden="true" />
                    <span className="hidden lg:inline">{item.actionLabel}</span>
                    <span className="sr-only lg:hidden">{item.actionLabel}</span>
                  </Button>
                </div>
              </div>
            )
          })}
        </div>
      ) : sourceError ? (
        <AppErrorState
          error={normalizeError(sourceError, { notFoundKind: "unexpected-error" })}
          title={t("loadFailed")}
          description={t("loadFailedDescription")}
          onRetry={() => Promise.all([assets.refetch(), scans.refetch(), agents.refetch(), database.refetch()])}
          variant="section"
          className="min-h-48 py-4"
        />
      ) : (
        <div className="flex min-h-48 flex-col items-center justify-center gap-3 text-center">
          <span className="flex size-10 items-center justify-center radius-control bg-muted text-muted-foreground">
            <SuccessIcon aria-hidden="true" className="size-5" />
          </span>
          <div className="space-y-1">
            <p className={textRole.bodyStrong}>{t("empty")}</p>
            <p className={textRole.caption}>{t("emptyDescription")}</p>
          </div>
        </div>
      )}
    </OverviewSectionPanel>
  )
}
