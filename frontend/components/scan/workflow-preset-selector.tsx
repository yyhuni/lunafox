"use client"

import React, { useMemo, useCallback } from "react"
import { Check, Play, Server, Settings, Zap } from "@/components/icons"
import { useTranslations } from "next-intl"

import { Badge } from "@/components/ui/badge"
import { Checkbox } from "@/components/ui/checkbox"
import { cn } from "@/lib/utils"
import { CAPABILITY_CONFIG, parseWorkflowCapabilities, mergeWorkflowConfigurations } from "@/lib/workflow-config"
import { usePresetWorkflows } from "@/hooks/use-workflows"
import { LoadingSpinner } from "@/components/loading-spinner"

import type { ScanWorkflow } from "@/types/workflow.types"

export interface WorkflowPreset {
  id: string
  label: string
  description: string
  icon: React.ComponentType<{ className?: string }>
  workflowIds: number[]
}

interface WorkflowPresetSelectorProps {
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

export function WorkflowPresetSelector({
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
}: WorkflowPresetSelectorProps) {
  const t = useTranslations("scan.initiate")
  const tStages = useTranslations("scan.progress.stages")
  
  // Get preset workflows from backend
  const { data: backendPresets, isLoading: isLoadingPresets } = usePresetWorkflows()

  // Convert backend presets to component format
  const workflowPresets = useMemo(() => {
    const presets: WorkflowPreset[] = []
    
    // Add backend presets
    if (backendPresets && backendPresets.length > 0) {
      backendPresets.forEach((preset) => {
        // Parse capabilities from preset configuration
        const caps = parseWorkflowCapabilities(preset.configuration || "")
        
        // Find matching workflow templates based on capabilities
        const matchingWorkflowIds: number[] = []
        if (workflows && workflows.length > 0) {
          workflows.forEach((workflow) => {
            const workflowCaps = parseWorkflowCapabilities(workflow.configuration || "")
            // Check if the workflow template has all capabilities required by preset
            const hasAllCaps = caps.every(cap => workflowCaps.includes(cap))
            if (hasAllCaps && caps.length > 0) {
              matchingWorkflowIds.push(workflow.id)
            }
          })
        }
        
        // Choose icon based on capabilities
        let Icon = Server
        if (caps.includes("vuln_scan")) {
          Icon = caps.some(c => ["subdomain_discovery", "port_scan", "site_scan"].includes(c)) ? Zap : Play
        }
        
        presets.push({
          id: preset.id,
          label: preset.name,
          description: preset.description || "",
          icon: Icon,
          workflowIds: matchingWorkflowIds,
        })
      })
    }
    
    // Add custom option at the end
    presets.push({
      id: "custom",
      label: t("presets.custom"),
      description: t("presets.customDesc"),
      icon: Settings,
      workflowIds: [],
    })
    
    return presets
  }, [backendPresets, workflows, t])

  const selectedWorkflows = useMemo(() => {
    if (!selectedWorkflowIds.length || !workflows) return []
    return workflows.filter((item) => selectedWorkflowIds.includes(item.id))
  }, [selectedWorkflowIds, workflows])

  const selectedCapabilities = useMemo(() => {
    if (!selectedWorkflows.length) return []
    const allCaps = new Set<string>()
    selectedWorkflows.forEach((workflow) => {
      parseWorkflowCapabilities(workflow.configuration || "").forEach((cap) => allCaps.add(cap))
    })
    return Array.from(allCaps)
  }, [selectedWorkflows])

  // Get currently selected preset details
  const selectedPreset = useMemo(() => {
    return workflowPresets.find((item) => item.id === selectedPresetId)
  }, [workflowPresets, selectedPresetId])

  // Get workflow templates for the selected preset
  const matchingWorkflows = useMemo(() => {
    if (!selectedPreset || selectedPreset.id === "custom") return []
    return workflows?.filter((item) => selectedPreset.workflowIds.includes(item.id)) || []
  }, [selectedPreset, workflows])
  
  // Update configuration when selected workflow templates change
  const updateConfigurationFromWorkflows = useCallback((workflowIds: number[]) => {
    if (!workflows) return
    const selectedItems = workflows.filter((item) => workflowIds.includes(item.id))
    const mergedConfig = mergeWorkflowConfigurations(selectedItems.map((item) => item.configuration || ""))
    onConfigurationChange(mergedConfig)
  }, [workflows, onConfigurationChange])

  const handlePresetSelect = useCallback((preset: WorkflowPreset) => {
    onPresetChange(preset.id)
    if (preset.id !== "custom") {
      // For backend presets, use preset configuration directly
      const backendPreset = backendPresets?.find(p => p.id === preset.id)
      if (backendPreset && backendPreset.configuration) {
        // Use preset configuration directly
        onConfigurationChange(backendPreset.configuration)
        // Also select matching workflow templates if available
        if (preset.workflowIds.length > 0) {
          onWorkflowIdsChange(preset.workflowIds)
        } else {
          // If no matching workflow template, clear selection
          onWorkflowIdsChange([])
        }
      } else {
        // Fallback to workflow-template-based configuration
        onWorkflowIdsChange(preset.workflowIds)
        updateConfigurationFromWorkflows(preset.workflowIds)
      }
    } else {
      // Custom mode - keep current selection or clear
      if (selectedWorkflowIds.length === 0) {
        onConfigurationChange("")
      }
    }
  }, [onPresetChange, onWorkflowIdsChange, updateConfigurationFromWorkflows, selectedWorkflowIds.length, onConfigurationChange, backendPresets])

  const handleWorkflowToggle = useCallback((workflowID: number, checked: boolean) => {
    let newWorkflowIds: number[]
    if (checked) {
      newWorkflowIds = [...selectedWorkflowIds, workflowID]
    } else {
      newWorkflowIds = selectedWorkflowIds.filter((id) => id !== workflowID)
    }
    onWorkflowIdsChange(newWorkflowIds)
    updateConfigurationFromWorkflows(newWorkflowIds)
  }, [selectedWorkflowIds, onWorkflowIdsChange, updateConfigurationFromWorkflows])

  // Show loading state while fetching presets
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
        {/* Compact preset cards */}
        <div className="grid gap-3 mb-4 grid-cols-2 sm:grid-cols-3 lg:grid-cols-4">
          {workflowPresets.map((preset) => {
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
                    {matchedWorkflows.length} {t("presets.enginesCount")}
                  </span>
                )}
              </button>
            )
          })}
        </div>
        
        {/* Selected preset details */}
        {selectedPresetId && selectedPresetId !== "custom" && (
          <div className="border rounded-lg p-4 bg-muted/10">
            <div className="flex items-start justify-between mb-3">
              <div>
                <h3 className="font-medium">{selectedPreset?.label}</h3>
                <p className="text-sm text-muted-foreground mt-1">{selectedPreset?.description}</p>
              </div>
            </div>
            
            {/* Capabilities */}
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
            
            {/* Workflows list */}
            <div>
              <h4 className="text-xs font-medium text-muted-foreground mb-2">{t("presets.usedEngines")}</h4>
              <div className="flex flex-wrap gap-2">
                {matchingWorkflows.length > 0 ? (
                  matchingWorkflows.map((workflow) => (
                    <span key={workflow.id} className="text-sm px-3 py-1.5 bg-background rounded-md border">
                      {workflow.name}
                    </span>
                  ))
                ) : (
                  <span className="text-xs text-muted-foreground">{t("presets.noMatchingWorkflows")}</span>
                )}
              </div>
            </div>
          </div>
        )}
        
        {/* Custom mode workflow template selection */}
        {selectedPresetId === "custom" && (
          <div className="border rounded-lg p-4 bg-muted/10">
            <div className="flex items-start justify-between mb-3">
              <div>
                <h3 className="font-medium">{selectedPreset?.label}</h3>
                <p className="text-sm text-muted-foreground mt-1">{selectedPreset?.description}</p>
                <p className="text-xs text-muted-foreground mt-2">{t("presets.customHint")}</p>
              </div>
            </div>
            
            {/* Capabilities - dynamically calculated from selected workflow templates */}
            <div className="mb-4">
              <h4 className="text-xs font-medium text-muted-foreground mb-2">{t("presets.capabilities")}</h4>
              <div className="flex flex-wrap gap-1.5">
                {selectedCapabilities.length > 0 ? (
                  selectedCapabilities.map((capKey) => {
                    const config = CAPABILITY_CONFIG[capKey]
                    return (
                      <Badge key={capKey} variant="outline" className={cn("text-xs", config?.color)}>
                        {tStages(capKey)}
                      </Badge>
                    )
                  })
                ) : (
                  <span className="text-xs text-muted-foreground">{t("presets.noCapabilities")}</span>
                )}
              </div>
            </div>
            
            {/* Workflow template list - selectable */}
            <div>
              <h4 className="text-xs font-medium text-muted-foreground mb-2">{t("presets.usedEngines")}</h4>
              <div className="flex flex-wrap gap-2">
                {workflows?.map((workflow) => {
                  const isSelected = selectedWorkflowIds.includes(workflow.id)
                  return (
                    <label
                      key={workflow.id}
                      htmlFor={`preset-workflow-${workflow.id}`}
                      className={cn(
                        "flex items-center gap-2 px-3 py-1.5 rounded-md cursor-pointer transition-[background-color,border-color,color,box-shadow] border",
                        isSelected
                          ? "bg-primary/10 border-primary/30"
                          : "hover:bg-muted/50 border-border",
                        disabled && "opacity-50 cursor-not-allowed"
                      )}
                    >
                      <Checkbox
                        id={`preset-workflow-${workflow.id}`}
                        checked={isSelected}
                        onCheckedChange={(checked) => {
                          handleWorkflowToggle(workflow.id, checked as boolean)
                        }}
                        disabled={disabled}
                        className="h-4 w-4"
                      />
                      <span className="text-sm">{workflow.name}</span>
                    </label>
                  )
                })}
              </div>
            </div>
          </div>
        )}
        
        {/* Empty state */}
        {!selectedPresetId && (
          <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
            <Server className="h-12 w-12 mb-4 opacity-50" />
            <p className="text-sm">{t("presets.selectHint")}</p>
          </div>
        )}
      </div>
    </div>
  )
}
