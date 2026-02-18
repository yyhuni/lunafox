import React from "react"
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
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandItem,
  CommandList,
} from "@/components/ui/command"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Separator } from "@/components/ui/separator"
import { EnginePresetSelector } from "@/components/scan/engine-preset-selector"
import { ScanConfigEditor } from "@/components/scan/scan-config-editor"
import { cn } from "@/lib/utils"
import { isCronExpressionValid } from "@/lib/scheduled-scan-helpers"
import type { ScheduledScanSelectionMode } from "@/lib/scheduled-scan-helpers"
import type { Organization } from "@/types/organization.types"
import type { Target } from "@/types/target.types"
import type { ScanEngine } from "@/types/engine.types"
import {
  IconX,
  IconLoader2,
  IconChevronRight,
  IconChevronLeft,
  IconCheck,
  IconBuilding,
  IconTarget,
  IconClock,
  IconSearch,
} from "@/components/icons"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

interface ScheduledScanBasicInfoStepProps {
  t: TranslationFn
  name: string
  setName: (value: string) => void
  selectionMode: ScheduledScanSelectionMode
  setSelectionMode: (mode: ScheduledScanSelectionMode) => void
  setSelectedOrgId: (value: number | null) => void
  setSelectedTargetId: (value: number | null) => void
}

export function ScheduledScanBasicInfoStep({
  t,
  name,
  setName,
  selectionMode,
  setSelectionMode,
  setSelectedOrgId,
  setSelectedTargetId,
}: ScheduledScanBasicInfoStepProps) {
  return (
    <div className="p-6 space-y-6 overflow-y-auto h-full">
      <div className="space-y-2">
        <Label htmlFor="name">{t("form.taskName")} *</Label>
        <Input
          id="name"
          name="taskName"
          autoComplete="off"
          placeholder={t("form.taskNamePlaceholder")}
          value={name}
          onChange={(event) => setName(event.target.value)}
        />
        <p className="text-xs text-muted-foreground">{t("form.taskNameDesc")}</p>
      </div>
      <Separator />
      <div className="space-y-3">
        <Label>{t("form.selectScanMode")}</Label>
        <div className="grid grid-cols-2 gap-4">
          <button
            type="button"
            className={cn(
              "flex flex-col items-center gap-3 p-4 border-2 rounded-lg cursor-pointer transition-colors",
              selectionMode === "organization" ? "border-primary bg-primary/5" : "border-muted hover:border-muted-foreground/50"
            )}
            onClick={() => {
              setSelectionMode("organization")
              setSelectedTargetId(null)
            }}
            aria-pressed={selectionMode === "organization"}
          >
            <IconBuilding className="h-8 w-8" />
            <div className="text-center">
              <p className="font-medium">{t("form.organizationScan")}</p>
              <p className="text-xs text-muted-foreground">{t("form.organizationScanDesc")}</p>
            </div>
            {selectionMode === "organization" && <IconCheck className="h-5 w-5 text-primary" />}
          </button>
          <button
            type="button"
            className={cn(
              "flex flex-col items-center gap-3 p-4 border-2 rounded-lg cursor-pointer transition-colors",
              selectionMode === "target" ? "border-primary bg-primary/5" : "border-muted hover:border-muted-foreground/50"
            )}
            onClick={() => {
              setSelectionMode("target")
              setSelectedOrgId(null)
            }}
            aria-pressed={selectionMode === "target"}
          >
            <IconTarget className="h-8 w-8" />
            <div className="text-center">
              <p className="font-medium">{t("form.targetScan")}</p>
              <p className="text-xs text-muted-foreground">{t("form.targetScanDesc")}</p>
            </div>
            {selectionMode === "target" && <IconCheck className="h-5 w-5 text-primary" />}
          </button>
        </div>
        <p className="text-sm text-muted-foreground">
          {selectionMode === "organization" ? t("form.organizationScanHint") : t("form.targetScanHint")}
        </p>
      </div>
    </div>
  )
}

interface ScheduledScanPresetInfoStepProps {
  t: TranslationFn
  name: string
  setName: (value: string) => void
  presetTargetName?: string
  presetOrganizationName?: string
  presetTargetId?: number
}

