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
import { useEngines, usePresetEngines, useCreateEngine, useUpdateEngine, useDeleteEngine } from "@/hooks/use-engines"
import { cn } from "@/lib/utils"
import type { ScanEngine, PresetEngine } from "@/types/engine.types"
import { MasterDetailSkeleton } from "@/components/ui/master-detail-skeleton"

const EngineEditDialog = dynamic(
  () => import("@/components/scan/engine/engine-edit-dialog").then((mod) => mod.EngineEditDialog),
  { ssr: false }
)

const EngineCreateDialog = dynamic(
  () => import("@/components/scan/engine/engine-create-dialog").then((mod) => mod.EngineCreateDialog),
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

/** Parse engine configuration to get enabled features */
function parseEngineFeatures(configuration?: string): Record<FeatureKey, boolean> {
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
  const features = parseEngineFeatures(configuration)
  return Object.values(features).filter(Boolean).length
}

/** Selection type for engine list */
type EngineSelection = 
  | { type: 'preset'; engine: PresetEngine }
  | { type: 'user'; engine: ScanEngine }
  | null

/**
 * Scan engine page
 */
export default function ScanEnginePage() {
  const locale = useLocale()
  const [selection, setSelection] = useState<EngineSelection>(null)
  const [searchQuery, setSearchQuery] = useState("")
  const [editingEngine, setEditingEngine] = useState<ScanEngine | null>(null)
  const [isEditDialogOpen, setIsEditDialogOpen] = useState(false)
  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false)
  const [createFromPreset, setCreateFromPreset] = useState<PresetEngine | null>(null)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [engineToDelete, setEngineToDelete] = useState<ScanEngine | null>(null)
  const [presetsOpen, setPresetsOpen] = useState(true)
  const [myEnginesOpen, setMyEnginesOpen] = useState(true)
  
  // Internationalization
  const tCommon = useTranslations("common")
  const tConfirm = useTranslations("common.confirm")
  const tNav = useTranslations("navigation")
  const tEngine = useTranslations("scan.engine")

  // API Hooks
  const { data: presetEngines = [], isLoading: isLoadingPresets } = usePresetEngines()
  const { data: userEngines = [], isLoading: isLoadingEngines } = useEngines()
  const createEngineMutation = useCreateEngine()
  const updateEngineMutation = useUpdateEngine()
  const deleteEngineMutation = useDeleteEngine()

  const isLoading = isLoadingPresets || isLoadingEngines

  // Filter engine lists based on search query
  const filteredPresetEngines = useMemo(() => {
    if (!searchQuery.trim()) return presetEngines
    const query = searchQuery.toLowerCase()
    return presetEngines.filter((e) => e.name.toLowerCase().includes(query))
  }, [presetEngines, searchQuery])

  const filteredUserEngines = useMemo(() => {
    if (!searchQuery.trim()) return userEngines
    const query = searchQuery.toLowerCase()
    return userEngines.filter((e) => e.name.toLowerCase().includes(query))
  }, [userEngines, searchQuery])

  // Get selected features
  const selectedFeatures = useMemo(() => {
    if (!selection) return null
    const config = selection.type === 'preset' 
      ? selection.engine.configuration 
      : selection.engine.configuration
    return parseEngineFeatures(config)
  }, [selection])

  const handleSelectPreset = (preset: PresetEngine) => {
    setSelection({ type: 'preset', engine: preset })
  }

  const handleSelectUserEngine = (engine: ScanEngine) => {
    setSelection({ type: 'user', engine })
  }

  const handleEdit = (engine: ScanEngine) => {
    setEditingEngine(engine)
    setIsEditDialogOpen(true)
  }

  const handleSaveYaml = async (engineId: number, yamlContent: string) => {
    await updateEngineMutation.mutateAsync({
      id: engineId,
      data: { configuration: yamlContent },
    })
  }

  const handleDelete = (engine: ScanEngine) => {
    setEngineToDelete(engine)
    setDeleteDialogOpen(true)
  }

  const confirmDelete = () => {
    if (!engineToDelete) return
    deleteEngineMutation.mutate(engineToDelete.id, {
      onSuccess: () => {
        if (selection?.type === 'user' && selection.engine.id === engineToDelete.id) {
          setSelection(null)
        }
        setDeleteDialogOpen(false)
        setEngineToDelete(null)
      },
    })
  }

  const handleCreateEngine = async (name: string, yamlContent: string) => {
    await createEngineMutation.mutateAsync({
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
            ENG-01
          </span>
          <h1 className="text-2xl font-bold tracking-tight">{tEngine("title")}</h1>
        </div>

        <div className="flex items-center gap-3 flex-1 justify-end max-w-xl">
          <div className="relative flex-1 max-w-xs">
            <Search className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
            <Input
              type="search"
              name="engineSearch"
              autoComplete="off"
              placeholder={tEngine("searchPlaceholder")}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-8 h-9 bg-muted/50"
            />
          </div>
          <Button onClick={handleOpenCreateDialog} size="sm" className="h-9">
            <Plus className="h-3.5 w-3.5 mr-1.5" />
            {tEngine("createEngine")}
          </Button>
        </div>
      </div>

      <Separator />

      {/* Main: Left list + Right details */}
      <div className="flex flex-1 min-h-0">
        <div className="flex flex-1 min-h-0">
          {/* Left: Engine list */}
          <div className="w-72 lg:w-80 border-r flex flex-col">
            <ScrollArea className="flex-1">
              {/* Preset engines section */}
              <Collapsible open={presetsOpen} onOpenChange={setPresetsOpen} className="p-2">
                <CollapsibleTrigger className="flex items-center justify-between w-full px-2 py-2 hover:bg-muted rounded-lg transition-colors">
                  <h2 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider flex items-center gap-1">
                    {presetsOpen ? (
                      <ChevronDown className="h-3.5 w-3.5" />
                    ) : (
                      <ChevronRight className="h-3.5 w-3.5" />
                    )}
                    {tEngine("presetEngines")}
                  </h2>
                  <span className="text-xs text-muted-foreground">{filteredPresetEngines.length}</span>
                </CollapsibleTrigger>
                <CollapsibleContent className="mt-1">
                  {filteredPresetEngines.length === 0 ? (
                    <div className="px-3 py-4 text-sm text-muted-foreground text-center">
                      {tEngine("noMatchingEngine")}
                    </div>
                  ) : (
                    filteredPresetEngines.map((preset) => (
                      <button type="button"
                        key={preset.id}
                        onClick={() => handleSelectPreset(preset)}
                        className={cn(
                          "w-full text-left rounded-lg px-3 py-2.5 transition-colors",
                          selection?.type === 'preset' && selection.engine.id === preset.id
                            ? "bg-primary/10 text-primary"
                            : "hover:bg-muted"
                        )}
                      >
                        <div className="flex items-center gap-2">
                          <Lock className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                          <span className="font-medium text-sm truncate">{preset.name}</span>
                        </div>
                        <div className="text-xs text-muted-foreground mt-0.5 ml-5.5">
                          {tEngine("featuresEnabled", { count: countEnabledFeatures(preset.configuration) })}
                        </div>
                      </button>
                    ))
                  )}
                </CollapsibleContent>
              </Collapsible>

              <Separator className="my-2" />

              {/* User engines section */}
              <Collapsible open={myEnginesOpen} onOpenChange={setMyEnginesOpen} className="p-2">
                <CollapsibleTrigger className="flex items-center justify-between w-full px-2 py-2 hover:bg-muted rounded-lg transition-colors">
                  <h2 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider flex items-center gap-1">
                    {myEnginesOpen ? (
                      <ChevronDown className="h-3.5 w-3.5" />
                    ) : (
                      <ChevronRight className="h-3.5 w-3.5" />
                    )}
                    {tEngine("myEngines")}
                  </h2>
                  <span className="text-xs text-muted-foreground">{filteredUserEngines.length}</span>
                </CollapsibleTrigger>
                <CollapsibleContent className="mt-1">
                  {filteredUserEngines.length === 0 ? (
                    <div className="px-3 py-4 text-sm text-muted-foreground text-center">
                      {searchQuery ? tEngine("noMatchingEngine") : tEngine("noEngines")}
                    </div>
                  ) : (
                    filteredUserEngines.map((engine) => (
                      <button type="button"
                        key={engine.id}
                        onClick={() => handleSelectUserEngine(engine)}
                        className={cn(
                          "w-full text-left rounded-lg px-3 py-2.5 transition-colors",
                          selection?.type === 'user' && selection.engine.id === engine.id
                            ? "bg-primary/10 text-primary"
                            : "hover:bg-muted"
                        )}
                      >
                        <div className="flex items-center gap-2">
                          {engine.isValid === false ? (
                            <AlertTriangle className="h-3.5 w-3.5 text-amber-500 shrink-0" />
                          ) : (
                            <Check className="h-3.5 w-3.5 text-green-500 shrink-0" />
                          )}
                          <span className="font-medium text-sm truncate">{engine.name}</span>
                        </div>
                        <div className="text-xs text-muted-foreground mt-0.5 ml-5.5">
                          {engine.isValid === false ? (
                            <span className="text-amber-500">{tEngine("configNeedsUpdate")}</span>
                          ) : (
                            tEngine("featuresEnabled", { count: countEnabledFeatures(engine.configuration) })
                          )}
                        </div>
                      </button>
                    ))
                  )}
                </CollapsibleContent>
              </Collapsible>
            </ScrollArea>
          </div>

          {/* Right: Engine details */}
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
                          {selection.engine.name}
                        </h2>
                        {selection.type === 'preset' && (
                          <Badge variant="secondary" className="text-xs">
                            {tEngine("preset")}
                          </Badge>
                        )}
                        {selection.type === 'user' && selection.engine.isValid === false && (
                          <Badge variant="outline" className="text-amber-500 border-amber-500 text-xs">
                            {tEngine("needsUpdate")}
                          </Badge>
                        )}
                      </div>
                      {selection.type === 'preset' && selection.engine.description && (
                        <p className="text-sm text-muted-foreground mt-0.5">
                          {selection.engine.description}
                        </p>
                      )}
                      {selection.type === 'user' && (
                        <p className="text-sm text-muted-foreground mt-0.5">
                          {tEngine("updatedAt")} {new Date(selection.engine.updatedAt).toLocaleString(locale)}
                        </p>
                      )}
                    </div>
                    <Badge variant="outline">
                      {tEngine("featuresCount", { 
                        count: countEnabledFeatures(selection.engine.configuration) 
                      })}
                    </Badge>
                  </div>
                </div>

                {/* Details content */}
                <div className="flex-1 flex flex-col min-h-0 p-6 gap-6">
                  {/* Feature status */}
                  <div className="shrink-0">
                    <h3 className="text-sm font-medium mb-3">{tEngine("enabledFeatures")}</h3>
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
                            {tEngine(`features.${feature.key}`)}
                          </Badge>
                        )
                      })}
                    </div>
                  </div>

                  {/* Configuration preview */}
                  {selection.engine.configuration && (
                    <div className="flex-1 flex flex-col min-h-0">
                      <h3 className="text-sm font-medium mb-3 shrink-0">{tEngine("configPreview")}</h3>
                      <YamlViewer 
                        value={selection.engine.configuration} 
                        className="flex-1 min-h-0"
                        showLineNumbers
                      />
                    </div>
                  )}
                </div>

                {/* Action buttons - only show for user engines */}
                {selection.type === 'user' && (
                  <div className="px-6 py-4 border-t flex items-center gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleEdit(selection.engine)}
                    >
                      <Pencil className="h-4 w-4 mr-1.5" />
                      {tEngine("editConfig")}
                    </Button>
                    <div className="flex-1" />
                    <Button
                      variant="outline"
                      size="sm"
                      className="text-destructive hover:text-destructive"
                      onClick={() => handleDelete(selection.engine)}
                      disabled={deleteEngineMutation.isPending}
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
                  <p className="text-sm">{tEngine("selectEngineHint")}</p>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Edit engine dialog */}
      {isEditDialogOpen ? (
        <EngineEditDialog
          engine={editingEngine}
          open={isEditDialogOpen}
          onOpenChange={setIsEditDialogOpen}
          onSave={handleSaveYaml}
        />
      ) : null}

      {/* Create engine dialog */}
      {isCreateDialogOpen ? (
        <EngineCreateDialog
          open={isCreateDialogOpen}
          onOpenChange={(open) => {
            setIsCreateDialogOpen(open)
            if (!open) setCreateFromPreset(null)
          }}
          onSave={handleCreateEngine}
          preSelectedPreset={createFromPreset || undefined}
        />
      ) : null}

      {/* Delete confirmation dialog */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{tConfirm("deleteTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {tConfirm("deleteEngineMessage", { name: engineToDelete?.name ?? "" })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{tCommon("actions.cancel")}</AlertDialogCancel>
            <AlertDialogAction 
              onClick={confirmDelete} 
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              disabled={deleteEngineMutation.isPending}
            >
              {deleteEngineMutation.isPending ? tConfirm("deleting") : tCommon("actions.delete")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
