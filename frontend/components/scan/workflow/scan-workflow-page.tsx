"use client"

import * as React from "react"
import { useLocale, useTranslations } from "next-intl"

import { semanticIcons } from "@/components/icons"
import { ScanConfigurationWorkspace } from "@/components/scan/scan-configuration-workspace"
import { BusinessListDataTable } from "@/components/shared/data-table"
import {
  getCurrentCursorNextPageToken,
  getCursorPaginationNavigation,
  getCursorPageTransition,
} from "@/components/shared/data-table/business-list-query"
import { useCursorPaginationScopeChange } from "@/components/shared/data-table/use-cursor-pagination-scope"
import { SimpleSearchToolbar } from "@/components/shared/data-table/simple-search-toolbar"
import { ContentHandoff } from "@/components/shared/loading/content-handoff"
import { ActionSkeleton } from "@/components/shared/loading/action-skeleton"
import {
  getLoadingOwnerAttributes,
  getLoadingStructureSlotAttributes,
  type LoadingLayer,
} from "@/components/shared/loading/loading-owner"
import { useCreateScanWorkflow, useScanWorkflowList, useUpdateScanWorkflow } from "@/hooks/use-scan-workflows"
import { useEngineCatalog } from "@/hooks/use-engine-catalog"
import { Button } from "@/components/ui/button"
import { buildWorkflowEngineLibrary } from "@/lib/engine-catalog"
import type { Locale } from "@/i18n/config"
import type { ScanWorkflow, ScanWorkflowStageView } from "@/types/scan-workflow.types"

import { createWorkflowManagementColumns, type WorkflowManagementColumnRow } from "./scan-workflow-management-columns"
import { WorkflowCompositionCanvas } from "./workflow-composition-canvas"

type WorkflowView = "management" | "builder"
type WorkflowCatalogItem = WorkflowManagementColumnRow & {
  key: string
  item: ScanWorkflow
}
type WorkflowLoadingRow = WorkflowManagementColumnRow & { id: string }

function ScanWorkflowToolbar({
  value,
  onChange,
  placeholder,
}: {
  value: string
  onChange: (value: string) => void
  placeholder: string
}) {
  return (
    <div {...getLoadingStructureSlotAttributes("scan-workflow-toolbar")} role="search">
      <SimpleSearchToolbar
        value={value}
        onChange={onChange}
        placeholder={placeholder}
        toolbarDensity="compact"
        className="w-full sm:w-auto"
      />
    </div>
  )
}

function ScanWorkflowPrimaryRegion({ children }: { children: React.ReactNode }) {
  return (
    <div
      {...getLoadingStructureSlotAttributes("scan-workflow-primary-region")}
      className="min-h-0 px-4 lg:px-6"
    >
      {children}
    </div>
  )
}

export function ScanWorkflowPageLoadingState({
  owner,
  layer = "route",
}: {
  owner?: string
  layer?: LoadingLayer
} = {}) {
  const tWorkflow = useTranslations("scan.workflow")
  const tActions = useTranslations("common.actions")
  const columns = React.useMemo(
    () => createWorkflowManagementColumns<WorkflowLoadingRow>({
      tWorkflow,
      editLabel: tActions("edit"),
      viewLabel: tActions("view"),
      builtinLabel: tWorkflow("preset"),
      unavailableLabel: tWorkflow("needsUpdate"),
      actionMenuLabel: tWorkflow("management.actions"),
    }),
    [tActions, tWorkflow]
  )
  const toolbarLeft = React.useMemo(() => (
    <ScanWorkflowToolbar
      value=""
      onChange={() => {}}
      placeholder={tWorkflow("management.searchPlaceholder")}
    />
  ), [tWorkflow])

  if (owner !== undefined && !owner.trim()) {
    throw new Error("ScanWorkflowPageLoadingState requires a non-empty owner.")
  }

  return (
    <div
      {...(owner ? getLoadingOwnerAttributes({ owner, layer, intent: "route" }) : {})}
      data-slot="scan-workflow-page-loading-state"
      className="flex h-full min-h-0 overflow-hidden"
    >
      <ScanConfigurationWorkspace activeTab="workflows">
        <ScanWorkflowPrimaryRegion>
          <BusinessListDataTable
              data={[]}
              columns={columns}
              getRowId={(row) => row.id}
              state={{
                pagination: { pageIndex: 0, pageSize: 20 },
                cursorPaginationSummary: { total: 0 },
                paginationNavigation: {
                  mode: "cursor",
                  canFirstPage: false,
                  canPreviousPage: false,
                  canNextPage: false,
                },
                onPaginationChange: () => {},
              }}
              actions={{ showAddButton: false, showBulkAdd: false, showBulkDelete: false }}
              behavior={{ expandColumnIds: ["title", "description"] }}
              ui={{
                toolbarLeft,
                toolbarRight: <ActionSkeleton size="sm" widthClassName="w-28" />,
                showColumnVisibility: false,
                emptyMessage: tWorkflow("noMatchingWorkflow"),
                pageSizeOptions: [10, 20, 50],
                loading: true,
                loadingRowCount: 1,
                loadingRowHeightEstimate: 50.5,
              }}
          />
        </ScanWorkflowPrimaryRegion>
      </ScanConfigurationWorkspace>
    </div>
  )
}

