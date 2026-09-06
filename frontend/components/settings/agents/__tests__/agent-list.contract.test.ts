import { describe, expect, it } from "vitest"
import { existsSync, readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/settings/agents/agent-list.tsx"), "utf8")
const overviewSource = readFileSync(path.resolve(process.cwd(), "components/settings/agents/agent-overview-section.tsx"), "utf8")
const overviewLoadingSource = readFileSync(path.resolve(process.cwd(), "components/settings/agents/agent-overview-loading-state.tsx"), "utf8")
const cardLoadingSource = readFileSync(path.resolve(process.cwd(), "components/settings/agents/agent-card-compact-loading-state.tsx"), "utf8")
const listLoadingSource = readFileSync(path.resolve(process.cwd(), "components/settings/agents/agent-list-loading-state.tsx"), "utf8")
const resultsRegionSource = readFileSync(path.resolve(process.cwd(), "components/settings/agents/agent-results-region.tsx"), "utf8")
const enMessages = JSON.parse(readFileSync(path.resolve(process.cwd(), "messages/en.json"), "utf8"))
const zhMessages = JSON.parse(readFileSync(path.resolve(process.cwd(), "messages/zh.json"), "utf8"))
const toolbarSource = source.slice(source.indexOf('owner="agent-list-toolbar"'), source.indexOf('owner="agent-list-results"'))

describe("agent-list contract", () => {
  it("preserves the Agent list section and deferred interaction owners", () => {
    expect(source).toContain("export function AgentList")
    expect(source).toContain('import { ContentHandoff } from "@/components/shared/loading/content-handoff"')
    expect(source).toContain("deferredInteractionUnmountDelayMs")
    expect(source).toContain("useDeferredInteractionMount(installOpen, sidebarMountOptions)")
    expect(source).toContain("useDeferredInteractionMount(configDialogOpen, sidebarMountOptions)")
    expect(source).toContain("useDeferredInteractionMount(logDrawerOpen, sidebarMountOptions)")
  })

  it("uses shared status entrypoints for the compact cluster summary", () => {
    expect(overviewSource).toContain("getStatusToneTextClass")
    expect(overviewSource).toContain("getStatusToneBgClass")
    expect(overviewSource).toContain("getStatusToneSurfaceClass")
    expect(source).toContain("useAgentClusterSummary")
    expect(source).not.toContain("buildAgentClusterHealthSummary")
    expect(source).not.toContain("buildAgentExecutionCapacity")
    expect(source).not.toContain("text-emerald-600")
    expect(source).not.toContain("text-amber-500")
    expect(source).not.toContain("text-destructive")
    expect(source).not.toContain("bg-destructive/80")
    expect(source).not.toContain("sr-only")
  })

  it("registers every authoritative summary bucket label in both locales", () => {
    expect(enMessages.settings.agents.stats).toMatchObject({
      healthy: "Healthy",
      warning: "Warning",
      offline: "Offline",
      unknown: "Unknown",
    })
    expect(zhMessages.settings.agents.stats).toMatchObject({
      healthy: "健康",
      warning: "预警",
      offline: "离线",
      unknown: "未知",
    })
  })

  it("renders the permanent scheme C summary from one overview owner", () => {
    expect(overviewSource).toContain("AGENT_CLUSTER_SUMMARY_ROOT_CLASS")
    expect(overviewSource).toContain("AGENT_CLUSTER_SUMMARY_STATE_CLASS")
    expect(overviewSource).toContain("AGENT_CLUSTER_CAPACITY_SUMMARY_CLASS")
    expect(overviewSource).toContain("function ClusterNodeSummary")
    expect(overviewSource).toContain("function ClusterAttentionSummary")
    expect(overviewSource).toContain("function HealthyNodeSummary")
    expect(overviewSource).toContain("function CompactCapacitySummary")
    expect(overviewSource).toContain("function AgentOverviewLayout")
    expect(overviewSource).toContain("AppErrorState")
    expect(overviewSource).toContain("SummaryStaleAlert")
    expect(overviewSource).not.toContain("OverviewStatCard")
    expect(overviewSource).not.toContain("ExecutionCapacityCard")
    expect(overviewSource).not.toContain("ShortcutCard")
    expect(overviewLoadingSource).toContain("<AgentOverviewSection loading />")
    expect(overviewSource).toContain('data-testid={loading ? "agent-overview-loading-state" : "agent-overview-section"}')
  })

  it("keeps authoritative configured-capacity and overcommit semantics", () => {
    expect(overviewSource).toContain('key: "occupiedSlots"')
    expect(overviewSource).toContain('key: "availableSlots"')
    expect(overviewSource).toContain('key: "unavailableSlots"')
    expect(overviewSource).toContain('key: "unavailableSlots"')
    expect(overviewSource).toContain('t("overview." + part.key)')
    expect(overviewSource).toContain("visualTotal")
    expect(overviewSource).toContain("capacity.configuredSlots")
    expect(overviewSource).toContain("capacity.overcommittedSlots")
    expect(overviewSource).not.toContain('t("overview.runningTasks")')
    expect(source).toContain("summary={clusterSummary.data}")
  })

  it("keeps the title and refresh control in the overview header", () => {
    expect(overviewSource).toContain("AGENT_OVERVIEW_HEADER_CLASS")
    expect(overviewSource).toContain("AGENT_OVERVIEW_TITLE_GROUP_CLASS")
    expect(overviewSource).toContain('className={cn(textRole.monoLabel, "shrink-0")}')
    expect(overviewSource).toContain("PageRefreshStatusButton")
    expect(overviewSource).toContain('t("overview.refresh")')
  })

  it("keeps reset action only in the empty-state recovery flow", () => {
    expect(source.match(/t\("overview\.reset"\)/g)).toHaveLength(1)
  })

  it("does not expose a critical filter bucket in the Agent toolbar", () => {
    expect(source).not.toContain('t("stats.critical")')
    expect(source).not.toContain('value="critical"')
    expect(source).not.toContain("stats.critical")
  })

  it("keeps search and status filtering while putting the two actions on the toolbar right", () => {
    expect(source).toContain('from "@/components/shared/search-input"')
    expect(source).toContain('from "@/components/shared/data-table"')
    expect(source).toContain("<SearchInput")
    expect(source).toContain('placeholder={t("overview.searchPlaceholder")}')
    expect(source).toContain("DataTableFacetedFilterGroup")
    expect(toolbarSource).toContain("AGENT_TOOLBAR_ROOT_CLASS")
    expect(toolbarSource).toContain("AGENT_TOOLBAR_CONTROLS_CLASS")
    expect(toolbarSource).toContain("AGENT_TOOLBAR_FILTERS_CLASS")
    expect(toolbarSource).toContain("AGENT_TOOLBAR_ACTIONS_CLASS")
    expect(toolbarSource).toContain("<Dialog open={installOpen}")
    expect(toolbarSource).toContain("<ArchitectureDialog")
    expect(toolbarSource).toContain('size="sm"')
    expect(toolbarSource).not.toContain('size="action-card"')
    expect(source).toContain("hasSelectedValues={statusFilters.length > 0}")
    expect(source).toContain("onReset={() => handleStatusFiltersChange([])}")
    expect(source).toContain('title={t("overview.filterStatus")}')
    expect(source).toContain("type AgentStatusFilterValue = AgentStatusDistributionStatus")
    expect(source).not.toContain("viewMode")
    expect(source).not.toContain("<Select ")
  })

  it("keeps the global summary authoritative and the paginated collection row-owned", () => {
    const listQueryStart = source.indexOf("const { data, isSuccess, refetch } = useAgents({")
    const listQuerySource = source.slice(listQueryStart, source.indexOf("const { data: statusOptionData }"))

    expect(source).toContain("useAgentClusterSummary({ refetchInterval: 15000 })")
    expect(listQuerySource).toContain("pageToken: query.pageToken")
    expect(listQuerySource).toContain("filter: compiledFilter")
    expect(source).not.toContain("overviewData")
    expect(source).not.toContain("isOverviewComplete")
    expect(source).not.toContain("buildAgentStatusDistribution")
    expect(source).toContain("const isUnfilteredFirstPage = !compiledFilter && page === 1")
    expect(source).toContain("useAgentManagementRefresh")
    expect(source).toContain("refetchCurrent: refetch")
    expect(source).toContain("refetchSummary: clusterSummary.refetch")
    expect(source).toContain("refetchFilterOptions: refetchAgentFilterOptions")
    expect(source).toContain("refetchStatusOptions()")
    expect(source).toContain("refetchHealthStateOptions()")
  })

  it("removes prototype controls and their URL behavior from production", () => {
    expect(source).not.toContain("clusterHealthDemo")
    expect(source).not.toContain("useSearchParams")
    expect(overviewSource).not.toContain("Prototype")
    expect(existsSync(path.resolve(process.cwd(), "components/settings/agents/agent-cluster-health-prototype.tsx"))).toBe(false)
  })

  it("shows one actionable expansion slot after every non-empty result set", () => {
    expect(source).toContain('import { AgentExpansionSlot } from "./agent-expansion-slot"')
    expect(source).toContain("const showExpansionSlot = hasVisibleAgents")
    expect(source).toContain("showExpansionSlot ? (<AgentExpansionSlot")
    expect(source).toContain('onOpenInstall={() => setInstallOpen(true)}')
  })

  it("keeps overview and toolbar loading geometry with the resolved owners", () => {
    expect(source).toContain("AgentOverviewSection")
    expect(source).toContain('from "./agent-overview-loading-state"')
    expect(source).toContain("skeleton={<AgentOverviewLoadingState />}")
    expect(overviewSource).toContain('getLoadingStructureSlotAttributes(AGENT_LIST_OVERVIEW_REGION_SLOT)')
    expect(overviewLoadingSource).toContain("return <AgentOverviewSection loading />")
    expect(listLoadingSource).toContain("AGENT_TOOLBAR_ACTIONS_CLASS")
    expect(listLoadingSource).toContain("AGENT_EXPANSION_SLOT_MIN_HEIGHT_CLASS")
    expect(listLoadingSource).toMatch(/<ActionSkeleton\s+size="action-card"/)
    const toolbarActionsLoadingSource = listLoadingSource.slice(listLoadingSource.lastIndexOf("AGENT_TOOLBAR_ACTIONS_CLASS"))
    expect(toolbarActionsLoadingSource.match(/<ActionSkeleton size="sm"/g)).toHaveLength(2)
    expect(listLoadingSource).toContain('getLoadingStructureSlotAttributes(AGENT_LIST_TOOLBAR_REGION_SLOT)')
  })

  it("routes initial Agent loading through synchronized section owners", () => {
    for (const owner of ["agent-list-overview", "agent-list-toolbar", "agent-list-results"]) {
      expect(source).toMatch(new RegExp(`owner="${owner}"\\s+layer="section"`))
    }
    expect(source.match(/isLoading=\{isInitialSectionLoading\}/g)).toHaveLength(3)
    expect(source.match(/mountContentWhileLoading/g)).toHaveLength(2)
    expect(source).toContain("const [isInitialSectionReady, setIsInitialSectionReady] = useState(false)")
    expect(source).not.toContain('owner="agent-list-page"')
    expect(source).not.toContain("AgentPageSkeleton")
    expect(source).toContain("<AgentResultsRegion>")
    expect(listLoadingSource).toContain("<AgentResultsRegion>")
    expect(resultsRegionSource).toContain('getLoadingStructureSlotAttributes(AGENT_LIST_RESULTS_PRIMARY_REGION_SLOT)')
  })

  it("keeps loading visuals lightweight and free of resolved dependencies", () => {
    for (const loadingSource of [overviewLoadingSource, cardLoadingSource, listLoadingSource]) {
      expect(loadingSource).not.toContain("@/hooks/")
      expect(loadingSource).not.toContain("next/dynamic")
      expect(loadingSource).not.toContain("framer-motion")
      expect(loadingSource).not.toContain("ArchitectureDialog")
      expect(loadingSource).not.toContain("AgentInstallDialog")
      expect(loadingSource).not.toContain("AgentLogDrawer")
    }
    expect(cardLoadingSource).toContain("<SegmentedMetricProgress key={metricIndex} loading />")
  })
})
