"use client"

import React, { useMemo, useCallback } from "react"
import { Check, Play, Server, Settings, Zap } from "@/components/icons"
import { useTranslations } from "next-intl"

import { Badge } from "@/components/ui/badge"
import { Checkbox } from "@/components/ui/checkbox"
import { cn } from "@/lib/utils"
import {
  CAPABILITY_CONFIG,
  extractWorkflowIds,
  mergeWorkflowConfigurations,
  parseWorkflowCapabilities,
  serializeWorkflowConfiguration,
} from "@/lib/workflow-config"
import { useWorkflowProfiles } from "@/hooks/use-workflows"
import { LoadingSpinner } from "@/components/loading-spinner"

import type { ScanWorkflow } from "@/types/workflow.types"

export interface WorkflowProfileItem {
  id: string
  label: string
  description: string
  icon: React.ComponentType<{ className?: string }>
  workflowIds: number[]
}

interface WorkflowProfileSelectorProps {
  workflows: ScanWorkflow[]
  selectedWorkflowIds: number[]
  selectedPresetId: string | null
  onPresetChange: (presetId: string | null) => void
  onWorkflowIdsChange: (workflowIds: number[]) => void
  onConfigurationChange: (config: string) => void
  disabled?: boolean
  className?: string
  showHeader?: boolean
  contentClassName?: string
}

