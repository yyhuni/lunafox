"use client"

import React, { useMemo, useState } from "react"
import dynamic from "next/dynamic"
import { Lock, Search, Settings, ChevronDown, ChevronRight } from "@/components/icons"
import { useTranslations } from "next-intl"
import { useWorkflowProfiles, useWorkflows } from "@/hooks/use-workflows"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Separator } from "@/components/ui/separator"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"
import { cn } from "@/lib/utils"
import type { WorkflowProfile, ScanWorkflow } from "@/types/workflow.types"
import { MasterDetailSkeleton } from "@/components/ui/master-detail-skeleton"

const YamlViewer = dynamic(
  () => import("@/components/ui/yaml-viewer").then((mod) => mod.YamlViewer),
  {
    ssr: false,
    loading: () => <Skeleton className="min-h-[280px] w-full" />,
  }
)

type WorkflowSelection =
  | { type: "preset"; item: WorkflowProfile }
  | { type: "workflow"; item: ScanWorkflow }
  | null

export default function ScanWorkflowPage() {
  const [selection, setSelection] = useState<WorkflowSelection>(null)
  const [searchQuery, setSearchQuery] = useState("")
  const [presetsOpen, setPresetsOpen] = useState(true)
  const [workflowsOpen, setWorkflowsOpen] = useState(true)

  const tNav = useTranslations("navigation")
  const tWorkflow = useTranslations("scan.workflow")

  const { data: presetWorkflows = [], isLoading: isLoadingPresets } = useWorkflowProfiles()
  const { data: workflows = [], isLoading: isLoadingWorkflows } = useWorkflows()

  const isLoading = isLoadingPresets || isLoadingWorkflows

  const filteredWorkflowProfiles = useMemo(() => {
    if (!searchQuery.trim()) return presetWorkflows
    const query = searchQuery.toLowerCase()
    return presetWorkflows.filter((item) => item.name.toLowerCase().includes(query))
  }, [presetWorkflows, searchQuery])

  const filteredWorkflows = useMemo(() => {
    if (!searchQuery.trim()) return workflows
    const query = searchQuery.toLowerCase()
    return workflows.filter((item) => {
      return item.name.toLowerCase().includes(query) ||
        (item.title || "").toLowerCase().includes(query) ||
        (item.description || "").toLowerCase().includes(query)
    })
  }, [workflows, searchQuery])

  if (isLoading) {
    return <MasterDetailSkeleton title={tNav("scanWorkflow")} listItemCount={4} />
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between gap-4 px-4 py-4 lg:px-6 shrink-0">
        <div className="flex items-center gap-3 shrink-0">
          <span className="px-1.5 py-0.5 text-[10px] font-mono bg-primary text-primary-foreground tracking-wider rounded-sm">
            WFL-01
          </span>
          <h1 className="text-2xl font-bold tracking-tight">{tWorkflow("title")}</h1>
        </div>

        <div className="relative w-full max-w-xs">
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
      </div>

      <Separator />

      <div className="flex flex-1 min-h-0">
        <div className="w-72 lg:w-80 border-r flex flex-col">
          <ScrollArea className="flex-1">
            <Collapsible open={presetsOpen} onOpenChange={setPresetsOpen} className="p-2">
              <CollapsibleTrigger className="flex items-center justify-between w-full px-2 py-2 hover:bg-muted rounded-lg transition-colors">
                <h2 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider flex items-center gap-1">
                  {presetsOpen ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
                  {tWorkflow("presetWorkflows")}
                </h2>
                <span className="text-xs text-muted-foreground">{filteredWorkflowProfiles.length}</span>
              </CollapsibleTrigger>
              <CollapsibleContent className="mt-1">
                {filteredWorkflowProfiles.length === 0 ? (
                  <div className="px-3 py-4 text-sm text-muted-foreground text-center">
                    {tWorkflow("noMatchingWorkflow")}
                  </div>
                ) : (
                  filteredWorkflowProfiles.map((preset) => (
                    <button
                      type="button"
                      key={preset.id}
                      onClick={() => setSelection({ type: "preset", item: preset })}
                      className={cn(
                        "w-full text-left rounded-lg px-3 py-2.5 transition-colors",
                        selection?.type === "preset" && selection.item.id === preset.id
                          ? "bg-primary/10 text-primary"
                          : "hover:bg-muted"
                      )}
                    >
                      <div className="flex items-center gap-2">
                        <Lock className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                        <span className="font-medium text-sm truncate">{preset.name}</span>
                      </div>
                      {preset.description && (
                        <div className="text-xs text-muted-foreground mt-0.5 ml-5.5 line-clamp-2">
                          {preset.description}
                        </div>
                      )}
                    </button>
                  ))
                )}
              </CollapsibleContent>
            </Collapsible>

            <Separator className="my-2" />

            <Collapsible open={workflowsOpen} onOpenChange={setWorkflowsOpen} className="p-2">
              <CollapsibleTrigger className="flex items-center justify-between w-full px-2 py-2 hover:bg-muted rounded-lg transition-colors">
                <h2 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider flex items-center gap-1">
                  {workflowsOpen ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
                  {tWorkflow("myWorkflows")}
                </h2>
                <span className="text-xs text-muted-foreground">{filteredWorkflows.length}</span>
              </CollapsibleTrigger>
              <CollapsibleContent className="mt-1">
                {filteredWorkflows.length === 0 ? (
                  <div className="px-3 py-4 text-sm text-muted-foreground text-center">
                    {searchQuery ? tWorkflow("noMatchingWorkflow") : tWorkflow("noWorkflows")}
                  </div>
                ) : (
                  filteredWorkflows.map((workflow) => (
                    <button
                      type="button"
                      key={workflow.name}
                      onClick={() => setSelection({ type: "workflow", item: workflow })}
                      className={cn(
                        "w-full text-left rounded-lg px-3 py-2.5 transition-colors",
                        selection?.type === "workflow" && selection.item.name === workflow.name
                          ? "bg-primary/10 text-primary"
                          : "hover:bg-muted"
                      )}
                    >
                      <div className="flex items-center gap-2">
                        <Settings className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                        <span className="font-medium text-sm truncate">{workflow.title || workflow.name}</span>
                      </div>
                      <div className="text-xs text-muted-foreground mt-0.5 ml-5.5 truncate">
                        {workflow.name}
                      </div>
                    </button>
                  ))
                )}
              </CollapsibleContent>
            </Collapsible>
          </ScrollArea>
        </div>

        <div className="flex-1 flex flex-col min-w-0">
          {selection ? (
            <>
              <div className="px-6 py-4 border-b">
                <div className="flex items-start gap-3">
                  <div className={cn(
                    "flex h-10 w-10 items-center justify-center rounded-lg shrink-0",
                    selection.type === "preset" ? "bg-muted" : "bg-primary/10"
                  )}>
                    {selection.type === "preset" ? (
                      <Lock className="h-5 w-5 text-muted-foreground" />
                    ) : (
                      <Settings className="h-5 w-5 text-primary" />
                    )}
                  </div>
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <h2 className="text-lg font-semibold truncate">
                        {selection.type === "preset" ? selection.item.name : (selection.item.title || selection.item.name)}
                      </h2>
                      <Badge variant="secondary" className="text-xs">
                        {selection.type === "preset" ? tWorkflow("preset") : "Workflow"}
                      </Badge>
                    </div>
                    {selection.type === "preset" && selection.item.description && (
                      <p className="text-sm text-muted-foreground mt-0.5">{selection.item.description}</p>
                    )}
                    {selection.type === "workflow" && (
                      <p className="text-sm text-muted-foreground mt-0.5">{selection.item.name}</p>
                    )}
                  </div>
                  {selection.type === "workflow" && selection.item.version && (
                    <Badge variant="outline">{selection.item.version}</Badge>
                  )}
                </div>
              </div>

              <div className="flex-1 flex flex-col min-h-0 p-6 gap-6">
                {selection.type === "workflow" ? (
                  <div className="rounded-lg border bg-muted/20 p-4 space-y-3 text-sm">
                    <div>
                      <p className="text-xs text-muted-foreground">Name</p>
                      <p className="font-medium">{selection.item.name}</p>
                    </div>
                    {selection.item.title && (
                      <div>
                        <p className="text-xs text-muted-foreground">Title</p>
                        <p className="font-medium">{selection.item.title}</p>
                      </div>
                    )}
                    {selection.item.description && (
                      <div>
                        <p className="text-xs text-muted-foreground">Description</p>
                        <p>{selection.item.description}</p>
                      </div>
                    )}
                    {selection.item.version && (
                      <div>
                        <p className="text-xs text-muted-foreground">Version</p>
                        <p className="font-medium">{selection.item.version}</p>
                      </div>
                    )}
                  </div>
                ) : (
                  <div className="flex-1 flex flex-col min-h-0">
                    <h3 className="text-sm font-medium mb-3 shrink-0">{tWorkflow("configPreview")}</h3>
                    <YamlViewer
                      value={selection.item.configuration}
                      className="flex-1 min-h-0"
                      showLineNumbers
                    />
                  </div>
                )}
              </div>
            </>
          ) : (
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
  )
}
