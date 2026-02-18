"use client"

import React from "react"
import { IconX, IconLoader2 } from "@/components/icons"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { cn } from "@/lib/utils"
import type { ScheduledScan } from "@/types/scheduled-scan.types"
import type { ScanEngine } from "@/types/engine.types"
import type { Target } from "@/types/target.types"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

type CronPreset = { label: string; value: string }

interface EditScheduledScanDialogHeaderProps {
  t: TranslationFn
}

export function EditScheduledScanDialogHeader({ t }: EditScheduledScanDialogHeaderProps) {
  return (
    <DialogHeader>
      <DialogTitle>{t("editTitle")}</DialogTitle>
      <DialogDescription>{t("editDesc")}</DialogDescription>
    </DialogHeader>
  )
}

interface EditScheduledScanNameFieldProps {
  t: TranslationFn
  name: string
  onNameChange: (value: string) => void
}

export function EditScheduledScanNameField({
  t,
  name,
  onNameChange,
}: EditScheduledScanNameFieldProps) {
  return (
    <div className="grid gap-4">
      <div className="grid gap-2">
        <Label htmlFor="edit-name">{t("form.taskName")} *</Label>
        <Input
          id="edit-name"
          name="taskName"
          autoComplete="off"
          placeholder={t("form.taskNamePlaceholder")}
          value={name}
          onChange={(event) => onNameChange(event.target.value)}
        />
      </div>
    </div>
  )
}

interface EditScheduledScanEngineSectionProps {
  t: TranslationFn
  engines: ScanEngine[]
  engineIds: number[]
  onToggle: (engineId: number, checked: boolean) => void
}

export function EditScheduledScanEngineSection({
  t,
  engines,
  engineIds,
  onToggle,
}: EditScheduledScanEngineSectionProps) {
  return (
    <div className="grid gap-2">
      <Label>{t("form.scanEngine")} *</Label>
      {engineIds.length > 0 && (
        <p className="text-xs text-muted-foreground">
          {t("form.selectedEngines", { count: engineIds.length })}
        </p>
      )}
      <div className="border rounded-md p-3 max-h-[150px] overflow-y-auto space-y-2">
        {engines.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t("form.noEngine")}</p>
        ) : (
          engines.map((engine) => (
            <label
              key={engine.id}
              htmlFor={`edit-engine-${engine.id}`}
              className={cn(
                "flex items-center gap-3 p-2 rounded-lg cursor-pointer transition-[background-color,border-color,box-shadow]",
                engineIds.includes(engine.id)
                  ? "bg-primary/10 border border-primary/30"
                  : "hover:bg-muted/50 border border-transparent"
              )}
            >
              <Checkbox
                id={`edit-engine-${engine.id}`}
                checked={engineIds.includes(engine.id)}
                onCheckedChange={(checked) => onToggle(engine.id, checked as boolean)}
              />
              <span className="text-sm">{engine.name}</span>
            </label>
          ))
        )}
      </div>
    </div>
  )
}

interface EditScheduledScanTargetSectionProps {
  t: TranslationFn
  scheduledScan: ScheduledScan
  targets: Target[]
  selectedTargetId: number | null
  onSelectTarget: (targetId: number) => void
}

export function EditScheduledScanTargetSection({
  t,
  scheduledScan,
  targets,
  selectedTargetId,
  onSelectTarget,
}: EditScheduledScanTargetSectionProps) {
  if (scheduledScan.scanMode === "organization") {
    return (
      <div className="grid gap-2">
        <Label>{t("form.scanScope")}</Label>
        <div className="border rounded-md p-3 bg-muted/50">
          <div className="flex items-center gap-2">
            <Badge variant="secondary">{t("form.organizationMode")}</Badge>
            <span className="font-medium">{scheduledScan.organizationName}</span>
          </div>
          <p className="text-xs text-muted-foreground mt-2">
            {t("form.organizationModeHint")}
          </p>
        </div>
      </div>
    )
  }

  return (
    <div className="grid gap-2">
      <Label>{t("form.scanScope")}</Label>
      <div className="border rounded-md p-3 max-h-[150px] overflow-y-auto">
        {targets.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t("form.noAvailableTarget")}</p>
        ) : (
          <div className="flex flex-wrap gap-2">
            {targets.map((target) => (
              <Badge
                asChild
                key={target.id}
                variant={selectedTargetId === target.id ? "default" : "outline"}
                className="cursor-pointer"
              >
                <button
                  type="button"
                  onClick={() => onSelectTarget(target.id)}
                >
                  {target.name}
                  {selectedTargetId === target.id && (
                    <IconX className="h-3 w-3 ml-1" />
                  )}
                </button>
              </Badge>
            ))}
          </div>
        )}
      </div>
      {selectedTargetId && (
        <p className="text-xs text-muted-foreground">
          {t("form.selected")}: {targets.find((item) => item.id === selectedTargetId)?.name}
        </p>
      )}
    </div>
  )
}

interface EditScheduledScanCronSectionProps {
  t: TranslationFn
  cronExpression: string
  onCronChange: (value: string) => void
  cronPresets: CronPreset[]
  onPresetSelect: (value: string) => void
}

export function EditScheduledScanCronSection({
  t,
  cronExpression,
  onCronChange,
  cronPresets,
  onPresetSelect,
}: EditScheduledScanCronSectionProps) {
  return (
    <div className="grid gap-4">
      <div className="grid gap-2">
        <Label>{t("form.cronExpression")} *</Label>
        <Input
          name="cronExpression"
          autoComplete="off"
          placeholder={t("form.cronPlaceholder")}
          value={cronExpression}
          onChange={(event) => onCronChange(event.target.value)}
          className="font-mono"
        />
        <p className="text-xs text-muted-foreground">{t("form.cronFormat")}</p>
      </div>

      <div className="grid gap-2">
        <Label className="text-xs text-muted-foreground">{t("form.quickSelect")}</Label>
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
                onClick={() => onPresetSelect(preset.value)}
              >
                {preset.label}
              </button>
            </Badge>
          ))}
        </div>
      </div>
    </div>
  )
}

interface EditScheduledScanDialogFooterProps {
  t: TranslationFn
  isPending: boolean
  onCancel: () => void
  onSubmit: () => void
}

export function EditScheduledScanDialogFooter({
  t,
  isPending,
  onCancel,
  onSubmit,
}: EditScheduledScanDialogFooterProps) {
  return (
    <DialogFooter>
      <Button variant="outline" onClick={onCancel}>
        {t("buttons.cancel")}
      </Button>
      <Button onClick={onSubmit} disabled={isPending}>
        {isPending && <IconLoader2 className="h-4 w-4 animate-spin" />}
        {t("buttons.saveChanges")}
      </Button>
    </DialogFooter>
  )
}
