"use client";
import React, { useCallback, useMemo } from "react";
import { useTranslations } from "next-intl";
import { Check, ChevronDown, ChevronRight, Cpu, RefreshCw, } from "@/components/icons";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { FieldError } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { NumberStepperInput } from "@/components/ui/number-stepper-input";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Switch } from "@/components/ui/switch";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { Collapsible, CollapsibleContent, CollapsibleTrigger, } from "@/components/ui/collapsible";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue, } from "@/components/ui/select";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger, } from "@/components/ui/tooltip";
import type { CompleteWordlistCatalogState } from "@/hooks/use-wordlists";
import {
    configResourceFieldId,
    configResourceFieldKey,
    type ConfigResourceFieldError,
    type ConfigResourceFieldLocation,
} from "@/lib/engine-config-resource-validation";
import {
    parseWorkflowConfigurationDraftStrict,
    validateCompleteProfileEngineConfig,
    WorkflowConfigurationDraftError,
} from "@/lib/workflow-config";
import { cn } from "@/lib/utils";
import { textRole } from "@/lib/typography";
import { toastFeedback } from "@/lib/toast-helpers";
import type { EngineConfigFormValues, EngineConfigSectionDefinition, EngineConfigSectionFormValue, EngineConfigStepFormValue, EngineParamDefinition, EngineParamValue, ScanWorkflowWithEngines, WorkflowStageWithEngines, WorkflowStepWithEngine, } from "@/types/engine-config.types";
// ---------------------------------------------------------------------------
// Helpers: initialize / read / write form values
// ---------------------------------------------------------------------------
function defaultParamsForSection(section: EngineConfigSectionDefinition): Record<string, EngineParamValue> {
    const params: Record<string, EngineParamValue> = {};
    for (const param of section.params) {
        if (param.default !== undefined) {
            params[param.key] = Array.isArray(param.default) ? [...param.default] : param.default;
        }
        else if (param.type === "integer") {
            params[param.key] = 0;
        }
        else if (param.type === "boolean") {
            params[param.key] = false;
        }
        else if (param.type === "stringArray") {
            params[param.key] = [];
        }
        else {
            params[param.key] = "";
        }
    }
    return params;
}

function findSection(workflow: ScanWorkflowWithEngines, stepId: string, sectionId: string): EngineConfigSectionDefinition | null {
    for (const stage of workflow.stages) {
        const step = stage.steps.find((candidate) => candidate.stepId === stepId);
        const section = step?.engine.execution.configSections.find((candidate) => candidate.id === sectionId);
        if (section) return section;
    }
    return null;
}

export function initFormValuesFromWorkflow(
    workflow: ScanWorkflowWithEngines,
    configuration?: unknown,
    previousValues: EngineConfigFormValues = {},
): EngineConfigFormValues {
    if (configuration === undefined || configuration === null || (typeof configuration === "string" && configuration.trim() === "")) {
        return {};
    }
    const parsed = parseWorkflowConfigurationDraftStrict(configuration);
    const rawSteps = parsed.steps;
    if (!isRecord(rawSteps)) {
        throw new WorkflowConfigurationDraftError("configuration.steps", "Profile steps must be an object");
    }
    const values: EngineConfigFormValues = {};
    const expectedStepIds = new Set(workflow.stages.flatMap((stage) => stage.steps.map((step) => step.stepId)));
    for (const stepId of Object.keys(rawSteps)) {
        if (!expectedStepIds.has(stepId)) {
            throw new WorkflowConfigurationDraftError(`configuration.steps["${stepId}"]`, "unknown Workflow Step");
        }
    }
    for (const stage of workflow.stages) {
        for (const step of stage.steps) {
            const path = `configuration.steps["${step.stepId}"]`;
            const rawStep = rawSteps[step.stepId];
            if (!isRecord(rawStep) || typeof rawStep.enabled !== "boolean") {
                throw new WorkflowConfigurationDraftError(`${path}.enabled`, "enabled must be an explicit boolean");
            }
            const previous = previousValues[step.stepId];
            let engineConfig: Record<string, unknown>;
            if (isRecord(rawStep.engineConfig)) {
                engineConfig = rawStep.engineConfig;
                validateCompleteProfileEngineConfig(engineConfig, step, `${path}.engineConfig`);
            } else if (!rawStep.enabled && previous) {
                engineConfig = formSectionsToEngineConfig(previous.sections);
            } else {
                throw new WorkflowConfigurationDraftError(`${path}.engineConfig`, "complete Engine configuration is required");
            }

            const sections: Record<string, EngineConfigSectionFormValue> = {};
            for (const section of step.engine.execution.configSections) {
                const rawSection = engineConfig[section.id];
                if (!isRecord(rawSection) || typeof rawSection.enabled !== "boolean") {
                    throw new WorkflowConfigurationDraftError(`${path}.engineConfig.${section.id}`, "Engine config section is required");
                }
                const params: Record<string, EngineParamValue> = {};
                if (rawSection.enabled) {
                    for (const param of section.params) {
                        const value = rawSection[param.key];
                        if (!isValidEngineParameterValue(value, param.type)) {
                            throw new WorkflowConfigurationDraftError(`${path}.engineConfig.${section.id}.${param.key}`, "Engine config parameter is invalid");
                        }
                        params[param.key] = cloneEngineParamValue(value as EngineParamValue);
                    }
                }
                sections[section.id] = { enabled: rawSection.enabled, params };
            }
            values[step.stepId] = { enabled: rawStep.enabled, sections };
        }
    }
    return values;
}

