"use client"

import React, { useMemo, useCallback } from "react"
import { Check, Play, Server, Settings, Zap } from "@/components/icons"
import { useTranslations } from "next-intl"

import { Badge } from "@/components/ui/badge"
import { Checkbox } from "@/components/ui/checkbox"
import { cn } from "@/lib/utils"
import { CAPABILITY_CONFIG, parseEngineCapabilities, mergeEngineConfigurations } from "@/lib/engine-config"
import { usePresetEngines } from "@/hooks/use-engines"
import { LoadingSpinner } from "@/components/loading-spinner"

import type { ScanEngine } from "@/types/engine.types"

export interface EnginePreset {
  id: string
  label: string
  description: string
  icon: React.ComponentType<{ className?: string }>
  engineIds: number[]
}

interface EnginePresetSelectorProps {
  engines: ScanEngine[]
  selectedEngineIds: number[]
  selectedPresetId: string | null
  onPresetChange: (presetId: string | null) => void
  onEngineIdsChange: (engineIds: number[]) => void
  onConfigurationChange: (config: string) => void
  disabled?: boolean
  className?: string
  showHeader?: boolean
  contentClassName?: string
}

export function EnginePresetSelector({
  engines,
  selectedEngineIds,
  selectedPresetId,
  onPresetChange,
  onEngineIdsChange,
  onConfigurationChange,
  disabled = false,
  className,
  showHeader = true,
  contentClassName,
}: EnginePresetSelectorProps) {
  const t = useTranslations("scan.initiate")
  const tStages = useTranslations("scan.progress.stages")
  
  // Get preset engines from backend
  const { data: backendPresets, isLoading: isLoadingPresets } = usePresetEngines()

  // Convert backend presets to component format
  const enginePresets = useMemo(() => {
    const presets: EnginePreset[] = []
    
    // Add backend presets
    if (backendPresets && backendPresets.length > 0) {
      backendPresets.forEach((preset) => {
        // Parse capabilities from preset configuration
        const caps = parseEngineCapabilities(preset.configuration || "")
        
        // Find matching engines based on capabilities
        const matchingEngineIds: number[] = []
        if (engines && engines.length > 0) {
          engines.forEach((engine) => {
            const engineCaps = parseEngineCapabilities(engine.configuration || "")
            // Check if engine has all capabilities required by preset
            const hasAllCaps = caps.every(cap => engineCaps.includes(cap))
            if (hasAllCaps && caps.length > 0) {
              matchingEngineIds.push(engine.id)
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
          engineIds: matchingEngineIds,
        })
      })
    }
    
    // Add custom option at the end
    presets.push({
      id: "custom",
      label: t("presets.custom"),
      description: t("presets.customDesc"),
      icon: Settings,
      engineIds: [],
    })
    
    return presets
  }, [backendPresets, engines, t])

  const selectedEngines = useMemo(() => {
    if (!selectedEngineIds.length || !engines) return []
    return engines.filter((e) => selectedEngineIds.includes(e.id))
  }, [selectedEngineIds, engines])

  const selectedCapabilities = useMemo(() => {
    if (!selectedEngines.length) return []
    const allCaps = new Set<string>()
    selectedEngines.forEach((engine) => {
      parseEngineCapabilities(engine.configuration || "").forEach((cap) => allCaps.add(cap))
    })
    return Array.from(allCaps)
  }, [selectedEngines])

  // Get currently selected preset details
  const selectedPreset = useMemo(() => {
    return enginePresets.find(p => p.id === selectedPresetId)
  }, [enginePresets, selectedPresetId])

  // Get engines for the selected preset
  const matchingEngines = useMemo(() => {
    if (!selectedPreset || selectedPreset.id === "custom") return []
    return engines?.filter(e => selectedPreset.engineIds.includes(e.id)) || []
  }, [selectedPreset, engines])
  
  // Update configuration when engines change
  const updateConfigurationFromEngines = useCallback((engineIds: number[]) => {
    if (!engines) return
    const selectedEngs = engines.filter(e => engineIds.includes(e.id))
    const mergedConfig = mergeEngineConfigurations(selectedEngs.map(e => e.configuration || ""))
    onConfigurationChange(mergedConfig)
  }, [engines, onConfigurationChange])

  const handlePresetSelect = useCallback((preset: EnginePreset) => {
    onPresetChange(preset.id)
    if (preset.id !== "custom") {
      // For backend presets, use preset configuration directly
      const backendPreset = backendPresets?.find(p => p.id === preset.id)
      if (backendPreset && backendPreset.configuration) {
        // Use preset configuration directly
        onConfigurationChange(backendPreset.configuration)
        // Also select matching engines if available
        if (preset.engineIds.length > 0) {
          onEngineIdsChange(preset.engineIds)
        } else {
          // If no matching engines, clear engine selection
          onEngineIdsChange([])
        }
      } else {
        // Fallback to engine-based configuration
        onEngineIdsChange(preset.engineIds)
        updateConfigurationFromEngines(preset.engineIds)
      }
    } else {
      // Custom mode - keep current selection or clear
      if (selectedEngineIds.length === 0) {
        onConfigurationChange("")
      }
    }
  }, [onPresetChange, onEngineIdsChange, updateConfigurationFromEngines, selectedEngineIds.length, onConfigurationChange, backendPresets])

  const handleEngineToggle = useCallback((engineId: number, checked: boolean) => {
    let newEngineIds: number[]
    if (checked) {
      newEngineIds = [...selectedEngineIds, engineId]
    } else {
      newEngineIds = selectedEngineIds.filter((id) => id !== engineId)
    }
    onEngineIdsChange(newEngineIds)
    updateConfigurationFromEngines(newEngineIds)
  }, [selectedEngineIds, onEngineIdsChange, updateConfigurationFromEngines])

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
            {selectedEngineIds.length > 0 && (
              <Badge variant="secondary" className="text-xs">
                {t("selectedCount", { count: selectedEngineIds.length })}
              </Badge>
            )}
          </div>
        )}
        {/* Compact preset cards */}
        <div className="grid gap-3 mb-4 grid-cols-2 sm:grid-cols-3 lg:grid-cols-4">
          {enginePresets.map((preset) => {
            const isActive = selectedPresetId === preset.id
            const PresetIcon = preset.icon
            const matchedEngines = preset.id === "custom" 
              ? [] 
              : engines?.filter(e => preset.engineIds.includes(e.id)) || []
            
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
                    {matchedEngines.length} {t("presets.enginesCount")}
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
            
            {/* Engines list */}
            <div>
              <h4 className="text-xs font-medium text-muted-foreground mb-2">{t("presets.usedEngines")}</h4>
              <div className="flex flex-wrap gap-2">
                {matchingEngines.length > 0 ? (
                  matchingEngines.map((engine) => (
                    <span key={engine.id} className="text-sm px-3 py-1.5 bg-background rounded-md border">
                      {engine.name}
                    </span>
                  ))
                ) : (
                  <span className="text-xs text-muted-foreground">{t("presets.noMatchingEngines")}</span>
                )}
              </div>
            </div>
          </div>
        )}
        
        {/* Custom mode engine selection */}
        {selectedPresetId === "custom" && (
          <div className="border rounded-lg p-4 bg-muted/10">
            <div className="flex items-start justify-between mb-3">
              <div>
                <h3 className="font-medium">{selectedPreset?.label}</h3>
                <p className="text-sm text-muted-foreground mt-1">{selectedPreset?.description}</p>
                <p className="text-xs text-muted-foreground mt-2">{t("presets.customHint")}</p>
              </div>
            </div>
            
            {/* Capabilities - dynamically calculated from selected engines */}
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
            
            {/* Engines list - selectable */}
            <div>
              <h4 className="text-xs font-medium text-muted-foreground mb-2">{t("presets.usedEngines")}</h4>
              <div className="flex flex-wrap gap-2">
                {engines?.map((engine) => {
                  const isSelected = selectedEngineIds.includes(engine.id)
                  return (
                    <label
                      key={engine.id}
                      htmlFor={`preset-engine-${engine.id}`}
                      className={cn(
                        "flex items-center gap-2 px-3 py-1.5 rounded-md cursor-pointer transition-[background-color,border-color,color,box-shadow] border",
                        isSelected
                          ? "bg-primary/10 border-primary/30"
                          : "hover:bg-muted/50 border-border",
                        disabled && "opacity-50 cursor-not-allowed"
                      )}
                    >
                      <Checkbox
                        id={`preset-engine-${engine.id}`}
                        checked={isSelected}
                        onCheckedChange={(checked) => {
                          handleEngineToggle(engine.id, checked as boolean)
                        }}
                        disabled={disabled}
                        className="h-4 w-4"
                      />
                      <span className="text-sm">{engine.name}</span>
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
