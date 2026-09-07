"use client"

import * as React from "react"

import { AlertTriangle, Circle, semanticIcons } from "@/components/icons"
import { Spinner } from "@/components/shared/loading/spinner"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  CommandGroup,
  CommandItem,
} from "@/components/ui/command"
import { SegmentedMetricProgress } from "@/components/shared/metrics/segmented-metric-progress"
import { Label } from "@/components/ui/label"
import { useSelectedAgentDetail } from "@/hooks/use-agents"
import { useScanAgentPicker } from "@/hooks/use-scan-agent-picker"
import { getAgentHealthDistributionStatus } from "@/lib/agent-status-distribution"
import { agentName } from "@/lib/resource-name"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import { useTranslations } from "next-intl"
import { ScanSearchablePicker } from "./scan-searchable-picker"

type ScanAgentSelectorProps = {
  value: number | null
  onChange: (agentID: number | null) => void
  disabled?: boolean
}

function isHealthy(state?: string) {
  return getAgentHealthDistributionStatus(state) === "healthy"
}

function getAgentStatusDotClass(status?: string, healthState?: string) {
  if (status !== "online") return "text-muted-foreground"
  return isHealthy(healthState) ? "fill-current text-success" : "fill-current text-warning"
}

const CpuIcon = semanticIcons.metric.cpu
const MemoryIcon = semanticIcons.metric.memory
const DiskIcon = semanticIcons.metric.disk
const TaskSlotsIcon = semanticIcons.metric.taskSlots
const AutomaticAssignmentIcon = semanticIcons.concept.automaticAssignment
const RetryIcon = semanticIcons.action.refresh

function AgentPickerMessage({
  children,
  actionLabel,
  onAction,
  tone = "muted",
}: {
  children: React.ReactNode
  actionLabel?: string
  onAction?: () => void
  tone?: "muted" | "warning" | "error"
}) {
  return (
    <div
      className={cn(
        "flex min-h-12 items-center justify-between gap-3 border-t px-3 py-2",
        tone === "warning" && "text-warning",
        tone === "error" && "text-destructive",
        tone === "muted" && "text-muted-foreground",
      )}
      role={tone === "error" ? "alert" : "status"}
    >
      <span className={cn("flex min-w-0 items-center gap-2", textRole.helperText)}>
        {tone !== "muted" ? <AlertTriangle className="size-4 shrink-0" /> : null}
        <span>{children}</span>
      </span>
      {actionLabel && onAction ? (
        <Button type="button" variant="ghost" size="sm" onClick={onAction}>
          <RetryIcon className="size-4" />
          {actionLabel}
        </Button>
      ) : null}
    </div>
  )
}