export function ScheduledScanPresetInfoStep({
  t,
  name,
  setName,
  presetTargetName,
  presetOrganizationName,
  presetTargetId,
}: ScheduledScanPresetInfoStepProps) {
  return (
    <div className="p-6 space-y-6 overflow-y-auto h-full">
      <div className="space-y-2">
        <Label htmlFor="name">{t("form.taskName")} *</Label>
        <Input
          id="name"
          name="taskName"
          autoComplete="off"
          placeholder={t("form.taskNamePlaceholder")}
          value={name}
          onChange={(event) => setName(event.target.value)}
        />
        <p className="text-xs text-muted-foreground">{t("form.taskNameDesc")}</p>
      </div>
      <Separator />
      <div className="space-y-3">
        <Label>{t("form.scanTarget")}</Label>
        <div className="flex items-center gap-2 p-4 border rounded-lg bg-muted/50">
          <IconTarget className="h-5 w-5 text-muted-foreground" />
          <span className="font-medium">{presetTargetName || presetOrganizationName}</span>
          <Badge variant="secondary" className="ml-auto">
            {presetTargetId ? t("form.targetScan") : t("form.organizationScan")}
          </Badge>
        </div>
        <p className="text-xs text-muted-foreground">{t("form.presetTargetHint")}</p>
      </div>
    </div>
  )
}

interface ScheduledScanTargetSelectionStepProps {
  t: TranslationFn
  selectionMode: ScheduledScanSelectionMode
  orgSearchInput: string
  setOrgSearchInput: (value: string) => void
  targetSearchInput: string
  setTargetSearchInput: (value: string) => void
  handleOrgSearch: () => void
  handleTargetSearch: () => void
  isOrgFetching: boolean
  isTargetFetching: boolean
  organizations: Organization[]
  targets: Target[]
  selectedOrgId: number | null
  selectedTargetId: number | null
  setSelectedOrgId: (value: number | null) => void
  setSelectedTargetId: (value: number | null) => void
  onSelectOrg: (id: number) => void
  onSelectTarget: (id: number) => void
}