function formSectionsToEngineConfig(sections: Record<string, EngineConfigSectionFormValue>): Record<string, unknown> {
    return Object.fromEntries(Object.entries(sections).map(([sectionId, section]) => [
        sectionId,
        section.enabled ? { enabled: true, ...section.params } : { enabled: false },
    ]));
}

function isRecord(value: unknown): value is Record<string, unknown> {
    return typeof value === "object" && value !== null && !Array.isArray(value);
}
export function serializeFormValuesToConfig(values: EngineConfigFormValues): Record<string, unknown> {
    const steps: Record<string, unknown> = {};
    for (const [stepId, stepValue] of Object.entries(values)) {
        if (!stepValue.enabled) {
            steps[stepId] = { enabled: false };
            continue;
        }
        const engineConfig: Record<string, unknown> = {};
        for (const [sectionId, sectionData] of Object.entries(stepValue.sections)) {
            if (!sectionData.enabled) {
                engineConfig[sectionId] = { enabled: false };
                continue;
            }
            const sectionConfig: Record<string, unknown> = { enabled: true };
            Object.assign(sectionConfig, sectionData.params);
            engineConfig[sectionId] = sectionConfig;
        }
        steps[stepId] = { enabled: true, engineConfig };
    }
    return { steps };
}
// ---------------------------------------------------------------------------
// Engine icon resolver
// ---------------------------------------------------------------------------
function getEngineIcon(engineId: string): React.ReactNode {
    void engineId;
    return <Cpu className="size-5"/>;
}
function getIntegerStep(param: EngineParamDefinition) {
    if (param.key === "timeout")
        return 60;
    if (param.key === "rate")
        return 100;
    return 1;
}

const PARAM_HELP_HOVER_OPEN_DELAY_MS = 200;

