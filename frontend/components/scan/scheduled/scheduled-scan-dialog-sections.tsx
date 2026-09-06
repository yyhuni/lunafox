import React from "react";
import { AlertDialog, AlertDialogClose, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, } from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { OrganizationSelectionWorkspace } from "@/components/organization/organization-selection-workspace";
import { TargetSelectionWorkspace } from "@/components/target/target-selection-workspace";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { isCronExpressionValid } from "@/lib/scheduled-scan-helpers";
import type { ScheduledScanSelectionMode } from "@/lib/scheduled-scan-helpers";
import type { Organization } from "@/types/organization.types";
import type { Target } from "@/types/target.types";
import type { CursorPaginationNavigation } from "@/types/data-table.types";
import { IconChevronRight, IconChevronLeft, IconCheck, IconClock, semanticIcons, } from "@/components/icons";
import { cn } from "@/lib/utils";
type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string;
const OrganizationIcon = semanticIcons.concept.organization;
const TargetIcon = semanticIcons.concept.target;
const SCHEDULED_SCAN_ORGANIZATION_PICKER_PAGE_SIZE_OPTIONS = [8, 16, 24, 50];
const SCHEDULED_SCAN_TARGET_PICKER_PAGE_SIZE_OPTIONS = [10, 20, 50];
interface ScheduledScanOrganizationPickerProps {
    t: TranslationFn;
    isLoading: boolean;
    orgSearchInput: string;
    setOrgSearchInput: (value: string) => void;
    orgPageSize: number;
    setOrgPageSize: React.Dispatch<React.SetStateAction<number>>;
    organizationTotalCount: number;
    organizationPaginationNavigation: CursorPaginationNavigation;
    onFirstOrgPage: () => void;
    onPreviousOrgPage: () => void;
    onNextOrgPage: () => void;
    organizations: Organization[];
    selectedOrgId: number | null;
    setSelectedOrgId: (value: number | null) => void;
    onSelectOrg: (id: number) => void;
}
function ScheduledScanOrganizationPicker({ t, isLoading, orgSearchInput, setOrgSearchInput, orgPageSize, setOrgPageSize, organizationTotalCount, organizationPaginationNavigation, onFirstOrgPage, onPreviousOrgPage, onNextOrgPage, organizations, selectedOrgId, setSelectedOrgId, onSelectOrg, }: ScheduledScanOrganizationPickerProps) {
    return (<OrganizationSelectionWorkspace id="scheduled-scan-organization-workspace" title={t("form.selectOrganization")} hint={t("form.organizationScanHint")} selectionMode="single" showSelectionCount={false} organizations={organizations} selectedOrganizationIds={selectedOrgId === null ? [] : [String(selectedOrgId)]} totalCount={organizationTotalCount} canFirstPage={organizationPaginationNavigation.canFirstPage ?? false} canPreviousPage={organizationPaginationNavigation.canPreviousPage} canNextPage={organizationPaginationNavigation.canNextPage} pageSize={orgPageSize} pageSizeOptions={SCHEDULED_SCAN_ORGANIZATION_PICKER_PAGE_SIZE_OPTIONS} searchQuery={orgSearchInput} isLoading={isLoading} onSearchQueryChange={setOrgSearchInput} onFirstPage={onFirstOrgPage} onPreviousPage={onPreviousOrgPage} onNextPage={onNextOrgPage} onPageSizeChange={setOrgPageSize} onToggleOrganization={(organization) => onSelectOrg(organization.id)} onClearOrganizations={() => setSelectedOrgId(null)}/>);
}
interface ScheduledScanScopeStepProps {
    t: TranslationFn;
    className?: string;
    name: string;
    setName: (value: string) => void;
    selectionMode: ScheduledScanSelectionMode;
    setSelectionMode: (mode: ScheduledScanSelectionMode) => void;
    orgSearchInput: string;
    setOrgSearchInput: (value: string) => void;
    orgPageSize: number;
    setOrgPageSize: React.Dispatch<React.SetStateAction<number>>;
    organizationTotalCount: number;
    organizationPaginationNavigation: CursorPaginationNavigation;
    onFirstOrgPage: () => void;
    onPreviousOrgPage: () => void;
    onNextOrgPage: () => void;
    targetSearchInput: string;
    setTargetSearchInput: (value: string) => void;
    targetPageSize: number;
    setTargetPageSize: React.Dispatch<React.SetStateAction<number>>;
    targetTotalCount: number;
    targetPaginationNavigation: CursorPaginationNavigation;
    onFirstTargetPage: () => void;
    onPreviousTargetPage: () => void;
    onNextTargetPage: () => void;
    isOrgFetching: boolean;
    isTargetFetching: boolean;
    organizations: Organization[];
    targets: Target[];
    selectedOrgId: number | null;
    selectedTargetId: number | null;
    setSelectedOrgId: (value: number | null) => void;
    setSelectedTargetId: (value: number | null) => void;
    onSelectOrg: (id: number) => void;
    onSelectTarget: (id: number) => void;
}
export function ScheduledScanScopeStep({ t, className, name, setName, selectionMode, setSelectionMode, orgSearchInput, setOrgSearchInput, orgPageSize, setOrgPageSize, organizationTotalCount, organizationPaginationNavigation, onFirstOrgPage, onPreviousOrgPage, onNextOrgPage, targetSearchInput, setTargetSearchInput, targetPageSize, setTargetPageSize, targetTotalCount, targetPaginationNavigation, onFirstTargetPage, onPreviousTargetPage, onNextTargetPage, isOrgFetching, isTargetFetching, organizations, targets, selectedOrgId, selectedTargetId, setSelectedOrgId, setSelectedTargetId, onSelectOrg, onSelectTarget, }: ScheduledScanScopeStepProps) {
    return (<div className={cn("min-w-0 max-w-full space-y-6", className)}>
      <div className="space-y-2">
        <Label htmlFor="name">{t("form.taskName")} *</Label>
        <Input id="name" name="taskName" autoComplete="off" placeholder={t("form.taskNamePlaceholder")} value={name} onChange={(event) => setName(event.target.value)}/>
        <p className="text-muted-foreground text-xs">{t("form.taskNameDesc")}</p>
      </div>
      <Separator />
      <div className="space-y-3">
        <Label>{t("form.scanScope")}</Label>
        <div className="gap-4 grid grid-cols-1 sm:grid-cols-2">
          <Button type="button" variant="outline" size="action-card" layout="actionTile" selected={selectionMode === "organization"} onClick={() => {
            setSelectionMode("organization");
            setSelectedTargetId(null);
        }} aria-pressed={selectionMode === "organization"}>
            <OrganizationIcon className="h-8 w-8 shrink-0"/>
            <div className="min-w-0 text-center">
              <p className="font-medium truncate">{t("form.organizationScan")}</p>
              <p className="text-muted-foreground text-xs break-words">{t("form.organizationScanDesc")}</p>
            </div>
            {selectionMode === "organization" && <IconCheck className="h-5 text-primary w-5"/>}
          </Button>
          <Button type="button" variant="outline" size="action-card" layout="actionTile" selected={selectionMode === "target"} onClick={() => {
            setSelectionMode("target");
            setSelectedOrgId(null);
        }} aria-pressed={selectionMode === "target"}>
            <TargetIcon className="h-8 w-8 shrink-0"/>
            <div className="min-w-0 text-center">
              <p className="font-medium truncate">{t("form.targetScan")}</p>
              <p className="text-muted-foreground text-xs break-words">{t("form.targetScanDesc")}</p>
            </div>
            {selectionMode === "target" && <IconCheck className="h-5 text-primary w-5"/>}
          </Button>
        </div>
        <p className="text-muted-foreground text-sm break-words">
          {selectionMode === "organization" ? t("form.organizationScanHint") : t("form.targetScanHint")}
        </p>
      </div>

      <Separator />

      <div className="min-w-0 max-w-full space-y-4">
        {selectionMode === "organization" ? (<ScheduledScanOrganizationPicker t={t} isLoading={isOrgFetching} orgSearchInput={orgSearchInput} setOrgSearchInput={setOrgSearchInput} orgPageSize={orgPageSize} setOrgPageSize={setOrgPageSize} organizationTotalCount={organizationTotalCount} organizationPaginationNavigation={organizationPaginationNavigation} onFirstOrgPage={onFirstOrgPage} onPreviousOrgPage={onPreviousOrgPage} onNextOrgPage={onNextOrgPage} organizations={organizations} selectedOrgId={selectedOrgId} setSelectedOrgId={setSelectedOrgId} onSelectOrg={onSelectOrg}/>) : (<TargetSelectionWorkspace id="scheduled-scan-target-workspace" title={t("form.selectTarget")} hint={t("form.targetScanHint")} targets={targets} selectedTargetId={selectedTargetId} totalCount={targetTotalCount} canFirstPage={targetPaginationNavigation.canFirstPage ?? false} canPreviousPage={targetPaginationNavigation.canPreviousPage} canNextPage={targetPaginationNavigation.canNextPage} pageSize={targetPageSize} pageSizeOptions={SCHEDULED_SCAN_TARGET_PICKER_PAGE_SIZE_OPTIONS} searchQuery={targetSearchInput} isLoading={isTargetFetching} onSearchQueryChange={setTargetSearchInput} onFirstPage={onFirstTargetPage} onPreviousPage={onPreviousTargetPage} onNextPage={onNextTargetPage} onPageSizeChange={setTargetPageSize} onToggleTarget={(target) => onSelectTarget(target.id)} onClearTarget={() => setSelectedTargetId(null)}/>) }
      </div>
    </div>);
}
interface ScheduledScanPresetInfoStepProps {
    t: TranslationFn;
    className?: string;
    name: string;
    setName: (value: string) => void;
    presetTargetName?: string;
    presetOrganizationName?: string;
    presetTargetId?: number;
}
export function ScheduledScanPresetInfoStep({ t, className, name, setName, presetTargetName, presetOrganizationName, presetTargetId, }: ScheduledScanPresetInfoStepProps) {
    return (<div className={cn("min-w-0 max-w-full space-y-6", className)}>
      <div className="space-y-2">
        <Label htmlFor="name">{t("form.taskName")} *</Label>
        <Input id="name" name="taskName" autoComplete="off" placeholder={t("form.taskNamePlaceholder")} value={name} onChange={(event) => setName(event.target.value)}/>
        <p className="text-muted-foreground text-xs">{t("form.taskNameDesc")}</p>
      </div>
      <Separator />
      <div className="space-y-3">
        <Label>{t("form.scanTarget")}</Label>
        <div className="bg-muted/50 border flex min-w-0 gap-2 items-center overflow-hidden p-4 rounded-lg">
          <TargetIcon className="h-5 shrink-0 text-muted-foreground w-5"/>
          <span className="min-w-0 flex-1 truncate font-medium">{presetTargetName || presetOrganizationName}</span>
          <Badge variant="secondary" className="ml-auto shrink-0">
            {presetTargetId ? t("form.targetScan") : t("form.organizationScan")}
          </Badge>
        </div>
        <p className="text-muted-foreground text-xs break-words">{t("form.presetTargetHint")}</p>
      </div>
    </div>);
}
interface ScheduledScanScheduleStepProps {
    t: TranslationFn;
    className?: string;
    cronExpression: string;
    setCronExpression: (value: string) => void;
    cronPresets: Array<{
        label: string;
        value: string;
    }>;
    getCronDescription: (value: string) => string;
    getNextExecutions: (cronExpression: string) => string[];
    disabled?: boolean;
}
export function ScheduledScanScheduleStep({ t, className, cronExpression, setCronExpression, cronPresets, getCronDescription, getNextExecutions, disabled = false, }: ScheduledScanScheduleStepProps) {
    return (<div className={cn("min-w-0 max-w-full space-y-6", className)}>
      <div className="space-y-2">
        <Label>{t("form.cronExpression")} *</Label>
        <Input name="cronExpression" autoComplete="off" placeholder={t("form.cronPlaceholder")} value={cronExpression} onChange={(event) => setCronExpression(event.target.value)} className="font-mono" disabled={disabled}/>
        <p className="text-muted-foreground text-xs">{t("form.cronFormat")}</p>
      </div>
      <div className="space-y-2">
        <Label className="text-muted-foreground">{t("form.quickSelect")}</Label>
        <div className="flex min-w-0 flex-wrap gap-2">
          {cronPresets.map((preset) => (<Badge key={preset.value} variant={cronExpression === preset.value ? "default" : "outline"} className="cursor-pointer" render={<button type="button" onClick={() => setCronExpression(preset.value)} disabled={disabled}/>}>
                {preset.label}
              </Badge>))}
        </div>
      </div>
      <div className="bg-muted/50 border min-w-0 overflow-hidden p-4 rounded-lg space-y-3">
        <div className="flex min-w-0 gap-2 items-center">
          <IconClock className="h-4 shrink-0 text-muted-foreground w-4"/>
          <span className="min-w-0 flex-1 truncate font-medium">{t("form.executionPreview")}</span>
          {isCronExpressionValid(cronExpression) && (<Badge variant="secondary" className="ml-auto shrink-0"><IconCheck className="h-3 mr-1 w-3"/>{t("form.valid")}</Badge>)}
        </div>
        <p className="text-sm break-words">{getCronDescription(cronExpression)}</p>
        <Separator />
        <div className="min-w-0 space-y-1">
          <p className="text-muted-foreground text-xs">{t("form.nextExecutionTime")}</p>
          {getNextExecutions(cronExpression).map((time, index) => (<p key={index} className="text-sm break-words">• {time}{index === 0 && <span className="ml-2 text-muted-foreground">{t("form.upcoming")}</span>}</p>))}
        </div>
      </div>
    </div>);
}
interface ScheduledScanFooterProps {
    t: TranslationFn;
    currentStep: number;
    totalSteps: number;
    isPending: boolean;
    onPrev: () => void;
    onNext: () => void;
    onSubmit: () => void;
}
export function ScheduledScanFooter({ t, currentStep, totalSteps, isPending, onPrev, onNext, onSubmit, }: ScheduledScanFooterProps) {
    return (<div className="border-t flex justify-between px-6 py-4">
      {currentStep > 1 ? (<Button variant="outline" onClick={onPrev}>
          <IconChevronLeft className="h-4 mr-1 w-4"/>{t("buttons.previous")}
        </Button>) : <div />}
      {currentStep < totalSteps ? (<Button onClick={onNext}>{t("buttons.next")}<IconChevronRight className="h-4 ml-1 w-4"/></Button>) : (<Button onClick={onSubmit} disabled={isPending} loading={isPending} loadingLabel={t("buttons.createTask")}>
          {t("buttons.createTask")}
        </Button>)}
    </div>);
}
interface ScheduledScanOverwriteDialogProps {
    t: TranslationFn;
    open: boolean;
    onOpenChange: (value: boolean) => void;
    onCancel: () => void;
    onConfirm: () => void;
}
export function ScheduledScanOverwriteDialog({ t, open, onOpenChange, onCancel, onConfirm, }: ScheduledScanOverwriteDialogProps) {
    return (<AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t("overwriteConfirm.title")}</AlertDialogTitle>
          <AlertDialogDescription>
            {t("overwriteConfirm.description")}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogClose onClick={onCancel} variant="outline">
            {t("overwriteConfirm.cancel")}
          </AlertDialogClose>
          <AlertDialogClose onClick={onConfirm}>
            {t("overwriteConfirm.confirm")}
          </AlertDialogClose>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>);
}