export function ScheduledScanTargetSelectionStep({
  t,
  selectionMode,
  orgSearchInput,
  setOrgSearchInput,
  targetSearchInput,
  setTargetSearchInput,
  handleOrgSearch,
  handleTargetSearch,
  isOrgFetching,
  isTargetFetching,
  organizations,
  targets,
  selectedOrgId,
  selectedTargetId,
  setSelectedOrgId,
  setSelectedTargetId,
  onSelectOrg,
  onSelectTarget,
}: ScheduledScanTargetSelectionStepProps) {
  return (
    <div className="p-6 space-y-4 overflow-y-auto h-full">
      {selectionMode === "organization" ? (
        <>
          <Label>{t("form.selectOrganization")}</Label>
          <div className="flex items-center gap-2 mb-2">
            <Input
              type="search"
              name="organizationSearch"
              autoComplete="off"
              placeholder={t("form.searchOrganization")}
              value={orgSearchInput}
              onChange={(event) => setOrgSearchInput(event.target.value)}
              onKeyDown={(event) => event.key === "Enter" && handleOrgSearch()}
              className="h-9 flex-1"
            />
            <Button
              type="button"
              variant="outline"
              size="icon"
              className="h-9 w-9"
              onClick={handleOrgSearch}
              disabled={isOrgFetching}
              aria-label={t("form.searchOrganization")}
            >
              {isOrgFetching ? <IconLoader2 className="h-4 w-4 animate-spin" /> : <IconSearch className="h-4 w-4" />}
            </Button>
          </div>
          <Command className="border rounded-lg" shouldFilter={false}>
            <CommandList className="max-h-[250px]">
              {organizations.length === 0 ? (
                <CommandEmpty>{t("form.noOrganization")}</CommandEmpty>
              ) : (
                <CommandGroup>
                  {organizations.map((org) => (
                    <CommandItem
                      key={org.id}
                      value={org.id.toString()}
                      onSelect={() => onSelectOrg(org.id)}
                      className="flex items-center justify-between"
                    >
                      <div className="flex items-center gap-2">
                        <Checkbox checked={selectedOrgId === org.id} onCheckedChange={() => onSelectOrg(org.id)} />
                        <span>{org.name}</span>
                      </div>
                      <span className="text-xs text-muted-foreground">{t("form.targetCount", { count: org.targetCount || 0 })}</span>
                    </CommandItem>
                  ))}
                </CommandGroup>
              )}
            </CommandList>
          </Command>
          {selectedOrgId && (
            <div className="space-y-2">
              <p className="text-sm text-muted-foreground">{t("form.selectedOrganization")}</p>
              <Badge variant="secondary">
                {organizations.find((org) => org.id === selectedOrgId)?.name}
                <button
                  type="button"
                  className="ml-1 inline-flex items-center justify-center rounded-sm hover:bg-muted/60"
                  onClick={() => setSelectedOrgId(null)}
                  aria-label={t("form.clearSelectedOrganization")}
                >
                  <IconX className="h-3 w-3" />
                </button>
              </Badge>
            </div>
          )}
        </>
      ) : (
        <>
          <Label>{t("form.selectTarget")}</Label>
          <div className="flex items-center gap-2 mb-2">
            <Input
              type="search"
              name="targetSearch"
              autoComplete="off"
              placeholder={t("form.searchTarget")}
              value={targetSearchInput}
              onChange={(event) => setTargetSearchInput(event.target.value)}
              onKeyDown={(event) => event.key === "Enter" && handleTargetSearch()}
              className="h-9 flex-1"
            />
            <Button
              type="button"
              variant="outline"
              size="icon"
              className="h-9 w-9"
              onClick={handleTargetSearch}
              disabled={isTargetFetching}
              aria-label={t("form.searchTarget")}
            >
              {isTargetFetching ? <IconLoader2 className="h-4 w-4 animate-spin" /> : <IconSearch className="h-4 w-4" />}
            </Button>
          </div>
          <Command className="border rounded-lg" shouldFilter={false}>
            <CommandList className="max-h-[250px]">
              {targets.length === 0 ? (
                <CommandEmpty>{t("form.noTarget")}</CommandEmpty>
              ) : (
                <CommandGroup>
                  {targets.map((target) => (
                    <CommandItem
                      key={target.id}
                      value={target.id.toString()}
                      onSelect={() => onSelectTarget(target.id)}
                      className="flex items-center justify-between"
                    >
                      <div className="flex items-center gap-2">
                        <Checkbox checked={selectedTargetId === target.id} onCheckedChange={() => onSelectTarget(target.id)} />
                        <span>{target.name}</span>
                      </div>
                      {target.organizations && target.organizations.length > 0 && (
                        <span className="text-xs text-muted-foreground">{target.organizations.map((org) => org.name).join(", ")}</span>
                      )}
                    </CommandItem>
                  ))}
                </CommandGroup>
              )}
            </CommandList>
          </Command>
          {selectedTargetId && (
            <div className="space-y-2">
              <p className="text-sm text-muted-foreground">{t("form.selectedTarget")}</p>
              <Badge variant="outline">
                {targets.find((target) => target.id === selectedTargetId)?.name}
                <button
                  type="button"
                  className="ml-1 inline-flex items-center justify-center rounded-sm hover:bg-muted/60"
                  onClick={() => setSelectedTargetId(null)}
                  aria-label={t("form.clearSelectedTarget")}
                >
                  <IconX className="h-3 w-3" />
                </button>
              </Badge>
            </div>
          )}
        </>
      )}
    </div>
  )
}

interface ScheduledScanEngineStepProps {
  engines: ScanEngine[]
  engineIds: number[]
  selectedPresetId: string | null
  setSelectedPresetId: (value: string | null) => void
  onEngineIdsChange: (value: number[]) => void
  onConfigChange: (value: string) => void
  disabled: boolean
}

export function ScheduledScanEngineStep({
  engines,
  engineIds,
  selectedPresetId,
  setSelectedPresetId,
  onEngineIdsChange,
  onConfigChange,
  disabled,
}: ScheduledScanEngineStepProps) {
  if (engines.length === 0) return null
  return (
    <EnginePresetSelector
      engines={engines}
      selectedEngineIds={engineIds}
      selectedPresetId={selectedPresetId}
      onPresetChange={setSelectedPresetId}
      onEngineIdsChange={onEngineIdsChange}
      onConfigurationChange={onConfigChange}
      disabled={disabled}
    />
  )
}

interface ScheduledScanConfigStepProps {
  configuration: string
  onConfigChange: (value: string) => void
  onValidationChange: (value: boolean) => void
  selectedEngines: ScanEngine[]
  isConfigEdited: boolean
  disabled: boolean
}

export function ScheduledScanConfigStep({
  configuration,
  onConfigChange,
  onValidationChange,
  selectedEngines,
  isConfigEdited,
  disabled,
}: ScheduledScanConfigStepProps) {
  return (
    <ScanConfigEditor
      configuration={configuration}
      onChange={onConfigChange}
      onValidationChange={onValidationChange}
      selectedEngines={selectedEngines}
      isConfigEdited={isConfigEdited}
      disabled={disabled}
    />
  )
}