function WordlistResourceSelect({ fieldId, value, disabled, catalog, invalid, describedBy, onChange, }: {
    fieldId: string;
    value: EngineParamValue;
    disabled: boolean;
    catalog: CompleteWordlistCatalogState;
    invalid: boolean;
    describedBy?: string;
    onChange: (value: string) => void;
}) {
    const tScanInitiate = useTranslations("scan.initiate");
    const options = catalog.wordlists;
    const fileNameByResourceName = useMemo(
        () => new Map(options.map((wordlist) => [wordlist.name, wordlist.fileName])),
        [options]
    );
    const currentValue = value === "" ? "" : String(value);
    const catalogErrorId = `${fieldId}-catalog-error`;
    const ariaDescribedBy = [describedBy, catalog.status === "failed" ? catalogErrorId : undefined].filter(Boolean).join(" ") || undefined;
    const isComplete = catalog.status === "complete";
    const placeholder = isComplete
        ? tScanInitiate("engineConfigForm.selectWordlist")
        : catalog.status === "failed"
            ? tScanInitiate("engineConfigForm.wordlistLoadFailed")
            : tScanInitiate("engineConfigForm.loading");
    return (<div className="space-y-1.5">
      <Select value={currentValue} onValueChange={onChange} disabled={disabled || !isComplete} itemToStringLabel={(resourceName) => {
        if (typeof resourceName !== "string") {
          return "";
        }
        return fileNameByResourceName.get(resourceName) ?? resourceName;
      }}>
      <SelectTrigger id={fieldId} size="sm" className="w-full" aria-invalid={invalid || undefined} aria-describedby={ariaDescribedBy}>
        <SelectValue placeholder={placeholder}/>
      </SelectTrigger>
      <SelectContent>
        {currentValue && !isComplete && !options.some((wordlist) => wordlist.name === currentValue) ? (<SelectItem value={currentValue}>{currentValue}</SelectItem>) : null}
        {options.map((wordlist) => (<SelectItem key={wordlist.name} value={wordlist.name}>
            {wordlist.fileName}
          </SelectItem>))}
      </SelectContent>
      </Select>
      {catalog.status === "failed" ? (<div className="flex min-w-0 items-center justify-between gap-2">
        <FieldError id={catalogErrorId}>{tScanInitiate("engineConfigForm.wordlistLoadFailed")}</FieldError>
        <Button type="button" variant="ghost" size="sm" onClick={catalog.retry} disabled={disabled}>
          <RefreshCw className="size-3.5"/>
          {tScanInitiate("engineConfigForm.retryWordlists")}
        </Button>
      </div>) : null}
    </div>);
}
// ---------------------------------------------------------------------------
// Param field renderer — matches mockup: label + input + help text
// ---------------------------------------------------------------------------
interface ParamFieldProps {
    location: ConfigResourceFieldLocation;
    param: EngineParamDefinition;
    value: EngineParamValue;
    disabled: boolean;
    wordlistCatalog: CompleteWordlistCatalogState;
    error?: ConfigResourceFieldError;
    onChange: (value: EngineParamValue) => void;
    onFieldRepaired?: (location: ConfigResourceFieldLocation) => void;
}

function ParamHelpText({ description }: { description: string }) {
    return (<TooltipProvider delay={PARAM_HELP_HOVER_OPEN_DELAY_MS}>
      <Tooltip>
        <TooltipTrigger render={<p className={cn("line-clamp-2 cursor-help", textRole.helperText)} tabIndex={0}/>}>{description}</TooltipTrigger>
        {/* Center the overlay on its full-width trigger so the arrow remains centered. */}
        <TooltipContent side="top" align="center" sideOffset={6} className="max-w-sm whitespace-normal text-left">
          {description}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>);
}

function EnumStringArrayPopover({ fieldId, labelId, param, value, disabled, onChange, }: {
    fieldId: string;
    labelId: string;
    param: EngineParamDefinition;
    value: EngineParamValue;
    disabled: boolean;
    onChange: (value: EngineParamValue) => void;
}) {
    const currentValues = Array.isArray(value) ? value : [];
    const options = param.enum ?? [];
    const selectedSummary = currentValues.length > 0 ? currentValues.join(", ") : "-";

    const updateSelection = (option: string, checked: boolean) => {
        const selected = currentValues.includes(option);
        if (!checked && selected && param.minItems !== undefined && currentValues.length <= param.minItems) return;
        if (checked && !selected && param.maxItems !== undefined && currentValues.length >= param.maxItems) return;

        const next = checked ? [...currentValues, option] : currentValues.filter((item) => item !== option);
        const order = new Map(options.map((item, index) => [item, index]));
        next.sort((left, right) => (order.get(left) ?? 0) - (order.get(right) ?? 0));
        onChange(next);
    };

    return (<Popover>
      <PopoverTrigger render={<Button id={fieldId} type="button" variant="outline" size="sm" layout="between" className="text-left font-normal" disabled={disabled} aria-labelledby={labelId}/> }>
        <span className="min-w-0 truncate">{selectedSummary}</span>
        <ChevronDown aria-hidden className="size-4 shrink-0 text-muted-foreground"/>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-[var(--anchor-width)] p-1">
        <div role="group" aria-labelledby={labelId} className="grid gap-1">
          {options.map((option) => {
              const selected = currentValues.includes(option);
              return (<Label key={option} htmlFor={`${fieldId}-${option}`} className="flex min-w-0 items-center gap-2 rounded-sm px-2 py-1.5 font-normal hover:bg-accent">
                <Checkbox id={`${fieldId}-${option}`} checked={selected} disabled={disabled} onCheckedChange={(checked) => updateSelection(option, checked === true)} />
                <span className="min-w-0 truncate">{option}</span>
              </Label>);
          })}
        </div>
      </PopoverContent>
    </Popover>);
}