export function ScanAgentSelector({ value, onChange, disabled = false }: ScanAgentSelectorProps) {
  const t = useTranslations("scan.agentSelector")
  const tAgents = useTranslations("settings.agents")
  const [isPickerOpen, setIsPickerOpen] = React.useState(false)
  const [searchValue, setSearchValue] = React.useState("")
  const listRef = React.useRef<HTMLDivElement>(null)
  const loadMoreRef = React.useRef<HTMLDivElement>(null)
  const consumedLoadMoreRangeRef = React.useRef<string | null>(null)
  const picker = useScanAgentPicker({
    open: isPickerOpen,
    search: searchValue,
  })
  const selectedResourceName = value === null ? null : agentName(value)
  const selectedAgentDetail = useSelectedAgentDetail(selectedResourceName)
  const agents = picker.agents
  const selectedAgent = selectedAgentDetail.data
  const selectedAgentUnavailable = value !== null && selectedAgentDetail.isInitialError
  const pickerCanLoadNextPage = picker.canLoadNextPage
  const pickerHasNextPage = picker.hasNextPage
  const pickerLoadNextPage = picker.loadNextPage
  const pickerLoadedPageCount = picker.loadedPageCount
  const pickerNormalizedSearch = picker.normalizedSearch

  React.useEffect(() => {
    if (!isPickerOpen) {
      consumedLoadMoreRangeRef.current = null
      return
    }
    const root = listRef.current
    const target = loadMoreRef.current
    if (!root || !target) return

    // Re-observe after each appended page so one stale visibility result cannot drain later pages.
    const loadedRangeKey = `${pickerNormalizedSearch}:${pickerLoadedPageCount}`

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (!entry?.isIntersecting || !pickerCanLoadNextPage) return
        if (consumedLoadMoreRangeRef.current === loadedRangeKey) return

        consumedLoadMoreRangeRef.current = loadedRangeKey
        void pickerLoadNextPage()
      },
      { root, rootMargin: "0px 0px 96px 0px" },
    )
    observer.observe(target)
    return () => observer.disconnect()
  }, [
    isPickerOpen,
    pickerCanLoadNextPage,
    pickerHasNextPage,
    pickerLoadNextPage,
    pickerLoadedPageCount,
    pickerNormalizedSearch,
  ])

  const handlePickerOpenChange = React.useCallback((open: boolean) => {
    setIsPickerOpen(open)
    if (!open) setSearchValue("")
  }, [])

  const selectedAgentName = selectedAgent?.displayName || selectedAgent?.name || (
    value === null ? t("automatic") : selectedResourceName
  )

  return (
    <div className="space-y-2">
      <Label htmlFor="scan-agent-selector">{t("label")}</Label>
      <ScanSearchablePicker
        id="scan-agent-selector"
        ariaLabel={t("label")}
        disabled={disabled}
        onOpenChange={handlePickerOpenChange}
        searchValue={searchValue}
        onSearchValueChange={setSearchValue}
        shouldFilter={false}
        listRef={listRef}
        listBusy={picker.isInitialLoading || picker.isFetchingNextPage || picker.isRefreshing}
        trigger={
          <span className="flex min-w-0 flex-1 items-center gap-2 overflow-hidden">
            {value === null ? (
              <span className="flex size-8 shrink-0 items-center justify-center text-muted-foreground">
                <AutomaticAssignmentIcon className="size-6" />
              </span>
            ) : (
              <Circle className={cn("size-2 shrink-0", getAgentStatusDotClass(selectedAgent?.status, selectedAgent?.health?.state))} />
            )}
            <span className="min-w-0 flex-1 overflow-hidden">
              <span className={cn("block truncate", textRole.navLabel)}>{selectedAgentName}</span>
              <span className={cn("block truncate", textRole.helperText)}>
                {value === null
                  ? t("automaticHint")
                  : selectedAgentUnavailable
                    ? t("unavailableHint")
                    : t("selectedHint")}
              </span>
            </span>
          </span>
        }
        searchPlaceholder={t("searchPlaceholder")}
      >
        <CommandGroup>
          <CommandItem value="automatic" keywords={[t("automatic")]} onSelect={() => onChange(null)} className="min-h-12 gap-3 py-2">
            <span className="flex size-8 shrink-0 items-center justify-center text-muted-foreground">
              <AutomaticAssignmentIcon className="size-6" />
            </span>
            <span className="min-w-0 flex-1">
              <span className={cn("block truncate", textRole.navLabel)}>{t("automatic")}</span>
              <span className={cn("block truncate", textRole.helperText)}>{t("automaticHint")}</span>
            </span>
            <Badge variant="success" className="shrink-0">{t("recommended")}</Badge>
          </CommandItem>
          {picker.isInitialLoading ? (
            <AgentPickerMessage>
              <Spinner className="size-4" role="presentation" aria-hidden="true" />
              {t("loading")}
            </AgentPickerMessage>
          ) : picker.isInitialError ? (
            <AgentPickerMessage tone="error" actionLabel={t("retry")} onAction={() => void picker.retryCurrentGeneration()}>
              {t("loadFailed")}
            </AgentPickerMessage>
          ) : null}
          {agents.map((agent) => {
            const heartbeat = agent.status === "online" ? agent.heartbeat : undefined
            const healthy = isHealthy(agent.health?.state)
            const healthLabel = agent.status !== "online"
              ? t("offline")
              : healthy
                ? tAgents("health.ok")
                : tAgents("health.warning")
            return (
              <CommandItem
                key={agent.id}
                value={String(agent.id)}
                keywords={[agent.displayName || agent.name, agent.name, agent.observedHostname ?? "", agent.connectionIp ?? ""]}
                onSelect={() => onChange(agent.id)}
                className="min-h-24 gap-3 px-3 py-3"
              >
                <Circle className={cn("size-2 shrink-0", getAgentStatusDotClass(agent.status, agent.health?.state))} />
                <span className="flex min-w-0 flex-1 flex-col gap-3 sm:flex-row sm:items-stretch sm:gap-4">
                  <span className="flex min-w-0 items-center justify-between gap-2 sm:w-36 sm:shrink-0">
                    <span className={cn("min-w-0 flex-1 truncate", textRole.navLabel)} title={agent.displayName || agent.name}>{agent.displayName || agent.name}</span>
                    <Badge variant={agent.status !== "online" ? "secondary" : healthy ? "success" : "warning"} className="shrink-0">
                      {healthLabel}
                    </Badge>
                  </span>
                  {heartbeat ? (
                    <span className="grid min-w-0 flex-1 grid-cols-2 gap-x-4 gap-y-3 border-t pt-3 sm:grid-cols-4 sm:border-t-0 sm:border-l sm:pl-4 sm:pt-0">
                      <SegmentedMetricProgress variant="inline" label={t("cpu")} value={heartbeat.cpu} threshold={agent.cpuThreshold} icon={<CpuIcon />} />
                      <SegmentedMetricProgress variant="inline" label={t("memory")} value={heartbeat.mem} threshold={agent.memThreshold} icon={<MemoryIcon />} />
                      <SegmentedMetricProgress variant="inline" label={t("disk")} value={heartbeat.disk} threshold={agent.diskThreshold} icon={<DiskIcon />} />
                      <SegmentedMetricProgress
                        variant="inline"
                        label={tAgents("metrics.usedTaskSlots")}
                        value={agent.maxTasks > 0 ? (heartbeat.taskSlotsUsed / agent.maxTasks) * 100 : 0}
                        threshold={100}
                        detail={`${heartbeat.taskSlotsUsed}/${agent.maxTasks}`}
                        icon={<TaskSlotsIcon />}
                      />
                    </span>
                  ) : agent.status === "offline" ? (
                    <span className="grid min-w-0 flex-1 grid-cols-2 gap-x-4 gap-y-3 border-t pt-3 sm:grid-cols-4 sm:border-t-0 sm:border-l sm:pl-4 sm:pt-0">
                      <SegmentedMetricProgress
                        variant="inline"
                        label={t("cpu")}
                        value={0}
                        detail="—"
                        valueTone="neutral"
                        barTone="neutral"
                        icon={<CpuIcon />}
                      />
                      <SegmentedMetricProgress
                        variant="inline"
                        label={t("memory")}
                        value={0}
                        detail="—"
                        valueTone="neutral"
                        barTone="neutral"
                        icon={<MemoryIcon />}
                      />
                      <SegmentedMetricProgress
                        variant="inline"
                        label={t("disk")}
                        value={0}
                        detail="—"
                        valueTone="neutral"
                        barTone="neutral"
                        icon={<DiskIcon />}
                      />
                      <SegmentedMetricProgress
                        variant="inline"
                        label={tAgents("metrics.usedTaskSlots")}
                        value={0}
                        detail="0/0"
                        valueTone="neutral"
                        barTone="neutral"
                        icon={<TaskSlotsIcon />}
                      />
                    </span>
                  ) : (
                    <span className={cn("border-t pt-3 sm:border-t-0 sm:border-l sm:pl-4 sm:pt-0", textRole.helperText)}>
                      {agent.status === "online" ? t("noHeartbeat") : t("offline")}
                    </span>
                  )}
                </span>
              </CommandItem>
            )
          })}
          {!picker.isInitialLoading && !picker.isInitialError && agents.length === 0 ? (
            <AgentPickerMessage>{t("searchEmpty")}</AgentPickerMessage>
          ) : null}
          {picker.isRefreshError ? (
            <AgentPickerMessage tone="warning" actionLabel={t("retry")} onAction={() => void picker.retryCurrentGeneration()}>
              {t("refreshFailed")}
            </AgentPickerMessage>
          ) : null}
          {picker.isNextPageError ? (
            <AgentPickerMessage tone="error" actionLabel={t("retry")} onAction={() => void picker.retryNextPage()}>
              {t("loadMoreFailed")}
            </AgentPickerMessage>
          ) : null}
          {picker.isFetchingNextPage ? (
            <AgentPickerMessage>
              <Spinner className="size-4" role="presentation" aria-hidden="true" />
              {t("loadingMore")}
            </AgentPickerMessage>
          ) : null}
          {picker.hasNextPage ? <div ref={loadMoreRef} className="h-px" aria-hidden="true" /> : null}
        </CommandGroup>
      </ScanSearchablePicker>
      <p className="text-xs text-muted-foreground">{t("hint")}</p>
    </div>
  )
}
