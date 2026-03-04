"use client"

import React, { useState, useMemo } from "react"
import dynamic from "next/dynamic"
import { Settings, Search, Pencil, Trash2, Check, Plus, Lock, AlertTriangle, ChevronDown, ChevronRight } from "@/components/icons"
import { useLocale, useTranslations } from "next-intl"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Separator } from "@/components/ui/separator"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { useWorkflows, usePresetWorkflows, useCreateWorkflow, useUpdateWorkflow, useDeleteWorkflow } from "@/hooks/use-workflows"
import { cn } from "@/lib/utils"
import type { ScanWorkflow, PresetWorkflow } from "@/types/workflow.types"
import { MasterDetailSkeleton } from "@/components/ui/master-detail-skeleton"

const WorkflowEditDialog = dynamic(
  () => import("@/components/scan/workflow/workflow-edit-dialog").then((mod) => mod.WorkflowEditDialog),
  { ssr: false }
)

const WorkflowCreateDialog = dynamic(
  () => import("@/components/scan/workflow/workflow-create-dialog").then((mod) => mod.WorkflowCreateDialog),
  { ssr: false }
)

const YamlViewer = dynamic(
  () => import("@/components/ui/yaml-viewer").then((mod) => mod.YamlViewer),
  {
    ssr: false,
    loading: () => <Skeleton className="min-h-[280px] w-full" />,
  }
)

/** Feature configuration item definition - corresponds to YAML configuration structure */
const FEATURE_LIST = [
  { key: "subdomain_discovery" },
  { key: "port_scan" },
  { key: "site_scan" },
  { key: "fingerprint_detect" },
  { key: "directory_scan" },
  { key: "screenshot" },
  { key: "url_fetch" },
  { key: "vuln_scan" },
] as const

type FeatureKey = typeof FEATURE_LIST[number]["key"]

const FEATURE_PATTERNS: Record<FeatureKey, RegExp> = {
  subdomain_discovery: /(?:^|\n)subdomain_discovery\s*:/m,
  port_scan: /(?:^|\n)port_scan\s*:/m,
  site_scan: /(?:^|\n)site_scan\s*:/m,
  fingerprint_detect: /(?:^|\n)fingerprint_detect\s*:/m,
  directory_scan: /(?:^|\n)directory_scan\s*:/m,
  screenshot: /(?:^|\n)screenshot\s*:/m,
  url_fetch: /(?:^|\n)url_fetch\s*:/m,
  vuln_scan: /(?:^|\n)vuln_scan\s*:/m,
}

/** Parse workflow configuration to get enabled features */
function parseWorkflowFeatures(configuration?: string): Record<FeatureKey, boolean> {
  const defaultFeatures: Record<FeatureKey, boolean> = {
    subdomain_discovery: false,
    port_scan: false,
    site_scan: false,
    fingerprint_detect: false,
    directory_scan: false,
    screenshot: false,
    url_fetch: false,
    vuln_scan: false,
  }

  if (!configuration) return defaultFeatures

  const normalizedConfiguration = configuration.replace(/\r\n?/g, "\n")
  for (const { key } of FEATURE_LIST) {
    defaultFeatures[key] = FEATURE_PATTERNS[key].test(normalizedConfiguration)
  }

  return defaultFeatures
}

/** Calculate the number of enabled features */
function countEnabledFeatures(configuration?: string) {
  const features = parseWorkflowFeatures(configuration)
  return Object.values(features).filter(Boolean).length
}

/** Selection type for workflow list */
type WorkflowSelection = 
  | { type: 'preset'; workflow: PresetWorkflow }
  | { type: 'user'; workflow: ScanWorkflow }
  | null

/**
 * Scan workflow template page
 */