export function WorkflowProfileSelector({
  workflows,
  selectedWorkflowIds,
  selectedPresetId,
  onPresetChange,
  onWorkflowIdsChange,
  onConfigurationChange,
  disabled = false,
  className,
  showHeader = true,
  contentClassName,
}: WorkflowProfileSelectorProps) {
  const t = useTranslations("scan.initiate")
  const tStages = useTranslations("scan.progress.stages")
  const { data: backendPresets, isLoading: isLoadingPresets } = useWorkflowProfiles()

  const workflowProfiles = useMemo(() => {
    const profiles: WorkflowProfileItem[] = []

    if (backendPresets && backendPresets.length > 0) {
      backendPresets.forEach((preset) => {
        const caps = parseWorkflowCapabilities(preset.configuration)

        const matchingWorkflowIds: number[] = []
        if (workflows && workflows.length > 0) {
          const preferredNames = (preset.workflowIds || preset.workflowNames || extractWorkflowIds(preset.configuration))
            .map((item) => item.trim())
            .filter(Boolean)
          if (preferredNames.length > 0) {
            workflows.forEach((workflow) => {
              if (preferredNames.includes(workflow.name)) {
                matchingWorkflowIds.push(workflow.id)
              }
            })
          } else {
            workflows.forEach((workflow) => {
              const workflowCaps = parseWorkflowCapabilities(workflow.configuration)
              const hasAllCaps = caps.every(cap => workflowCaps.includes(cap))
              if (hasAllCaps && caps.length > 0) {
                matchingWorkflowIds.push(workflow.id)
              }
            })
          }
        }

        let Icon = Server
        if (caps.includes("vuln_scan")) {
          Icon = caps.some(c => ["subdomain_discovery", "port_scan", "site_scan"].includes(c)) ? Zap : Play
        }

        profiles.push({
          id: preset.id,
          label: preset.name,
          description: preset.description || "",
          icon: Icon,
          workflowIds: matchingWorkflowIds,
        })
      })
    }

    profiles.push({
      id: "custom",
      label: t("presets.custom"),
      description: t("presets.customDesc"),
      icon: Settings,
      workflowIds: [],
    })

    return profiles
  }, [backendPresets, workflows, t])

  const selectedWorkflows = useMemo(() => {
    if (!selectedWorkflowIds.length || !workflows) return []
    return workflows.filter((item) => selectedWorkflowIds.includes(item.id))
  }, [selectedWorkflowIds, workflows])

  const selectedCapabilities = useMemo(() => {
    if (!selectedWorkflows.length) return []
    const allCaps = new Set<string>()
    selectedWorkflows.forEach((workflow) => {
      parseWorkflowCapabilities(workflow.configuration).forEach((cap) => allCaps.add(cap))
    })
    return Array.from(allCaps)
  }, [selectedWorkflows])

  const selectedPreset = useMemo(() => {
    return workflowProfiles.find((item) => item.id === selectedPresetId)
  }, [workflowProfiles, selectedPresetId])

  const matchingWorkflows = useMemo(() => {
    if (!selectedPreset || selectedPreset.id === "custom") return []
    return workflows?.filter((item) => selectedPreset.workflowIds.includes(item.id)) || []
  }, [selectedPreset, workflows])

  const updateConfigurationFromWorkflows = useCallback((workflowIds: number[]) => {
    if (!workflows) return
    const selectedItems = workflows.filter((item) => workflowIds.includes(item.id))
    const mergedConfig = mergeWorkflowConfigurations(selectedItems.map((item) => item.configuration))
    onConfigurationChange(serializeWorkflowConfiguration(mergedConfig))
  }, [workflows, onConfigurationChange])

  const handlePresetSelect = useCallback((preset: WorkflowProfileItem) => {
    onPresetChange(preset.id)
    if (preset.id !== "custom") {
      const backendPreset = backendPresets?.find(p => p.id === preset.id)
      if (backendPreset && backendPreset.configuration) {
        onConfigurationChange(serializeWorkflowConfiguration(backendPreset.configuration))
        if (preset.workflowIds.length > 0) {
          onWorkflowIdsChange(preset.workflowIds)
        } else {
          onWorkflowIdsChange([])
        }
      } else {
        onWorkflowIdsChange(preset.workflowIds)
        updateConfigurationFromWorkflows(preset.workflowIds)
      }
    } else if (selectedWorkflowIds.length === 0) {
      onConfigurationChange("")
    }
  }, [onPresetChange, onWorkflowIdsChange, updateConfigurationFromWorkflows, selectedWorkflowIds.length, onConfigurationChange, backendPresets])

  const handleWorkflowToggle = useCallback((workflowID: number, checked: boolean) => {
    const newWorkflowIds = checked
      ? [...selectedWorkflowIds, workflowID]
      : selectedWorkflowIds.filter((id) => id !== workflowID)
    onWorkflowIdsChange(newWorkflowIds)
    updateConfigurationFromWorkflows(newWorkflowIds)
  }, [selectedWorkflowIds, onWorkflowIdsChange, updateConfigurationFromWorkflows])

  if (isLoadingPresets) {
    return (
      <div className={cn("flex flex-col h-full items-center justify-center", className)}>
        <LoadingSpinner />
        <p className="text-sm text-muted-foreground mt-4">{t("presets.loading")}</p>
      </div>
    )
  }

  return (
    <div className={cn("flex flex-col", className)}>
      <div className={cn("p-6", contentClassName)}>
        {showHeader && (
          <div className="flex items-start justify-between gap-3 mb-4">
            <div>
              <p className="text-sm font-medium">{t("presets.title")}</p>
              <p className="text-xs text-muted-foreground">{t("presets.selectHint")}</p>
            </div>
            {selectedWorkflowIds.length > 0 && (
              <Badge variant="secondary" className="text-xs">
                {t("selectedCount", { count: selectedWorkflowIds.length })}
              </Badge>
            )}
          </div>
        )}
        <div className="grid gap-3 mb-4 grid-cols-2 sm:grid-cols-3 lg:grid-cols-4">
          {workflowProfiles.map((preset) => {
            const isActive = selectedPresetId === preset.id
            const PresetIcon = preset.icon
            const matchedWorkflows = preset.id === "custom"
              ? []
              : workflows?.filter((item) => preset.workflowIds.includes(item.id)) || []

            return (
              <button
                key={preset.id}
                type="button"
                onClick={() => handlePresetSelect(preset)}
                disabled={disabled}
                aria-pressed={isActive}
                className={cn(
                  "relative flex flex-col items-center p-3 rounded-lg border-2 text-center transition-[background-color,border-color,color,box-shadow]",
                  isActive
                    ? "border-primary bg-primary/5 shadow-sm"
                    : "border-border hover:border-primary/50 hover:bg-muted/30",
                  disabled && "opacity-50 cursor-not-allowed"
                )}
              >
                {isActive && (
                  <span className="absolute right-2 top-2 inline-flex h-5 w-5 items-center justify-center rounded-full bg-primary text-primary-foreground shadow">
                    <Check className="h-3 w-3" />
                  </span>
                )}
                <div className={cn(
                  "flex h-10 w-10 items-center justify-center rounded-lg mb-2",
                  isActive ? "bg-primary text-primary-foreground" : "bg-muted"
                )}>
                  <PresetIcon className="h-5 w-5" />
                </div>
                <span className="text-sm font-medium">{preset.label}</span>
                {preset.id !== "custom" && (
                  <span className="text-xs text-muted-foreground mt-1">
                    {matchedWorkflows.length} {t("presets.workflowCount")}
                  </span>
                )}
              </button>
            )
          })}
        </div>

        {selectedPresetId && selectedPresetId !== "custom" && (
          <div className="border rounded-lg p-4 bg-muted/10">
            <div className="flex items-start justify-between mb-3">
              <div>
                <h3 className="font-medium">{selectedPreset?.label}</h3>
                <p className="text-sm text-muted-foreground mt-1">{selectedPreset?.description}</p>
              </div>
            </div>

            <div className="mb-4">
              <h4 className="text-xs font-medium text-muted-foreground mb-2">{t("presets.capabilities")}</h4>
              <div className="flex flex-wrap gap-1.5">
                {selectedCapabilities.map((capKey) => {
                  const config = CAPABILITY_CONFIG[capKey]
                  return (
                    <Badge key={capKey} variant="outline" className={cn("text-xs", config?.color)}>
                      {tStages(capKey)}
                    </Badge>
                  )
                })}
              </div>
            </div>

            {matchingWorkflows.length > 0 && (
              <div>
                <h4 className="text-xs font-medium text-muted-foreground mb-2">{t("presets.includedWorkflows")}</h4>
                <div className="space-y-2">
                  {matchingWorkflows.map((workflow) => {
                    const checked = selectedWorkflowIds.includes(workflow.id)
                    return (
                      <label key={workflow.id} className="flex items-center gap-3 rounded-md border p-3 cursor-pointer hover:bg-muted/20">
                        <Checkbox
                          checked={checked}
                          onCheckedChange={(value) => handleWorkflowToggle(workflow.id, Boolean(value))}
                          disabled={disabled}
                        />
                        <div className="min-w-0 flex-1">
                          <p className="text-sm font-medium truncate">{workflow.title || workflow.name}</p>
                          {workflow.description && (
                            <p className="text-xs text-muted-foreground truncate">{workflow.description}</p>
                          )}
                        </div>
                      </label>
                    )
                  })}
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
