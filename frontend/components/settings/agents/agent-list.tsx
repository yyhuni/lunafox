"use client";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import dynamic from "next/dynamic";
import { useTranslations } from "next-intl";
import { IconCloud, semanticIcons, } from "@/components/icons";
import { Button } from "@/components/ui/button";
import { Dialog, DialogTrigger } from "@/components/ui/dialog";
import { ContentHandoff } from "@/components/shared/loading/content-handoff";
import { getLoadingStructureSlotAttributes } from "@/components/shared/loading/loading-owner";
import { DataTableFacetedFilter, DataTableFacetedFilterGroup, type DataTableFacetedFilterOption } from "@/components/shared/data-table";
import { Card } from "@/components/ui/card";
import { ConfirmDialog } from "@/components/shared/feedback/confirm-dialog";
import { SearchInput } from "@/components/shared/search-input";
import { applyBusinessListControlChange, compileBusinessListFilter, compileBusinessListOrderBy, createBusinessListQuery, setBusinessListPage, type BusinessListFilterCompilerConfig, type BusinessListQuery, type BusinessListSortableFieldConfig, type BusinessListSorting, } from "@/components/shared/data-table/business-list-query";
import { useAgentClusterSummary, useAgentFilterOptions, useAgents, useCreateRegistrationToken, useDeleteAgent, } from "@/hooks/use-agents";
import { useAgentManagementRefresh } from "@/hooks/use-agent-management-refresh";
import { useAgentInstallConnection } from "@/hooks/use-agent-install-connection";
import {
    deferredInteractionUnmountDelayMs,
    useDeferredInteractionMount,
} from "@/hooks/use-deferred-interaction-mount";
import {
  AGENT_HEALTHY_HEALTH_STATES,
  AGENT_WARNING_HEALTH_STATES,
    type AgentStatusDistributionStatus,
} from "@/lib/agent-status-distribution";
import { textRole } from "@/lib/typography";
import { cn } from "@/lib/utils";
import type { Agent, RegistrationTokenResponse } from "@/types/agent.types";
import { AgentCardCompact } from "./agent-card-compact";
import { AgentExpansionSlot } from "./agent-expansion-slot";
import { AgentCardsLoadingState, AgentToolbarLoadingState } from "./agent-list-loading-state";
import { AgentOverviewSection } from "./agent-overview-section";
import { AgentOverviewLoadingState } from "./agent-overview-loading-state";
import { ArchitectureDialog } from "./architecture-dialog";
import { AgentResultsRegion } from "./agent-results-region";
import { AGENT_CARD_GRID_CLASS, AGENT_LIST_TOOLBAR_REGION_SLOT, AGENT_TOOLBAR_ACTIONS_CLASS, AGENT_TOOLBAR_CONTROLS_CLASS, AGENT_TOOLBAR_FILTERS_CLASS, AGENT_TOOLBAR_ROOT_CLASS, AGENT_TOOLBAR_SEARCH_MAX_WIDTH_CLASS, } from "./agent-layout-contract";
const AgentConfigDialog = dynamic(() => import("./agent-dialog").then((mod) => ({
    default: mod.AgentConfigDialog,
})), { ssr: false, loading: () => null });
const AgentInstallDialog = dynamic(() => import("./agent-install-dialog").then((mod) => ({
    default: mod.AgentInstallDialog,
})), { ssr: false, loading: () => null });
const AgentLogDrawer = dynamic(() => import("./agent-log-drawer").then((mod) => ({
    default: mod.AgentLogDrawer,
})), { ssr: false, loading: () => null });
type AgentStatusFilterValue = AgentStatusDistributionStatus;
const AGENT_LIST_PAGE_SIZE = 100;
const AGENT_SEARCH_DEBOUNCE_MS = 300;
const AGENT_DEFAULT_SORTING: BusinessListSorting = { field: "createdAt", direction: "desc" };
const AGENT_FILTER_FIELDS: BusinessListFilterCompilerConfig = {
    search: [
        { field: "displayName", operator: "=" },
        { field: "observedHostname", operator: "=" },
        { field: "connectionIp", operator: "=" },
    ],
    facets: {},
};
const AGENT_SORTABLE_FIELDS: Record<string, BusinessListSortableFieldConfig> = {
    createdAt: { orderBy: "createdAt", firstDirection: "desc" },
};
const AgentIcon = semanticIcons.concept.agent;
function compileEqualsGroup(field: "status" | "healthState", values: readonly string[]) {
    const predicates = values.map((value) => `${field}==\"${value}\"`);
    return predicates.length === 1 ? predicates[0] : `(${predicates.join(" || ")})`;
}
function compileAgentStatusBucket(value: AgentStatusFilterValue) {
    if (value === "offline")
        return 'status=="offline"';
    if (value === "healthy")
        return `(status=="online" && ${compileEqualsGroup("healthState", AGENT_HEALTHY_HEALTH_STATES)})`;
    if (value === "warning")
        return `(status=="online" && ${compileEqualsGroup("healthState", AGENT_WARNING_HEALTH_STATES)})`;
    return '(status!="online" && status!="offline")';
}
function compileAgentStatusFilters(values: AgentStatusFilterValue[]) {
    const selected = Array.from(new Set(values)).filter(Boolean);
    if (selected.length === 0)
        return undefined;
    const clauses = selected.map((value) => compileAgentStatusBucket(value));
    return clauses.length === 1 ? clauses[0] : `(${clauses.join(" || ")})`;
}
function combineFilterClauses(...clauses: Array<string | undefined>) {
    const selected = clauses.filter((clause): clause is string => Boolean(clause));
    return selected.length > 0 ? selected.join(" && ") : undefined;
}
function countOptions(options: Array<{ value: string; count?: number }> | undefined, values: readonly string[]) {
    return (options ?? []).reduce((total, option) => values.includes(option.value) ? total + (option.count ?? 0) : total, 0);
}