function ParamField({ location, param, value, disabled, wordlistCatalog, error, onChange, onFieldRepaired }: ParamFieldProps) {
    const tScanInitiate = useTranslations("scan.initiate");
    const fieldId = configResourceFieldId(location);
    const errorMessageId = `${fieldId}-error`;
    const unitHint = useMemo(() => {
        if (param.type !== "integer")
            return null;
        if (param.key === "timeout")
            return `(${tScanInitiate("engineConfigForm.unitSeconds")})`;
        if (param.description?.includes("threads"))
            return null;
        return null;
    }, [param, tScanInitiate]);
    const constraintHint = useMemo(() => {
        if (param.type !== "integer") {
            return null;
        }
        if (param.minimum !== undefined && param.maximum !== undefined) {
            return tScanInitiate("engineConfigForm.range", { min: param.minimum, max: param.maximum });
        }
        if (param.minimum !== undefined) {
            return tScanInitiate("engineConfigForm.minimum", { value: param.minimum });
        }
        if (param.maximum !== undefined) {
            return tScanInitiate("engineConfigForm.maximum", { value: param.maximum });
        }
        return null;
    }, [param, tScanInitiate]);
    if (param.type === "boolean") {
        const labelId = `${fieldId}-label`;
        return (<div className="space-y-1">
        <p id={labelId} className={textRole.compactPrimary}>{param.key}</p>
        <ToggleGroup type="single" value={Boolean(value) ? "on" : "off"} onValueChange={(nextValue) => {
            if (nextValue === "on") onChange(true);
            if (nextValue === "off") onChange(false);
        }} aria-labelledby={labelId} className="radius-control relative grid h-8 w-32 grid-cols-2 overflow-hidden border border-input bg-muted/20 p-0">
          <span aria-hidden className={cn("pointer-events-none absolute inset-y-0 left-0 z-0 w-1/2 bg-primary transition-transform duration-200 ease-out", Boolean(value) ? "translate-x-full" : "translate-x-0")}/>
          <ToggleGroupItem value="off" size="sm" className="radius-none relative z-10 h-full min-w-0 px-0 text-muted-foreground transition-colors hover:bg-transparent hover:text-muted-foreground data-pressed:bg-transparent data-pressed:text-primary-foreground data-pressed:hover:bg-transparent data-pressed:hover:text-primary-foreground" disabled={disabled}>
            {tScanInitiate("engineConfigForm.off")}
            {!Boolean(value) ? <Check aria-hidden className="absolute right-2 size-3"/> : null}
          </ToggleGroupItem>
          <ToggleGroupItem value="on" size="sm" className="radius-none relative z-10 h-full min-w-0 px-0 text-muted-foreground transition-colors hover:bg-transparent hover:text-muted-foreground data-pressed:bg-transparent data-pressed:text-primary-foreground data-pressed:hover:bg-transparent data-pressed:hover:text-primary-foreground" disabled={disabled}>
            {tScanInitiate("engineConfigForm.on")}
            {Boolean(value) ? <Check aria-hidden className="absolute right-2 size-3"/> : null}
          </ToggleGroupItem>
        </ToggleGroup>
        {param.description ? <ParamHelpText description={param.description}/> : null}
      </div>);
    }
    return (<div className="space-y-1">
      <div className="flex items-center justify-between gap-2">
        <Label id={`${fieldId}-label`} htmlFor={param.type === "stringArray" && param.enum?.length ? undefined : fieldId} className={cn("flex min-w-0 items-center gap-1 truncate", textRole.compactPrimary)}>
          {param.key}
          {unitHint ? (<span className={cn(textRole.helperText, "opacity-60")}>{unitHint}</span>) : null}
        </Label>
        {constraintHint ? (<span className={cn("shrink-0", textRole.helperText)}>{constraintHint}</span>) : null}
      </div>
      {param.resource?.kind === "wordlist" ? (<WordlistResourceSelect fieldId={fieldId} value={value} disabled={disabled} catalog={wordlistCatalog} invalid={Boolean(error)} describedBy={error ? errorMessageId : undefined} onChange={(next) => {
          onChange(next);
          if (next.trim() !== "") onFieldRepaired?.(location);
      }}/>) : param.type === "string" && param.enum && param.enum.length > 0 ? (<Select value={String(value)} onValueChange={onChange} disabled={disabled}>
          <SelectTrigger id={fieldId} size="sm" className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {param.enum.map((option) => (<SelectItem key={String(option)} value={String(option)}>
                {String(option)}
              </SelectItem>))}
          </SelectContent>
        </Select>) : param.type === "integer" ? (<NumberStepperInput id={fieldId} value={typeof value === "number" ? value : Number(value) || 0} min={param.minimum} max={param.maximum} step={getIntegerStep(param)} disabled={disabled} onChange={onChange}/>) : param.type === "stringArray" && param.enum && param.enum.length > 0 && (param.key === "scan-targets" || param.key === "severity") ? (<EnumStringArrayPopover fieldId={fieldId} labelId={`${fieldId}-label`} param={param} value={value} disabled={disabled} onChange={onChange}/>) : param.type === "stringArray" && param.enum && param.enum.length > 0 ? (<div id={fieldId} role="group" aria-labelledby={`${fieldId}-label`} className="grid gap-2 rounded-md border border-input bg-muted/10 p-2 sm:grid-cols-2">
          {param.enum.map((option) => {
              const selected = Array.isArray(value) && value.includes(option);
              return (<Label key={option} htmlFor={`${fieldId}-${option}`} className="flex min-w-0 items-center gap-2 font-normal">
                <Checkbox id={`${fieldId}-${option}`} checked={selected} disabled={disabled} onCheckedChange={(checked) => {
                    const current = Array.isArray(value) ? value : [];
                    if (!checked && selected && param.minItems !== undefined && current.length <= param.minItems) return;
                    if (checked && !selected && param.maxItems !== undefined && current.length >= param.maxItems) return;
                    const next = checked ? [...current, option] : current.filter((item) => item !== option);
                    const order = new Map(param.enum!.map((item, index) => [item, index]));
                    next.sort((left, right) => (order.get(left) ?? 0) - (order.get(right) ?? 0));
                    onChange(next);
                }} />
                <span className="truncate">{option}</span>
              </Label>);
          })}
        </div>) : param.type === "stringArray" ? (<Input id={fieldId} type="text" value={Array.isArray(value) ? value.join(",") : ""} disabled={disabled} onChange={(e) => {
                onChange(e.target.value.split(",").map((item) => item.trim()).filter(Boolean));
            }} variant="subtle" size="sm"/>) : (<Input id={fieldId} type="text" value={value === "" ? "" : String(value)} min={param.minimum} max={param.maximum} pattern={param.pattern} disabled={disabled} onChange={(e) => {
                onChange(e.target.value);
            }} variant="subtle" size="sm"/>)}
      {error ? <FieldError id={errorMessageId}>{tScanInitiate("engineConfigForm.resourceRequired")}</FieldError> : null}
      {param.description ? <ParamHelpText description={param.description}/> : null}
    </div>);
}
// ---------------------------------------------------------------------------
// Section content — parameter fields (no collapsible)
// ---------------------------------------------------------------------------
interface SectionContentProps {
    stepId: string;
    section: EngineConfigSectionDefinition;
    sectionData: {
        enabled: boolean;
        params: Record<string, EngineParamValue>;
    };
    disabled: boolean;
    wordlistCatalog: CompleteWordlistCatalogState;
    fieldErrors: ReadonlyMap<string, ConfigResourceFieldError>;
    onParamChange: (paramKey: string, value: EngineParamValue) => void;
    onFieldRepaired?: (location: ConfigResourceFieldLocation) => void;
    isLast: boolean;
}
function SectionContent({ stepId, section, sectionData, disabled, wordlistCatalog, fieldErrors, onParamChange, onFieldRepaired, isLast, }: SectionContentProps) {
    return (<div className={cn("px-4 py-3", !sectionData.enabled && "opacity-65")}>
      <div className="grid gap-3 sm:grid-cols-2">
        {section.params.map((param) => {
          const value = sectionData.params[param.key];
          if (value === undefined) return null;
          const location = { stepId, sectionId: section.id, paramKey: param.key };
          return <ParamField key={param.key} location={location} param={param} value={value} disabled={disabled || !sectionData.enabled} wordlistCatalog={wordlistCatalog} error={fieldErrors.get(configResourceFieldKey(location))} onChange={(nextValue) => onParamChange(param.key, nextValue)} onFieldRepaired={onFieldRepaired} />;
        })}
      </div>

      {/* Divider between sections */}
      {!isLast ? <div className="mt-4 h-px bg-border/60"/> : null}
    </div>);
}
function SectionHeader({ section, sectionData, disabled, onToggleSection, }: {
    section: EngineConfigSectionDefinition;
    sectionData: EngineConfigSectionFormValue;
    disabled: boolean;
    onToggleSection: (enabled: boolean) => void;
}) {
    const enabled = sectionData.enabled !== false;
    return (<div className="flex items-start justify-between gap-3 px-4 py-3">
      <div className="min-w-0 space-y-1">
        <p className={cn("truncate", textRole.navLabel)}>{section.name}</p>
        {section.description ? (<p className={cn("line-clamp-2", textRole.helperText)} title={section.description}>{section.description}</p>) : null}
      </div>
      <div className="flex shrink-0 items-center gap-2 pt-0.5">
        <span className={cn(textRole.helperText, enabled ? "text-primary" : "text-muted-foreground/50")}>
          {enabled ? "ON" : "OFF"}
        </span>
        <Switch checked={enabled} onCheckedChange={onToggleSection} disabled={disabled || section.requiredEnabled} aria-label={section.name}/>
      </div>
    </div>);
}
// ---------------------------------------------------------------------------
// Engine card — collapsible, contains all config sections
// ---------------------------------------------------------------------------
interface EngineCardProps {
    step: WorkflowStepWithEngine;
    stepValues: EngineConfigStepFormValue;
    disabled: boolean;
    open: boolean;
    wordlistCatalog: CompleteWordlistCatalogState;
    fieldErrors: ReadonlyMap<string, ConfigResourceFieldError>;
    onOpenChange: (open: boolean) => void;
    onToggleStep: (enabled: boolean) => void;
    onToggleSection: (sectionId: string, enabled: boolean) => void;
    onParamChange: (sectionId: string, paramKey: string, value: EngineParamValue) => void;
    onFieldRepaired?: (location: ConfigResourceFieldLocation) => void;
}
function EngineCard({ step, stepValues, disabled, open, wordlistCatalog, fieldErrors, onOpenChange, onToggleStep, onToggleSection, onParamChange, onFieldRepaired, }: EngineCardProps) {
    const tScanInitiate = useTranslations("scan.initiate");
    const configSections = step.engine.execution.configSections;
    return (<div className="overflow-hidden border-t bg-card first:border-t-0">
      <Collapsible open={open} onOpenChange={onOpenChange}>
        <div className="flex items-center gap-3 px-4 py-3">
          <span className={cn("flex size-5 shrink-0 items-center justify-center", stepValues.enabled ? "text-interaction-accent" : "text-muted-foreground")}>
            {getEngineIcon(step.engineId)}
          </span>
          <div className="min-w-0 flex-1">
            <p className={cn("truncate", textRole.navLabel)}>{step.engine.displayName}</p>
            {step.engine.description ? (
              <p className={cn("truncate", textRole.helperText)} title={step.engine.description}>
                {step.engine.description}
              </p>
            ) : null}
          </div>
          <Switch checked={stepValues.enabled} disabled={disabled} onCheckedChange={onToggleStep} aria-label={step.engine.displayName}/>
          <CollapsibleTrigger render={<Button type="button" variant="ghost" size="icon-sm" aria-label={open ? tScanInitiate("engineConfigForm.collapse") : tScanInitiate("engineConfigForm.expand")}/> }>
              {open ? (<ChevronDown className="size-4"/>) : (<ChevronRight className="size-4"/>)}
            </CollapsibleTrigger>
        </div>

        <CollapsibleContent>
          <div className="border-t bg-muted/10">
            {configSections.map((section, index) => {
            const sectionData = stepValues.sections[section.id];
            if (!sectionData) return null;
            return (<div key={section.id} className="border-b last:border-b-0">
                  <SectionHeader section={section} sectionData={sectionData} disabled={disabled || !stepValues.enabled} onToggleSection={(enabled) => onToggleSection(section.id, enabled)}/>
                  {sectionData.enabled ? (<SectionContent stepId={step.stepId} section={section} sectionData={sectionData} disabled={disabled || !stepValues.enabled} wordlistCatalog={wordlistCatalog} fieldErrors={fieldErrors} onParamChange={(paramKey, value) => onParamChange(section.id, paramKey, value)} onFieldRepaired={onFieldRepaired} isLast={index === configSections.length - 1}/>) : null}
                </div>);
        })}
          </div>
        </CollapsibleContent>
      </Collapsible>
    </div>);
}
// ---------------------------------------------------------------------------
// Stage block — dot + label header, then engine cards
// ---------------------------------------------------------------------------
interface StageBlockProps {
    stage: WorkflowStageWithEngines;
    stageIndex: number;
    formValues: EngineConfigFormValues;
    disabled: boolean;
    expandedStepIds: ReadonlySet<string>;
    wordlistCatalog: CompleteWordlistCatalogState;
    fieldErrors: ReadonlyMap<string, ConfigResourceFieldError>;
    onExpandedStepChange: (stepId: string, open: boolean) => void;
    onToggleStep: (stepId: string, enabled: boolean) => void;
    onToggleSection: (stepId: string, sectionId: string, enabled: boolean) => void;
    onParamChange: (stepId: string, sectionId: string, paramKey: string, value: EngineParamValue) => void;
    onFieldRepaired?: (location: ConfigResourceFieldLocation) => void;
}
function StageBlock({ stage, stageIndex, formValues, disabled, expandedStepIds, wordlistCatalog, fieldErrors, onExpandedStepChange, onToggleStep, onToggleSection, onParamChange, onFieldRepaired, }: StageBlockProps) {
    const tScanInitiate = useTranslations("scan.initiate");
    const enabledEngineCount = stage.steps.filter((step) => {
        const stepValue = formValues[step.stepId];
        if (!stepValue)
            return false;
        return Boolean(stepValue?.enabled) && step.engine.execution.configSections.some((section) => stepValue.sections[section.id]?.enabled !== false);
    }).length;
    return (<div className="radius-surface overflow-hidden border border-sidebar-border bg-card">
      <div className="flex items-center justify-between gap-3 bg-muted px-4 py-3">
        <div className="flex min-w-0 items-center gap-2">
          <span className="size-2 rounded-full bg-primary"/>
          <span className={cn("truncate", textRole.navLabel)}>
            {tScanInitiate("engineConfigForm.stageLabel", { index: stageIndex + 1, stageId: stage.stageId })}
          </span>
        </div>
        <Badge variant="outline">
          {tScanInitiate("engineConfigForm.enabledSummary", { enabled: enabledEngineCount, total: stage.steps.length })}
        </Badge>
      </div>

      {stage.steps.map((step) => {
        const stepValues = formValues[step.stepId];
        return <EngineCard key={step.stepId} step={step} stepValues={stepValues ?? { enabled: false, sections: {} }} disabled={disabled || !stepValues} open={expandedStepIds.has(step.stepId)} wordlistCatalog={wordlistCatalog} fieldErrors={fieldErrors} onOpenChange={(open) => onExpandedStepChange(step.stepId, open)} onToggleStep={(enabled) => onToggleStep(step.stepId, enabled)} onToggleSection={(sectionId, enabled) => onToggleSection(step.stepId, sectionId, enabled)} onParamChange={(sectionId, paramKey, value) => onParamChange(step.stepId, sectionId, paramKey, value)} onFieldRepaired={onFieldRepaired}/>;
      })}
    </div>);
}
// ---------------------------------------------------------------------------
// Main form component
// ---------------------------------------------------------------------------
interface EngineConfigFormProps {
    workflow: ScanWorkflowWithEngines;
    values: EngineConfigFormValues;
    disabled?: boolean;
    wordlistCatalog?: CompleteWordlistCatalogState;
    fieldErrors?: ReadonlyMap<string, ConfigResourceFieldError>;
    expandedStepIds?: ReadonlySet<string>;
    onExpandedStepChange?: (stepId: string, open: boolean) => void;
    onFieldRepaired?: (location: ConfigResourceFieldLocation) => void;
    onChange: (values: EngineConfigFormValues) => void;
}
const EMPTY_WORDLIST_CATALOG: CompleteWordlistCatalogState = { status: "complete", wordlists: [], retry: () => undefined };
const EMPTY_FIELD_ERRORS = new Map<string, ConfigResourceFieldError>();