function buildCatalogItems(workflows: ScanWorkflow[]): WorkflowCatalogItem[] {
  return workflows.map((workflow) => {
    const stages = workflow.stages ?? []
    const engineCount = stages.length > 0
      ? stages.reduce((count, stage) => count + stage.steps.length, 0)
      : workflow.steps?.length ?? 0

    return {
      key: workflow.name,
      item: workflow,
      scanWorkflowId: workflow.name.replace(/^scanWorkflows\//, ""),
      name: workflow.name,
      title: workflow.displayName || workflow.name,
      description: workflow.description,
      engineCount,
      isBuiltin: workflow.isBuiltin === true,
      isExecutable: workflow.isExecutable === true,
    }
  })
}

export default function ScanWorkflowPage() {
  const tWorkflow = useTranslations("scan.workflow")
  const tActions = useTranslations("common.actions")
  const locale = useLocale() as Locale
  const [searchQuery, setSearchQuery] = React.useState("")
  const deferredSearchQuery = React.useDeferredValue(searchQuery)
  const [pagination, setPagination] = React.useState({ pageIndex: 0, pageSize: 20 })
  const [pageTokens, setPageTokens] = React.useState<Record<number, string | undefined>>({ 0: undefined })
  const [view, setView] = React.useState<WorkflowView>("management")
  const [editingWorkflow, setEditingWorkflow] = React.useState<WorkflowCatalogItem | null>(null)
  const [draftDisplayName, setDraftDisplayName] = React.useState("")
  const [draftDescription, setDraftDescription] = React.useState("")
  const createWorkflow = useCreateScanWorkflow()
  const updateWorkflow = useUpdateScanWorkflow()
  const engineCatalog = useEngineCatalog(view === "builder")
  const workflowFilter = deferredSearchQuery.trim() || undefined
  const cursorScopeKey = `${workflowFilter ?? ""}\u0000${pagination.pageSize}`
  const hasCursorScopeChanged = useCursorPaginationScopeChange(cursorScopeKey)
  const activePageIndex = hasCursorScopeChanged ? 0 : pagination.pageIndex
  const activePageToken = hasCursorScopeChanged ? undefined : pageTokens[pagination.pageIndex]
  const {
    data: workflowPage,
    isLoading,
    isPlaceholderData,
  } = useScanWorkflowList({
    pageSize: pagination.pageSize,
    pageToken: activePageToken,
    filter: workflowFilter,
  })
  const workflows = React.useMemo(
    () => workflowPage?.scanWorkflows ?? [],
    [workflowPage?.scanWorkflows]
  )
  const workflowEngineLibrary = React.useMemo(
    () => engineCatalog.data ? buildWorkflowEngineLibrary(engineCatalog.data, locale) : [],
    [engineCatalog.data, locale]
  )
  const workflowEngineCatalogStatus = engineCatalog.isError
    ? "error"
    : engineCatalog.data
      ? "ready"
      : "loading"

  React.useEffect(() => {
    if (!hasCursorScopeChanged) return

    setPagination((current) => ({ ...current, pageIndex: 0 }))
    setPageTokens({ 0: undefined })
  }, [hasCursorScopeChanged])

  const activeNextPageToken = getCurrentCursorNextPageToken(
    workflowPage?.nextPageToken,
    isPlaceholderData || hasCursorScopeChanged,
  )

  React.useEffect(() => {
    if (!activeNextPageToken) return
    setPageTokens((tokens) => ({ ...tokens, [activePageIndex + 1]: activeNextPageToken }))
  }, [activeNextPageToken, activePageIndex])

  const paginationNavigation = React.useMemo(() => getCursorPaginationNavigation({
    currentPage: activePageIndex,
    pageTokens: hasCursorScopeChanged ? { 0: undefined } : pageTokens,
    nextPageToken: activeNextPageToken,
    firstPage: 0,
  }), [activeNextPageToken, activePageIndex, hasCursorScopeChanged, pageTokens])

  const handlePaginationChange = React.useCallback((next: { pageIndex: number; pageSize: number }) => {
    if (next.pageSize !== pagination.pageSize) {
      setPagination({ pageIndex: 0, pageSize: next.pageSize })
      setPageTokens({ 0: undefined })
      return
    }
    if (next.pageIndex === 0) {
      setPagination((current) => ({ ...current, pageIndex: 0 }))
      setPageTokens({ 0: undefined })
      return
    }
    const transition = getCursorPageTransition({
      currentPage: activePageIndex,
      pageTokens: hasCursorScopeChanged ? { 0: undefined } : pageTokens,
      nextPageToken: activeNextPageToken,
      requestedPage: next.pageIndex,
      firstPage: 0,
    })
    if (!transition.reachable) return
    setPageTokens((tokens) => ({ ...tokens, [next.pageIndex]: transition.pageToken }))
    setPagination(next)
  }, [activeNextPageToken, activePageIndex, hasCursorScopeChanged, pageTokens, pagination])

  const catalogItems = React.useMemo(
    () => buildCatalogItems(workflows),
    [workflows]
  )
  const openBuilder = React.useCallback((workflow: WorkflowCatalogItem) => {
    setEditingWorkflow(workflow)
    setDraftDisplayName(workflow.item.displayName)
    setDraftDescription(workflow.item.description)
    setView("builder")
  }, [])

  const openCreateBuilder = React.useCallback(() => {
    setEditingWorkflow(null)
    setDraftDisplayName("")
    setDraftDescription("")
    setView("builder")
  }, [])

  const saveWorkflow = React.useCallback(async (stages: ScanWorkflowStageView[]) => {
    if (editingWorkflow?.item) {
      await updateWorkflow.mutateAsync({
        id: editingWorkflow.item.name,
        input: {
          scanWorkflow: {
            name: editingWorkflow.item.name,
            displayName: draftDisplayName,
            description: draftDescription,
            stages,
            etag: editingWorkflow.item.etag,
          },
          updateMask: ["displayName", "description", "stages"],
        },
      })
    } else {
      await createWorkflow.mutateAsync({
        scanWorkflow: { displayName: draftDisplayName, description: draftDescription, stages },
        requestId: crypto.randomUUID(),
      })
    }
    setView("management")
  }, [createWorkflow, draftDescription, draftDisplayName, editingWorkflow, updateWorkflow])

  const columns = React.useMemo(
    () => createWorkflowManagementColumns<WorkflowCatalogItem>({
      tWorkflow,
      editLabel: tActions("edit"),
      viewLabel: tActions("view"),
      builtinLabel: tWorkflow("preset"),
      unavailableLabel: tWorkflow("needsUpdate"),
      actionMenuLabel: tWorkflow("management.actions"),
      onEdit: openBuilder,
    }),
    [openBuilder, tActions, tWorkflow]
  )
  const toolbarLeft = React.useMemo(() => (
    <ScanWorkflowToolbar
      value={searchQuery}
      onChange={setSearchQuery}
      placeholder={tWorkflow("management.searchPlaceholder")}
    />
  ), [searchQuery, tWorkflow])
  const toolbarRight = React.useMemo(() => (
    <Button type="button" size="sm" onClick={openCreateBuilder}>
      <semanticIcons.action.add className="size-4" />
      {tWorkflow("createWorkflow")}
    </Button>
  ), [openCreateBuilder, tWorkflow])

  return (
    <ContentHandoff
      owner="scan-workflow-page-content"
      layer="workspace"
      isLoading={isLoading}
      skeleton={<ScanWorkflowPageLoadingState />}
      className="h-full"
      skeletonClassName="h-full"
      contentClassName="h-full"
    >
      {view === "builder" ? (
        <WorkflowCompositionCanvas
          key={editingWorkflow?.key ?? "workflow-builder"}
          engines={workflowEngineLibrary}
          engineCatalogStatus={workflowEngineCatalogStatus}
          className="h-full min-h-0"
          backLabel={tWorkflow("management.backToList")}
          onBack={() => setView("management")}
          stages={editingWorkflow?.item.stages}
          readOnly={editingWorkflow?.item.isBuiltin === true}
          displayName={draftDisplayName}
          description={draftDescription}
          onDisplayNameChange={setDraftDisplayName}
          onDescriptionChange={setDraftDescription}
          onSave={saveWorkflow}
          isSaving={createWorkflow.isPending || updateWorkflow.isPending}
        />
      ) : (
        <ScanConfigurationWorkspace activeTab="workflows">
          <ScanWorkflowPrimaryRegion>
              <BusinessListDataTable
                data={catalogItems}
                columns={columns}
                getRowId={(workflow) => workflow.key}
                actions={{
                  showAddButton: false,
                  showBulkAdd: false,
                  showBulkDelete: false,
                }}
                behavior={{ expandColumnIds: ["title", "description"] }}
                state={{
                  pagination: {
                    ...pagination,
                    pageIndex: activePageIndex,
                  },
                  cursorPaginationSummary: { total: workflowPage?.totalSize ?? 0 },
                  paginationNavigation,
                  onPaginationChange: handlePaginationChange,
                }}
                ui={{
                  toolbarLeft,
                  toolbarRight,
                  showColumnVisibility: false,
                  emptyMessage: tWorkflow("noMatchingWorkflow"),
                  pageSizeOptions: [10, 20, 50],
                }}
              />
          </ScanWorkflowPrimaryRegion>
        </ScanConfigurationWorkspace>
      )}
    </ContentHandoff>
  )
}