export default function ScanWorkflowPage() {
  const locale = useLocale()
  const [selection, setSelection] = useState<WorkflowSelection>(null)
  const [searchQuery, setSearchQuery] = useState("")
  const [editingWorkflow, setEditingWorkflow] = useState<ScanWorkflow | null>(null)
  const [isEditDialogOpen, setIsEditDialogOpen] = useState(false)
  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false)
  const [createFromPreset, setCreateFromPreset] = useState<PresetWorkflow | null>(null)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [workflowToDelete, setWorkflowToDelete] = useState<ScanWorkflow | null>(null)
  const [presetsOpen, setPresetsOpen] = useState(true)
  const [myWorkflowsOpen, setMyWorkflowsOpen] = useState(true)
  
  // Internationalization
  const tCommon = useTranslations("common")
  const tConfirm = useTranslations("common.confirm")
  const tNav = useTranslations("navigation")
  const tWorkflow = useTranslations("scan.workflow")

  // API Hooks
  const { data: presetWorkflows = [], isLoading: isLoadingPresets } = usePresetWorkflows()
  const { data: userWorkflows = [], isLoading: isLoadingWorkflows } = useWorkflows()
  const createWorkflowMutation = useCreateWorkflow()
  const updateWorkflowMutation = useUpdateWorkflow()
  const deleteWorkflowMutation = useDeleteWorkflow()

  const isLoading = isLoadingPresets || isLoadingWorkflows

  // Filter workflow lists based on search query
  const filteredPresetWorkflows = useMemo(() => {
    if (!searchQuery.trim()) return presetWorkflows
    const query = searchQuery.toLowerCase()
    return presetWorkflows.filter((e) => e.name.toLowerCase().includes(query))
  }, [presetWorkflows, searchQuery])

  const filteredUserWorkflows = useMemo(() => {
    if (!searchQuery.trim()) return userWorkflows
    const query = searchQuery.toLowerCase()
    return userWorkflows.filter((e) => e.name.toLowerCase().includes(query))
  }, [userWorkflows, searchQuery])

  // Get selected features
  const selectedFeatures = useMemo(() => {
    if (!selection) return null
    const config = selection.type === 'preset' 
      ? selection.workflow.configuration 
      : selection.workflow.configuration
    return parseWorkflowFeatures(config)
  }, [selection])

  const handleSelectPreset = (preset: PresetWorkflow) => {
    setSelection({ type: 'preset', workflow: preset })
  }

  const handleSelectUserWorkflow = (workflow: ScanWorkflow) => {
    setSelection({ type: 'user', workflow })
  }

  const handleEdit = (workflow: ScanWorkflow) => {
    setEditingWorkflow(workflow)
    setIsEditDialogOpen(true)
  }

  const handleSaveYaml = async (workflowID: number, yamlContent: string) => {
    await updateWorkflowMutation.mutateAsync({
      id: workflowID,
      data: { configuration: yamlContent },
    })
  }

  const handleDelete = (workflow: ScanWorkflow) => {
    setWorkflowToDelete(workflow)
    setDeleteDialogOpen(true)
  }

  const confirmDelete = () => {
    if (!workflowToDelete) return
    deleteWorkflowMutation.mutate(workflowToDelete.id, {
      onSuccess: () => {
        if (selection?.type === 'user' && selection.workflow.id === workflowToDelete.id) {
          setSelection(null)
        }
        setDeleteDialogOpen(false)
        setWorkflowToDelete(null)
      },
    })
  }

  const handleCreateWorkflow = async (name: string, yamlContent: string) => {
    await createWorkflowMutation.mutateAsync({
      name,
      configuration: yamlContent,
    })
    setCreateFromPreset(null)
  }

  const handleOpenCreateDialog: React.MouseEventHandler<HTMLButtonElement> = () => {
    setCreateFromPreset(null)
    setIsCreateDialogOpen(true)
  }

  // Loading state
  if (isLoading) {
    return <MasterDetailSkeleton title={tNav("scanEngine")} listItemCount={4} />
  }

  return (
    <div className="flex flex-col h-full">
      {/* Top: Header + Toolbar combined */}
      <div className="flex items-center justify-between gap-4 px-4 py-4 lg:px-6 shrink-0">
        <div className="flex items-center gap-3 shrink-0">
          <span className="px-1.5 py-0.5 text-[10px] font-mono bg-primary text-primary-foreground tracking-wider rounded-sm">
            WFL-01
          </span>
          <h1 className="text-2xl font-bold tracking-tight">{tWorkflow("title")}</h1>
        </div>

        <div className="flex items-center gap-3 flex-1 justify-end max-w-xl">
          <div className="relative flex-1 max-w-xs">
            <Search className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
            <Input
              type="search"
              name="workflowSearch"
              autoComplete="off"
              placeholder={tWorkflow("searchPlaceholder")}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-8 h-9 bg-muted/50"
            />
          </div>
          <Button onClick={handleOpenCreateDialog} size="sm" className="h-9">
            <Plus className="h-3.5 w-3.5 mr-1.5" />
            {tWorkflow("createEngine")}
          </Button>
        </div>
      </div>

      <Separator />

      {/* Main: Left list + Right details */}
      <div className="flex flex-1 min-h-0">
        <div className="flex flex-1 min-h-0">
          {/* Left: Workflow list */}
          <div className="w-72 lg:w-80 border-r flex flex-col">
            <ScrollArea className="flex-1">
              {/* Preset workflows section */}
              <Collapsible open={presetsOpen} onOpenChange={setPresetsOpen} className="p-2">
                <CollapsibleTrigger className="flex items-center justify-between w-full px-2 py-2 hover:bg-muted rounded-lg transition-colors">
                  <h2 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider flex items-center gap-1">
                    {presetsOpen ? (
                      <ChevronDown className="h-3.5 w-3.5" />
                    ) : (
                      <ChevronRight className="h-3.5 w-3.5" />
                    )}
                    {tWorkflow("presetWorkflows")}
                  </h2>
                  <span className="text-xs text-muted-foreground">{filteredPresetWorkflows.length}</span>
                </CollapsibleTrigger>
                <CollapsibleContent className="mt-1">
                  {filteredPresetWorkflows.length === 0 ? (
                    <div className="px-3 py-4 text-sm text-muted-foreground text-center">
                      {tWorkflow("noMatchingWorkflow")}
                    </div>
                  ) : (
                    filteredPresetWorkflows.map((preset) => (
                      <button type="button"
                        key={preset.id}
                        onClick={() => handleSelectPreset(preset)}
                        className={cn(
                          "w-full text-left rounded-lg px-3 py-2.5 transition-colors",
                          selection?.type === 'preset' && selection.workflow.id === preset.id
                            ? "bg-primary/10 text-primary"
                            : "hover:bg-muted"
                        )}
                      >
                        <div className="flex items-center gap-2">
                          <Lock className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                          <span className="font-medium text-sm truncate">{preset.name}</span>
                        </div>
                        <div className="text-xs text-muted-foreground mt-0.5 ml-5.5">
                          {tWorkflow("featuresEnabled", { count: countEnabledFeatures(preset.configuration) })}
                        </div>
                      </button>
                    ))
                  )}
                </CollapsibleContent>
              </Collapsible>

              <Separator className="my-2" />

              {/* User workflows section */}
              <Collapsible open={myWorkflowsOpen} onOpenChange={setMyWorkflowsOpen} className="p-2">
                <CollapsibleTrigger className="flex items-center justify-between w-full px-2 py-2 hover:bg-muted rounded-lg transition-colors">
                  <h2 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider flex items-center gap-1">
                    {myWorkflowsOpen ? (
                      <ChevronDown className="h-3.5 w-3.5" />
                    ) : (
                      <ChevronRight className="h-3.5 w-3.5" />
                    )}
                    {tWorkflow("myWorkflows")}
                  </h2>
                  <span className="text-xs text-muted-foreground">{filteredUserWorkflows.length}</span>
                </CollapsibleTrigger>
                <CollapsibleContent className="mt-1">
                  {filteredUserWorkflows.length === 0 ? (
                    <div className="px-3 py-4 text-sm text-muted-foreground text-center">
                      {searchQuery ? tWorkflow("noMatchingWorkflow") : tWorkflow("noWorkflows")}
                    </div>
                  ) : (
                    filteredUserWorkflows.map((workflow) => (
                      <button type="button"
                        key={workflow.id}
                        onClick={() => handleSelectUserWorkflow(workflow)}
                        className={cn(
                          "w-full text-left rounded-lg px-3 py-2.5 transition-colors",
                          selection?.type === 'user' && selection.workflow.id === workflow.id
                            ? "bg-primary/10 text-primary"
                            : "hover:bg-muted"
                        )}
                      >
                        <div className="flex items-center gap-2">
                          {workflow.isValid === false ? (
                            <AlertTriangle className="h-3.5 w-3.5 text-amber-500 shrink-0" />
                          ) : (
                            <Check className="h-3.5 w-3.5 text-green-500 shrink-0" />
                          )}
                          <span className="font-medium text-sm truncate">{workflow.name}</span>
                        </div>
                        <div className="text-xs text-muted-foreground mt-0.5 ml-5.5">
                          {workflow.isValid === false ? (
                            <span className="text-amber-500">{tWorkflow("configNeedsUpdate")}</span>
                          ) : (
                            tWorkflow("featuresEnabled", { count: countEnabledFeatures(workflow.configuration) })
                          )}
                        </div>
                      </button>
                    ))
                  )}
                </CollapsibleContent>
              </Collapsible>
            </ScrollArea>
          </div>

          {/* Right: Workflow details */}
          <div className="flex-1 flex flex-col min-w-0">
            {selection && selectedFeatures ? (
              <>
                {/* Details header */}
                <div className="px-6 py-4 border-b">
                  <div className="flex items-start gap-3">
                    <div className={cn(
                      "flex h-10 w-10 items-center justify-center rounded-lg shrink-0",
                      selection.type === 'preset' ? "bg-muted" : "bg-primary/10"
                    )}>
                      {selection.type === 'preset' ? (
                        <Lock className="h-5 w-5 text-muted-foreground" />
                      ) : (
                        <Settings className="h-5 w-5 text-primary" />
                      )}
                    </div>
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <h2 className="text-lg font-semibold truncate">
                          {selection.workflow.name}
                        </h2>
                        {selection.type === 'preset' && (
                          <Badge variant="secondary" className="text-xs">
                            {tWorkflow("preset")}
                          </Badge>
                        )}
                        {selection.type === 'user' && selection.workflow.isValid === false && (
                          <Badge variant="outline" className="text-amber-500 border-amber-500 text-xs">
                            {tWorkflow("needsUpdate")}
                          </Badge>
                        )}
                      </div>
                      {selection.type === 'preset' && selection.workflow.description && (
                        <p className="text-sm text-muted-foreground mt-0.5">
                          {selection.workflow.description}
                        </p>
                      )}
                      {selection.type === 'user' && (
                        <p className="text-sm text-muted-foreground mt-0.5">
                          {tWorkflow("updatedAt")} {new Date(selection.workflow.updatedAt).toLocaleString(locale)}
                        </p>
                      )}
                    </div>
                    <Badge variant="outline">
                      {tWorkflow("featuresCount", { 
                        count: countEnabledFeatures(selection.workflow.configuration) 
                      })}
                    </Badge>
                  </div>
                </div>

                {/* Details content */}
                <div className="flex-1 flex flex-col min-h-0 p-6 gap-6">
                  {/* Feature status */}
                  <div className="shrink-0">
                    <h3 className="text-sm font-medium mb-3">{tWorkflow("enabledFeatures")}</h3>
                    <div className="flex flex-wrap gap-2">
                      {FEATURE_LIST.map((feature) => {
                        const enabled = selectedFeatures[feature.key as keyof typeof selectedFeatures]
                        return (
                          <Badge
                            key={feature.key}
                            variant={enabled ? "default" : "outline"}
                            className={cn(
                              "text-xs",
                              enabled 
                                ? "bg-primary/10 text-primary hover:bg-primary/10" 
                                : "text-muted-foreground/50"
                            )}
                          >
                            {enabled && <Check className="h-3 w-3 mr-1" />}
                            {tWorkflow(`features.${feature.key}`)}
                          </Badge>
                        )
                      })}
                    </div>
                  </div>

                  {/* Configuration preview */}
                  {selection.workflow.configuration && (
                    <div className="flex-1 flex flex-col min-h-0">
                      <h3 className="text-sm font-medium mb-3 shrink-0">{tWorkflow("configPreview")}</h3>
                      <YamlViewer 
                        value={selection.workflow.configuration} 
                        className="flex-1 min-h-0"
                        showLineNumbers
                      />
                    </div>
                  )}
                </div>

                {/* Action buttons - only show for user workflows */}
                {selection.type === 'user' && (
                  <div className="px-6 py-4 border-t flex items-center gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleEdit(selection.workflow)}
                    >
                      <Pencil className="h-4 w-4 mr-1.5" />
                      {tWorkflow("editConfig")}
                    </Button>
                    <div className="flex-1" />
                    <Button
                      variant="outline"
                      size="sm"
                      className="text-destructive hover:text-destructive"
                      onClick={() => handleDelete(selection.workflow)}
                      disabled={deleteWorkflowMutation.isPending}
                    >
                      <Trash2 className="h-4 w-4 mr-1.5" />
                      {tCommon("actions.delete")}
                    </Button>
                  </div>
                )}
              </>
            ) : (
              // Unselected state
              <div className="flex-1 flex items-center justify-center">
                <div className="text-center text-muted-foreground">
                  <Settings className="h-12 w-12 mx-auto mb-3 opacity-50" />
                  <p className="text-sm">{tWorkflow("selectWorkflowHint")}</p>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Edit workflow dialog */}
      {isEditDialogOpen ? (
        <WorkflowEditDialog
          workflow={editingWorkflow}
          open={isEditDialogOpen}
          onOpenChange={setIsEditDialogOpen}
          onSave={handleSaveYaml}
        />
      ) : null}

      {/* Create workflow dialog */}
      {isCreateDialogOpen ? (
        <WorkflowCreateDialog
          open={isCreateDialogOpen}
          onOpenChange={(open) => {
            setIsCreateDialogOpen(open)
            if (!open) setCreateFromPreset(null)
          }}
          onSave={handleCreateWorkflow}
          preSelectedPreset={createFromPreset || undefined}
        />
      ) : null}

      {/* Delete confirmation dialog */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{tConfirm("deleteTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {tConfirm("deleteEngineMessage", { name: workflowToDelete?.name ?? "" })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{tCommon("actions.cancel")}</AlertDialogCancel>
            <AlertDialogAction 
              onClick={confirmDelete} 
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              disabled={deleteWorkflowMutation.isPending}
            >
              {deleteWorkflowMutation.isPending ? tConfirm("deleting") : tCommon("actions.delete")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