function EmptyState({ onOpenInstall }: {
    onOpenInstall: () => void;
}) {
    const t = useTranslations("settings.agents");
    return (<div className="flex flex-col items-center justify-center py-16 text-center">
      <div className="bg-muted mb-4 p-4 rounded-full">
        <AgentIcon className="h-12 text-muted-foreground w-12"/>
      </div>
      <h3 className={cn("mb-2", textRole.panelTitle)}>{t("empty.title")}</h3>
      <p className={cn("max-w-md mb-6", textRole.bodySubtle)}>{t("empty.desc")}</p>
      <Button onClick={onOpenInstall}>{t("empty.cta")}</Button>
    </div>);
}
export function AgentList() {
    const t = useTranslations("settings.agents");
    const tDataTable = useTranslations("dataTable");
    const tPagination = useTranslations("common.pagination");
    const [query, setQuery] = useState<BusinessListQuery>(() => createBusinessListQuery({ pageSize: AGENT_LIST_PAGE_SIZE, sorting: AGENT_DEFAULT_SORTING }));
    const [pageTokens, setPageTokens] = useState<Record<number, string | undefined>>({ 1: undefined });
    const [searchQuery, setSearchQuery] = useState("");
    const [installOpen, setInstallOpen] = useState(false);
    const [selectedAgent, setSelectedAgent] = useState<Agent | null>(null);
    const [configDialogOpen, setConfigDialogOpen] = useState(false);
    const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
    const [agentNodeToDelete, setAgentToDelete] = useState<Agent | null>(null);
    const [token, setToken] = useState<RegistrationTokenResponse | null>(null);
    const installConnection = useAgentInstallConnection(token, installOpen);
    const [logDrawerOpen, setLogDrawerOpen] = useState(false);
    const [logAgent, setLogAgent] = useState<Agent | null>(null);
    const sectionReadyFramesRef = useRef<number[]>([]);
    const sidebarMountOptions = { unmountDelayMs: deferredInteractionUnmountDelayMs };
    const shouldMountInstallDialog = useDeferredInteractionMount(installOpen, sidebarMountOptions);
    const shouldMountConfigDialog = useDeferredInteractionMount(configDialogOpen, sidebarMountOptions);
    const shouldMountLogDrawer = useDeferredInteractionMount(logDrawerOpen, sidebarMountOptions);
    const statusFilters = (query.filters.agentStatus ?? []) as AgentStatusFilterValue[];
    const page = query.pageIndex ?? 1;
    const pageSize = query.pageSize;
    const searchFilter = compileBusinessListFilter({ search: query.search, filters: {} }, AGENT_FILTER_FIELDS);
    const statusFilter = compileAgentStatusFilters(statusFilters);
    const compiledFilter = combineFilterClauses(searchFilter, statusFilter);
    const compiledOrderBy = compileBusinessListOrderBy(query.sorting, AGENT_SORTABLE_FIELDS) ?? "createdAt desc";
    const clusterSummary = useAgentClusterSummary({ refetchInterval: 15000 });
    const { data, isSuccess, refetch } = useAgents({
        pageSize,
        pageToken: query.pageToken,
        filter: compiledFilter,
        orderBy: compiledOrderBy,
    });
    const { data: statusOptionData, refetch: refetchStatusOptions } = useAgentFilterOptions("status");
    const { data: healthStateOptionData, refetch: refetchHealthStateOptions } = useAgentFilterOptions("healthState");
    const refetchAgentFilterOptions = useCallback(() => Promise.all([
        refetchStatusOptions(),
        refetchHealthStateOptions(),
    ]), [refetchHealthStateOptions, refetchStatusOptions]);
    const { isRefreshing: isManualRefreshing, lastRefreshedAt, refresh: handleRefresh } = useAgentManagementRefresh({
        refetchCurrent: refetch,
        refetchSummary: clusterSummary.refetch,
        refetchFilterOptions: refetchAgentFilterOptions,
    });
    const createToken = useCreateRegistrationToken();
    const deleteAgent = useDeleteAgent();
    const [isInitialSectionReady, setIsInitialSectionReady] = useState(false);
    const agentNodes = useMemo(() => data?.results || [], [data?.results]);
    useEffect(() => {
        if (data?.nextPageToken) {
            setPageTokens((tokens) => ({ ...tokens, [page + 1]: data.nextPageToken }));
        }
    }, [data?.nextPageToken, page]);
    useEffect(() => {
        if (isInitialSectionReady || !isSuccess || (!clusterSummary.isSuccess && !clusterSummary.isInitialError && !clusterSummary.data)) {
            return;
        }
        let cancelled = false;
        const pendingFrames = sectionReadyFramesRef.current;
        const first = window.requestAnimationFrame(() => {
            const second = window.requestAnimationFrame(() => {
                if (!cancelled) {
                    setIsInitialSectionReady(true);
                }
            });
            pendingFrames.push(second);
        });
        pendingFrames.push(first);
        return () => {
            cancelled = true;
            while (pendingFrames.length > 0) {
                const frame = pendingFrames.pop();
                if (frame !== undefined)
                    window.cancelAnimationFrame(frame);
            }
        };
    }, [clusterSummary.data, clusterSummary.isInitialError, clusterSummary.isSuccess, isInitialSectionReady, isSuccess]);
    const isInitialSectionLoading = !isInitialSectionReady;
    const statusFilterOptions = useMemo<Array<DataTableFacetedFilterOption<AgentStatusFilterValue>>>(() => {
        const statusOptions = statusOptionData?.results;
        const healthOptions = healthStateOptionData?.results;
        return [
            {
                value: "healthy",
                label: t("health.ok"),
                count: countOptions(healthOptions, AGENT_HEALTHY_HEALTH_STATES),
            },
            {
                value: "warning",
                label: t("health.warning"),
                count: countOptions(healthOptions, AGENT_WARNING_HEALTH_STATES),
            },
            {
                value: "offline",
                label: t("status.offline"),
                count: countOptions(statusOptions, ["offline"]),
            },
            {
                value: "unknown",
                label: t("status.unknown"),
                count: countOptions(statusOptions, ["unknown"]),
            },
        ];
    }, [healthStateOptionData?.results, statusOptionData?.results, t]);
    const hasVisibleAgents = agentNodes.length > 0;
    const isUnfilteredFirstPage = !compiledFilter && page === 1;
    const showExpansionSlot = hasVisibleAgents;
    const handleResetFilters = () => {
        setSearchQuery("");
        setPageTokens({ 1: undefined });
        setQuery((current) => applyBusinessListControlChange(current, { search: undefined, filters: {} }));
    };
    const commitSearch = useCallback((search: string) => {
        const normalizedSearch = search.trim();
        if ((query.search ?? "") === normalizedSearch) {
            return;
        }
        setPageTokens({ 1: undefined });
        setQuery((current) => applyBusinessListControlChange(current, { search: normalizedSearch || undefined }));
    }, [query.search]);
    useEffect(() => {
        const timeout = window.setTimeout(() => commitSearch(searchQuery), AGENT_SEARCH_DEBOUNCE_MS);
        return () => window.clearTimeout(timeout);
    }, [commitSearch, searchQuery]);
    const handleStatusFiltersChange = (values: AgentStatusFilterValue[]) => {
        setPageTokens({ 1: undefined });
        setQuery((current) => applyBusinessListControlChange(current, { filters: { ...current.filters, agentStatus: values } }));
    };
    const handlePageChange = (nextPage: number) => {
        if (nextPage === 1 || pageTokens[nextPage] !== undefined || nextPage < page) {
            setQuery((current) => setBusinessListPage(current, { pageIndex: nextPage, pageToken: pageTokens[nextPage] }));
        }
    };
    const handleGenerateToken = async () => {
        try {
            const response = await createToken.mutateAsync();
            setToken(response);
        }
        catch {
            // handled by hook
        }
    };
    const handleConfigure = (agentNode: Agent) => {
        setSelectedAgent(agentNode);
        setConfigDialogOpen(true);
    };
    const handleDelete = (agentNode: Agent) => {
        setAgentToDelete(agentNode);
        setDeleteDialogOpen(true);
    };
    const handleOpenLogs = (agentNode: Agent) => {
        setLogAgent(agentNode);
        setLogDrawerOpen(true);
    };
    const confirmDelete = async () => {
        if (!agentNodeToDelete)
            return;
        try {
            await deleteAgent.mutateAsync(agentNodeToDelete.id);
            setDeleteDialogOpen(false);
            setAgentToDelete(null);
        }
        catch {
            // handled by hook
        }
    };
    return (<div className="min-w-0 space-y-6">
      <div className="mb-6 space-y-4">
        <ContentHandoff owner="agent-list-overview" layer="section" isLoading={isInitialSectionLoading} skeleton={<AgentOverviewLoadingState />} mountContentWhileLoading>
          <AgentOverviewSection
            summary={clusterSummary.data}
            error={clusterSummary.error}
            isInitialError={clusterSummary.isInitialError}
            isRefetchStale={clusterSummary.isRefetchStale}
            lastSuccessfulAt={clusterSummary.lastSuccessfulAt}
            lastRefreshedAt={lastRefreshedAt}
            isRefreshing={isManualRefreshing}
            onRefresh={handleRefresh}
            onRetry={clusterSummary.refetch}
          />
        </ContentHandoff>

        <ContentHandoff owner="agent-list-toolbar" layer="section" isLoading={isInitialSectionLoading} skeleton={<AgentToolbarLoadingState />}>
          <div {...getLoadingStructureSlotAttributes(AGENT_LIST_TOOLBAR_REGION_SLOT)} className={AGENT_TOOLBAR_ROOT_CLASS}>
            <div className={AGENT_TOOLBAR_CONTROLS_CLASS}>
              <div className={cn("w-full", AGENT_TOOLBAR_SEARCH_MAX_WIDTH_CLASS)}>
                <SearchInput value={searchQuery} onChange={(event) => setSearchQuery(event.target.value)} onKeyDown={(event) => {
            if (event.key === "Enter") {
                event.preventDefault();
                commitSearch(searchQuery);
            }
        }} placeholder={t("overview.searchPlaceholder")}/>
              </div>

              <div className={AGENT_TOOLBAR_FILTERS_CLASS}>
                <DataTableFacetedFilterGroup hasSelectedValues={statusFilters.length > 0} onReset={() => handleStatusFiltersChange([])}>
                  <DataTableFacetedFilter title={t("overview.filterStatus")} values={statusFilters} onValuesChange={handleStatusFiltersChange} options={statusFilterOptions} emptyLabel={t("overview.filterAll")} clearLabel={tDataTable("clearFilter")}/>
                </DataTableFacetedFilterGroup>

              </div>
            </div>

            <div className={AGENT_TOOLBAR_ACTIONS_CLASS}>
              <Dialog open={installOpen} onOpenChange={setInstallOpen}>
                <DialogTrigger render={<Button type="button" variant="surface" size="sm" />}>
                  <AgentIcon className="size-4" />
                  {t("install.openDialog")}
                </DialogTrigger>
                {shouldMountInstallDialog ? <AgentInstallDialog open={installOpen} token={token} connection={installConnection} isGenerating={createToken.isPending} onGenerate={handleGenerateToken}/> : null}
              </Dialog>
              <ArchitectureDialog
                trigger={(
                  <Button type="button" variant="surface" size="sm">
                    <IconCloud className="size-4" />
                    {t("overview.architectureTitle")}
                  </Button>
                )}
              />
            </div>

          </div>
        </ContentHandoff>
      </div>

      <ContentHandoff owner="agent-list-results" layer="section" isLoading={isInitialSectionLoading} skeleton={<AgentCardsLoadingState />} mountContentWhileLoading>
        <AgentResultsRegion>
          {!hasVisibleAgents && isUnfilteredFirstPage ? (<EmptyState onOpenInstall={() => setInstallOpen(true)}/>) : !hasVisibleAgents ? (<Card className="border-dashed px-6 py-14 text-center">
              <h3 className={textRole.panelTitle}>{t("overview.emptyTitle")}</h3>
              <p className={cn("mt-2", textRole.bodySubtle)}>{t("overview.emptyDesc")}</p>
              <Button variant="outline" className="mt-4 self-center" onClick={handleResetFilters}>
                {t("overview.reset")}
              </Button>
            </Card>) : (<div className={AGENT_CARD_GRID_CLASS}>
              {agentNodes.map((agentNode) => (<AgentCardCompact key={agentNode.id} agentNode={agentNode} onConfig={handleConfigure} onDelete={handleDelete} onLogs={handleOpenLogs}/>))}
              {showExpansionSlot ? (<AgentExpansionSlot
                title={t("expansion.title")}
                description={t("expansion.desc")}
                actionLabel={t("expansion.action")}
                onOpenInstall={() => setInstallOpen(true)}
              />) : null}
            </div>)}
          {page > 1 || data?.nextPageToken ? (<div className="flex items-center justify-end gap-2">
              <Button type="button" variant="outline" size="sm" disabled={page <= 1} onClick={() => handlePageChange(page - 1)}>
                {tPagination("previous")}
              </Button>
              <span className={cn("text-muted-foreground", textRole.helperText)}>{tPagination("page", { current: page, total: Math.max(page, data?.nextPageToken ? page + 1 : page) })}</span>
              <Button type="button" variant="outline" size="sm" disabled={!data?.nextPageToken} onClick={() => handlePageChange(page + 1)}>
                {tPagination("next")}
              </Button>
            </div>) : null}
        </AgentResultsRegion>
      </ContentHandoff>

      {shouldMountConfigDialog ? (<AgentConfigDialog open={configDialogOpen} onOpenChange={setConfigDialogOpen} agentNode={selectedAgent}/>) : null}

      <ConfirmDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen} title={t("actions.deleteTitle")} description={t("actions.deleteDesc", { name: agentNodeToDelete?.name ?? "" })} onConfirm={confirmDelete} variant="destructive" loading={deleteAgent.isPending}/>

      {shouldMountLogDrawer ? (<AgentLogDrawer open={logDrawerOpen} onOpenChange={setLogDrawerOpen} agentNode={logAgent}/>) : null}
    </div>);
}