export function EngineConfigForm({ workflow, values, disabled = false, wordlistCatalog = EMPTY_WORDLIST_CATALOG, fieldErrors = EMPTY_FIELD_ERRORS, expandedStepIds, onExpandedStepChange, onFieldRepaired, onChange, }: EngineConfigFormProps) {
    const tScanInitiate = useTranslations("scan.initiate");
    const [internalExpandedStepIds, setInternalExpandedStepIds] = React.useState<Set<string>>(() => new Set());
    const resolvedExpandedStepIds = expandedStepIds ?? internalExpandedStepIds;
    const handleExpandedStepChange = useCallback((stepId: string, open: boolean) => {
        onExpandedStepChange?.(stepId, open);
        if (expandedStepIds) return;
        setInternalExpandedStepIds((current) => {
            const next = new Set(current);
            if (open) next.add(stepId);
            else next.delete(stepId);
            return next;
        });
    }, [expandedStepIds, onExpandedStepChange]);
    const handleToggleStep = useCallback((stepId: string, enabled: boolean) => {
        onChange({
            ...values,
            [stepId]: {
                ...values[stepId],
                enabled,
            },
        });
    }, [values, onChange]);
    const handleToggleSection = useCallback((stepId: string, sectionId: string, enabled: boolean) => {
        const stepValue = values[stepId];
        if (!stepValue?.enabled) return;
        const currentSection = stepValue.sections[sectionId];
        if (!currentSection) return;
		const sectionDefinition = findSection(workflow, stepId, sectionId);
		if (!enabled && sectionDefinition?.requiredEnabled) return;
        if (!enabled && Object.values(stepValue.sections).filter((section) => section.enabled).length <= 1) {
            toastFeedback.warning(tScanInitiate("engineConfigForm.lastSectionHint"));
            return;
        }
        onChange({
            ...values,
            [stepId]: {
                ...stepValue,
                sections: {
                    ...stepValue.sections,
                    [sectionId]: {
                        ...currentSection,
                        enabled,
                        params: enabled && sectionDefinition ? defaultParamsForSection(sectionDefinition) : {},
                    },
                },
            },
        });
    }, [tScanInitiate, values, workflow, onChange]);
    const handleParamChange = useCallback((stepId: string, sectionId: string, paramKey: string, value: EngineParamValue) => {
        const stepVal = values[stepId];
        if (!stepVal?.enabled || !stepVal.sections[sectionId]?.enabled) return;
        const sectionVal = stepVal.sections[sectionId];
        onChange({
            ...values,
            [stepId]: {
                ...stepVal,
                sections: {
                    ...stepVal.sections,
                    [sectionId]: {
                        ...sectionVal,
                        params: {
                            ...sectionVal.params,
                            [paramKey]: value,
                        },
                    },
                },
            },
        });
    }, [values, onChange]);
    return (<div className="space-y-3">
      {workflow.stages.map((stage, index) => (<StageBlock key={stage.stageId} stage={stage} stageIndex={index} formValues={values} disabled={disabled} expandedStepIds={resolvedExpandedStepIds} wordlistCatalog={wordlistCatalog} fieldErrors={fieldErrors} onExpandedStepChange={handleExpandedStepChange} onToggleStep={handleToggleStep} onToggleSection={handleToggleSection} onParamChange={handleParamChange} onFieldRepaired={onFieldRepaired}/>))}
    </div>);
}

function isValidEngineParameterValue(value: unknown, type: string): boolean {
    if (type === "integer") return typeof value === "number" && Number.isInteger(value);
    if (type === "boolean") return typeof value === "boolean";
    if (type === "string") return typeof value === "string";
    if (type === "stringArray") return Array.isArray(value) && value.every((item) => typeof item === "string");
    return false;
}

function cloneEngineParamValue(value: EngineParamValue): EngineParamValue {
    return Array.isArray(value) ? [...value] : value;
}