interface ScheduledScanScheduleStepProps {
  t: TranslationFn
  cronExpression: string
  setCronExpression: (value: string) => void
  cronPresets: Array<{ label: string; value: string }>
  getCronDescription: (value: string) => string
  getNextExecutions: (value: string) => string[]
}

export function ScheduledScanScheduleStep({
  t,
  cronExpression,
  setCronExpression,
  cronPresets,
  getCronDescription,
  getNextExecutions,
}: ScheduledScanScheduleStepProps) {
  return (
    <div className="p-6 space-y-6 overflow-y-auto h-full">
      <div className="space-y-2">
        <Label>{t("form.cronExpression")} *</Label>
        <Input
          name="cronExpression"
          autoComplete="off"
          placeholder={t("form.cronPlaceholder")}
          value={cronExpression}
          onChange={(event) => setCronExpression(event.target.value)}
          className="font-mono"
        />
        <p className="text-xs text-muted-foreground">{t("form.cronFormat")}</p>
      </div>
      <div className="space-y-2">
        <Label className="text-muted-foreground">{t("form.quickSelect")}</Label>
        <div className="flex flex-wrap gap-2">
          {cronPresets.map((preset) => (
            <Badge
              asChild
              key={preset.value}
              variant={cronExpression === preset.value ? "default" : "outline"}
              className="cursor-pointer"
            >
              <button
                type="button"
                onClick={() => setCronExpression(preset.value)}
              >
                {preset.label}
              </button>
            </Badge>
          ))}
        </div>
      </div>
      <div className="rounded-lg border bg-muted/50 p-4 space-y-3">
        <div className="flex items-center gap-2">
          <IconClock className="h-4 w-4 text-muted-foreground" />
          <span className="font-medium">{t("form.executionPreview")}</span>
          {isCronExpressionValid(cronExpression) && (
            <Badge variant="secondary" className="ml-auto"><IconCheck className="h-3 w-3 mr-1" />{t("form.valid")}</Badge>
          )}
        </div>
        <p className="text-sm">{getCronDescription(cronExpression)}</p>
        <Separator />
        <div className="space-y-1">
          <p className="text-xs text-muted-foreground">{t("form.nextExecutionTime")}</p>
          {getNextExecutions(cronExpression).map((time, index) => (
            <p key={index} className="text-sm">• {time}{index === 0 && <span className="text-muted-foreground ml-2">{t("form.upcoming")}</span>}</p>
          ))}
        </div>
      </div>
    </div>
  )
}

interface ScheduledScanFooterProps {
  t: TranslationFn
  currentStep: number
  totalSteps: number
  isPending: boolean
  onPrev: () => void
  onNext: () => void
  onSubmit: () => void
}

export function ScheduledScanFooter({
  t,
  currentStep,
  totalSteps,
  isPending,
  onPrev,
  onNext,
  onSubmit,
}: ScheduledScanFooterProps) {
  return (
    <div className="px-6 py-4 border-t flex justify-between">
      <Button variant="outline" onClick={onPrev} disabled={currentStep === 1}>
        <IconChevronLeft className="h-4 w-4 mr-1" />{t("buttons.previous")}
      </Button>
      {currentStep < totalSteps ? (
        <Button onClick={onNext}>{t("buttons.next")}<IconChevronRight className="h-4 w-4 ml-1" /></Button>
      ) : (
        <Button onClick={onSubmit} disabled={isPending}>
          {isPending && <IconLoader2 className="h-4 w-4 mr-1 animate-spin" />}{t("buttons.createTask")}
        </Button>
      )}
    </div>
  )
}

interface ScheduledScanOverwriteDialogProps {
  t: TranslationFn
  open: boolean
  onOpenChange: (value: boolean) => void
  onCancel: () => void
  onConfirm: () => void
}

export function ScheduledScanOverwriteDialog({
  t,
  open,
  onOpenChange,
  onCancel,
  onConfirm,
}: ScheduledScanOverwriteDialogProps) {
  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t("overwriteConfirm.title")}</AlertDialogTitle>
          <AlertDialogDescription>
            {t("overwriteConfirm.description")}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel onClick={onCancel}>
            {t("overwriteConfirm.cancel")}
          </AlertDialogCancel>
          <AlertDialogAction onClick={onConfirm}>
            {t("overwriteConfirm.confirm")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
