"use client";
import React from "react";
import { IconX } from "@/components/icons";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { ScheduledScan } from "@/types/scheduled-scan.types";
import type { Target } from "@/types/target.types";
type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string;
type CronPreset = {
    label: string;
    value: string;
};
interface EditScheduledScanDialogHeaderProps {
    t: TranslationFn;
}
export function EditScheduledScanDialogHeader({ t }: EditScheduledScanDialogHeaderProps) {
    return (<DialogHeader>
      <DialogTitle>{t("editTitle")}</DialogTitle>
      <DialogDescription>{t("editDesc")}</DialogDescription>
    </DialogHeader>);
}
interface EditScheduledScanNameFieldProps {
    t: TranslationFn;
    name: string;
    onNameChange: (value: string) => void;
}
export function EditScheduledScanNameField({ t, name, onNameChange, }: EditScheduledScanNameFieldProps) {
    return (<div className="gap-4 grid">
      <div className="gap-2 grid">
        <Label htmlFor="edit-name">{t("form.taskName")} *</Label>
        <Input id="edit-name" name="taskName" autoComplete="off" placeholder={t("form.taskNamePlaceholder")} value={name} onChange={(event) => onNameChange(event.target.value)}/>
      </div>
    </div>);
}
interface EditScheduledScanTargetSectionProps {
    t: TranslationFn;
    scheduledScan: ScheduledScan;
    targets: Target[];
    selectedTargetId: number | null;
    onSelectTarget: (targetId: number) => void;
}
export function EditScheduledScanTargetSection({ t, scheduledScan, targets, selectedTargetId, onSelectTarget, }: EditScheduledScanTargetSectionProps) {
    if (scheduledScan.scanMode === "organization") {
        return (<div className="gap-2 grid">
        <Label>{t("form.scanScope")}</Label>
        <div className="bg-muted/50 border p-3 rounded-md">
          <div className="flex gap-2 items-center">
            <Badge variant="secondary">{t("form.organizationMode")}</Badge>
            <span className="font-medium">{scheduledScan.organizationName}</span>
          </div>
          <p className="mt-2 text-muted-foreground text-xs">
            {t("form.organizationModeHint")}
          </p>
        </div>
      </div>);
    }
    return (<div className="gap-2 grid">
      <Label>{t("form.scanScope")}</Label>
      <div className="border max-h-[150px] overflow-y-auto p-3 rounded-md">
        {targets.length === 0 ? (<p className="text-muted-foreground text-sm">{t("form.noAvailableTarget")}</p>) : (<div className="flex flex-wrap gap-2">
            {targets.map((target) => (<Badge key={target.id} variant={selectedTargetId === target.id ? "default" : "outline"} className="cursor-pointer" render={<button type="button" onClick={() => onSelectTarget(target.id)}/>}>
                  {target.name}
                  {selectedTargetId === target.id && (<IconX className="h-3 ml-1 w-3"/>)}
                </Badge>))}
          </div>)}
      </div>
      {selectedTargetId && (<p className="text-muted-foreground text-xs">
          {t("form.selected")}: {targets.find((item) => item.id === selectedTargetId)?.name}
        </p>)}
    </div>);
}
interface EditScheduledScanCronSectionProps {
    t: TranslationFn;
    cronExpression: string;
    onCronChange: (value: string) => void;
    cronPresets: CronPreset[];
    onPresetSelect: (value: string) => void;
    disabled?: boolean;
}
export function EditScheduledScanCronSection({ t, cronExpression, onCronChange, cronPresets, onPresetSelect, disabled = false, }: EditScheduledScanCronSectionProps) {
    return (<div className="gap-4 grid">
      <div className="gap-2 grid">
        <Label>{t("form.cronExpression")} *</Label>
        <Input name="cronExpression" autoComplete="off" placeholder={t("form.cronPlaceholder")} value={cronExpression} onChange={(event) => onCronChange(event.target.value)} className="font-mono" disabled={disabled}/>
        <p className="text-muted-foreground text-xs">{t("form.cronFormat")}</p>
      </div>

      <div className="gap-2 grid">
        <Label className="text-muted-foreground text-xs">{t("form.quickSelect")}</Label>
        <div className="flex flex-wrap gap-2">
          {cronPresets.map((preset) => (<Badge key={preset.value} variant={cronExpression === preset.value ? "default" : "outline"} className="cursor-pointer" render={<button type="button" onClick={() => onPresetSelect(preset.value)} disabled={disabled}/>}>
                {preset.label}
              </Badge>))}
        </div>
      </div>
    </div>);
}
interface EditScheduledScanDialogFooterProps {
    t: TranslationFn;
    isPending: boolean;
}
export function EditScheduledScanDialogFooter({ t, isPending, }: EditScheduledScanDialogFooterProps) {
    return (<div className="flex justify-end gap-2">
      <Button type="submit" disabled={isPending} loading={isPending} loadingLabel={t("buttons.saveChanges")}>
        {t("buttons.saveChanges")}
      </Button>
    </div>);
}
