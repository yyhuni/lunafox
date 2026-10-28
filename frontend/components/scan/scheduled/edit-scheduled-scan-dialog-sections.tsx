"use client";
import React from "react";
import { IconCheck, IconClock, semanticIcons } from "@/components/icons";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { ScheduledScanTimeZoneField } from "@/components/scan/scheduled/scheduled-scan-time-zone-field";
import { isCronExpressionValid } from "@/lib/scheduled-scan-helpers";
import type { ScheduledScanSelectionMode } from "@/lib/scheduled-scan-helpers";
import type { Organization } from "@/types/organization.types";
import type { CursorPaginationNavigation } from "@/types/data-table.types";
import { OrganizationSelectionWorkspace } from "@/components/organization/organization-selection-workspace";
import { TargetSelectionWorkspace } from "@/components/target/target-selection-workspace";
import type { Target } from "@/types/target.types";
import { compactSurfaceClassNames } from "@/lib/ui/compact-surface-contract";
import { cn } from "@/lib/utils";
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
    return (<div className="gap-3 grid">
      <div className="gap-2 grid">
        <Label htmlFor="edit-name">{t("form.taskName")} *</Label>
        <Input id="edit-name" name="taskName" size="sm" autoComplete="off" placeholder={t("form.taskNamePlaceholder")} value={name} onChange={(event) => onNameChange(event.target.value)}/>
      </div>
    </div>);
}
interface EditScheduledScanTargetSectionProps {
    t: TranslationFn;
    selectionMode: ScheduledScanSelectionMode;
    onSelectionModeChange: (mode: ScheduledScanSelectionMode) => void;
    organizations: Organization[];
    targets: Target[];
    selectedOrgId: number | null;
    selectedTargetId: number | null;
    orgSearchInput: string;
    onOrgSearchChange: (value: string) => void;
    orgPageSize: number;
    onOrgPageSizeChange: React.Dispatch<React.SetStateAction<number>>;
    organizationTotalCount: number;
    organizationPaginationNavigation: CursorPaginationNavigation;
    onFirstOrgPage: () => void;
    onPreviousOrgPage: () => void;
    onNextOrgPage: () => void;
    targetSearchInput: string;
    onTargetSearchChange: (value: string) => void;
    targetPageSize: number;
    onTargetPageSizeChange: React.Dispatch<React.SetStateAction<number>>;
    targetTotalCount: number;
    targetPaginationNavigation: CursorPaginationNavigation;
    onFirstTargetPage: () => void;
    onPreviousTargetPage: () => void;
    onNextTargetPage: () => void;
    isOrgFetching: boolean;
    isTargetFetching: boolean;
    onSelectOrganization: (organizationId: number) => void;
    onSelectTarget: (targetId: number) => void;
    onClearOrganization: () => void;
    onClearTarget: () => void;
}
export function EditScheduledScanTargetSection({
    t,
    selectionMode,
    onSelectionModeChange,
    organizations,
    targets,
    selectedOrgId,
    selectedTargetId,
    orgSearchInput,
    onOrgSearchChange,
    orgPageSize,
    onOrgPageSizeChange,
    organizationTotalCount,
    organizationPaginationNavigation,
    onFirstOrgPage,
    onPreviousOrgPage,
    onNextOrgPage,
    targetSearchInput,
    onTargetSearchChange,
    targetPageSize,
    onTargetPageSizeChange,
    targetTotalCount,
    targetPaginationNavigation,
    onFirstTargetPage,
    onPreviousTargetPage,
    onNextTargetPage,
    isOrgFetching,
    isTargetFetching,
    onSelectOrganization,
    onSelectTarget,
    onClearOrganization,
    onClearTarget,
}: EditScheduledScanTargetSectionProps) {
    const OrganizationIcon = semanticIcons.concept.organization;
    const TargetIcon = semanticIcons.concept.target;

    return (<div className="gap-3 grid">
      <Label>{t("form.scanScope")}</Label>
      <div className="gap-3 grid grid-cols-1 sm:grid-cols-2">
        <Button type="button" variant="outline" size="action-card" layout="actionTile" selected={selectionMode === "organization"} onClick={() => onSelectionModeChange("organization")} aria-pressed={selectionMode === "organization"}>
          <OrganizationIcon className="h-8 w-8 shrink-0" />
          <div className="min-w-0 text-center">
            <p className="font-medium truncate">{t("form.organizationScan")}</p>
            <p className="text-muted-foreground text-xs break-words">{t("form.organizationScanDesc")}</p>
          </div>
        </Button>
        <Button type="button" variant="outline" size="action-card" layout="actionTile" selected={selectionMode === "target"} onClick={() => onSelectionModeChange("target")} aria-pressed={selectionMode === "target"}>
          <TargetIcon className="h-8 w-8 shrink-0" />
          <div className="min-w-0 text-center">
            <p className="font-medium truncate">{t("form.targetScan")}</p>
            <p className="text-muted-foreground text-xs break-words">{t("form.targetScanDesc")}</p>
          </div>
        </Button>
      </div>
      <p className="text-muted-foreground text-sm break-words">
        {selectionMode === "organization" ? t("form.organizationScanHint") : t("form.targetScanHint")}
      </p>
      {selectionMode === "organization" ? (
        <OrganizationSelectionWorkspace
          id="edit-scheduled-scan-organization-workspace"
          title={t("form.selectOrganization")}
          hint={t("form.organizationScanHint")}
          selectionMode="single"
          showSelectionCount={false}
          organizations={organizations}
          selectedOrganizationIds={selectedOrgId === null ? [] : [String(selectedOrgId)]}
          totalCount={organizationTotalCount}
          canFirstPage={organizationPaginationNavigation.canFirstPage ?? false}
          canPreviousPage={organizationPaginationNavigation.canPreviousPage}
          canNextPage={organizationPaginationNavigation.canNextPage}
          pageSize={orgPageSize}
          pageSizeOptions={[8, 16, 24, 50]}
          searchQuery={orgSearchInput}
          isLoading={isOrgFetching}
          onSearchQueryChange={onOrgSearchChange}
          onFirstPage={onFirstOrgPage}
          onPreviousPage={onPreviousOrgPage}
          onNextPage={onNextOrgPage}
          onPageSizeChange={onOrgPageSizeChange}
          onToggleOrganization={(organization) => onSelectOrganization(organization.id)}
          onClearOrganizations={onClearOrganization}
        />
      ) : (
        <TargetSelectionWorkspace
          id="edit-scheduled-scan-target-workspace"
          title={t("form.selectTarget")}
          hint={t("form.targetScanHint")}
          targets={targets}
          selectedTargetId={selectedTargetId}
          totalCount={targetTotalCount}
          canFirstPage={targetPaginationNavigation.canFirstPage ?? false}
          canPreviousPage={targetPaginationNavigation.canPreviousPage}
          canNextPage={targetPaginationNavigation.canNextPage}
          pageSize={targetPageSize}
          pageSizeOptions={[10, 20, 50]}
          searchQuery={targetSearchInput}
          isLoading={isTargetFetching}
          onSearchQueryChange={onTargetSearchChange}
          onFirstPage={onFirstTargetPage}
          onPreviousPage={onPreviousTargetPage}
          onNextPage={onNextTargetPage}
          onPageSizeChange={onTargetPageSizeChange}
          onToggleTarget={(target) => onSelectTarget(target.id)}
          onClearTarget={onClearTarget}
        />
      )}
    </div>);
}
interface EditScheduledScanCronSectionProps {
    t: TranslationFn;
    timeZone: string;
    onTimeZoneChange: (value: string) => void;
    cronExpression: string;
    onCronChange: (value: string) => void;
    cronPresets: CronPreset[];
    onPresetSelect: (value: string) => void;
    getCronDescription: (value: string) => string;
    getNextExecutions: (cronExpression: string, timeZone: string) => string[];
    disabled?: boolean;
}
export function EditScheduledScanCronSection({ t, timeZone, onTimeZoneChange, cronExpression, onCronChange, cronPresets, onPresetSelect, getCronDescription, getNextExecutions, disabled = false, }: EditScheduledScanCronSectionProps) {
    return (<div className="gap-3 grid">
      <ScheduledScanTimeZoneField id="edit-scheduled-scan-time-zone" value={timeZone} onChange={onTimeZoneChange} disabled={disabled} label={t("form.timeZone")} placeholder={t("form.timeZonePlaceholder")} searchPlaceholder={t("form.timeZoneSearchPlaceholder")} emptyLabel={t("form.timeZoneEmpty")} description={t("form.timeZoneDesc")}/>
      <div className="gap-2 grid">
        <Label>{t("form.cronExpression")} *</Label>
        <Input name="cronExpression" size="sm" autoComplete="off" placeholder={t("form.cronPlaceholder")} value={cronExpression} onChange={(event) => onCronChange(event.target.value)} className="font-mono" disabled={disabled}/>
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

      <div className={cn(compactSurfaceClassNames.mutedInfo, "grid min-w-0 gap-3 overflow-hidden")}>
        <div className="flex gap-2 items-center min-w-0">
          <IconClock className="h-4 shrink-0 text-muted-foreground w-4"/>
          <span className="font-medium min-w-0 flex-1 truncate">{t("form.executionPreview")}</span>
          {isCronExpressionValid(cronExpression) && (<Badge variant="secondary" className="shrink-0"><IconCheck className="h-3 mr-1 w-3"/>{t("form.valid")}</Badge>)}
        </div>
        <p className="break-words text-sm">{t("form.scheduleRule", { rule: getCronDescription(cronExpression), timeZone: timeZone || t("form.timeZonePlaceholder") })}</p>
        <Separator />
        <div className="min-w-0 space-y-1">
          <p className="text-muted-foreground text-xs">{t("form.nextExecutionTime")}</p>
          {getNextExecutions(cronExpression, timeZone).map((time, index) => (<p key={index} className="break-words text-sm">• {time}{index === 0 && <span className="ml-2 text-muted-foreground">{t("form.upcoming")}</span>}</p>))}
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
      <Button type="submit" size="sm" disabled={isPending} loading={isPending} loadingLabel={t("buttons.saveChanges")}>
        {t("buttons.saveChanges")}
      </Button>
    </div>);
}
