"use client";
import { Check, Filter, Plus, XIcon } from "@/components/icons";
import { useTranslations } from "next-intl";
import { useEffect, useState, type ReactNode } from "react";
import { Button } from "@/components/ui/button";
import { Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList, CommandSeparator, } from "@/components/ui/command";
import { Popover, PopoverContent, PopoverTrigger, } from "@/components/ui/popover";
import { Separator } from "@/components/ui/separator";
import { TabsCountBadge } from "@/components/ui/tabs";
import { textRole } from "@/lib/typography";
import { cn } from "@/lib/utils";
export type DataTableFacetedFilterOption<TValue extends string> = {
    value: TValue;
    label: string;
    count?: number;
};
export type DataTableFacetedFilterContentSize = "default" | "wide";
const facetedFilterContentSizeClassNames: Record<DataTableFacetedFilterContentSize, string> = {
    default: "w-48",
    wide: "w-56",
};
interface DataTableFacetedFilterProps<TValue extends string> {
    title: string;
    values: TValue[];
    onValuesChange: (values: TValue[]) => void;
    options: Array<DataTableFacetedFilterOption<TValue>>;
    emptyLabel: string;
    clearLabel: string;
    contentSize?: DataTableFacetedFilterContentSize;
    contentClassName?: string;
    searchPlaceholder?: string;
}
interface DataTableFacetedFilterGroupProps {
    children: ReactNode;
    hasSelectedValues: boolean;
    onReset: () => void;
    className?: string;
}
export type DataTableFacetPanelFacet = {
  id: string;
  label: string;
  values: string[];
  onValuesChange: (values: string[]) => void;
  options: Array<DataTableFacetedFilterOption<string>>;
  emptyLabel: string;
  clearLabel: string;
  searchPlaceholder?: string;
};
interface DataTableFacetPanelProps {
  title: string;
  facets: DataTableFacetPanelFacet[];
  activeCount: number;
  contentClassName?: string;
}
function toggleFacetedFilterValue<TValue extends string>(values: TValue[], value: TValue): TValue[] {
    return values.includes(value)
        ? values.filter((item) => item !== value)
        : [...values, value];
}
function createFacetDraft(facets: DataTableFacetPanelFacet[]): Record<string, string[]> {
  return Object.fromEntries(facets.map((facet) => [facet.id, [...facet.values]]));
}
function haveSameFacetValues(left: string[], right: string[]): boolean {
  return left.length === right.length && left.every((value, index) => value === right[index]);
}
function DataTableFacetedFilterOptions<TValue extends string>({
  title,
  values,
  onValuesChange,
  options,
  emptyLabel,
  clearLabel,
  searchPlaceholder,
  showClear = true,
}: DataTableFacetedFilterProps<TValue> & { showClear?: boolean }) {
    const tDataTable = useTranslations("dataTable");
    const selectedValues = new Set(values);
    const selectedOptions = options.filter((option) => selectedValues.has(option.value));
    const onClearFilters = () => onValuesChange([]);
    return (<Command>
      <CommandInput placeholder={searchPlaceholder ?? tDataTable("searchFilter", { title })}/>
      <CommandList className="max-h-64">
        <CommandEmpty>{emptyLabel}</CommandEmpty>
        <CommandGroup>
          {options.map((option) => {
        const isSelected = selectedValues.has(option.value);
        return (<CommandItem key={option.value} value={option.value} keywords={[option.label]} onSelect={() => onValuesChange(toggleFacetedFilterValue(values, option.value))} className="gap-2">
                <div className={cn("radius-control-subtle flex size-4 items-center justify-center border", isSelected
                ? "border-interaction-accent bg-interaction-accent text-interaction-accent-foreground"
                : "border-input [&_svg]:invisible")}>
                  <Check className="size-3"/>
                </div>
                <span className="min-w-0 truncate">{option.label}</span>
                {typeof option.count === "number" ? (<span className={cn("ml-auto shrink-0 tabular-nums text-muted-foreground", textRole.metadataLabel)}>
                    {option.count.toLocaleString()}
                  </span>) : null}
              </CommandItem>);
    })}
        </CommandGroup>
      </CommandList>
      {showClear && selectedOptions.length > 0 ? (<div className="shrink-0">
          <CommandSeparator />
          <CommandGroup>
            <CommandItem onSelect={onClearFilters} className="justify-center text-center">
              {clearLabel}
            </CommandItem>
          </CommandGroup>
        </div>) : null}
    </Command>);
}
export function DataTableFacetedFilter<TValue extends string>({ title, values, onValuesChange, options, emptyLabel, clearLabel, contentSize = "default", contentClassName, searchPlaceholder, }: DataTableFacetedFilterProps<TValue>) {
    const tActions = useTranslations("common.actions");
    const tDataTable = useTranslations("dataTable");
    const [open, setOpen] = useState(false);
    const [draftValues, setDraftValues] = useState<TValue[]>(values);
    const selectedValues = new Set(values);
    const selectedOptions = options.filter((option) => selectedValues.has(option.value));
    const hasDraftChanges = !haveSameFacetValues(values, draftValues);
    const handleOpenChange = (nextOpen: boolean) => {
      setOpen(nextOpen);
      if (nextOpen) {
        setDraftValues(values);
      }
    };
    const resetDraft = () => setDraftValues([]);
    const applyDraft = () => {
      if (hasDraftChanges) {
        onValuesChange(draftValues);
      }
      setOpen(false);
    };

    return (<Popover open={open} onOpenChange={handleOpenChange}>
        <PopoverTrigger render={<Button variant="outline" size="sm" className="border-dashed"/>}>
            <span className="radius-round flex size-4 items-center justify-center border border-foreground/80">
              <Plus className="size-3"/>
            </span>
            {title}
            {selectedOptions.length > 0 ? (<>
                  <Separator orientation="vertical" className="mx-2 h-4"/>
                  <TabsCountBadge className="font-normal lg:hidden">
                    {selectedOptions.length}
                  </TabsCountBadge>
                  <span className="hidden gap-1 lg:flex">
                    {selectedOptions.length > 1 ? (<TabsCountBadge className="font-normal">
                        {selectedOptions.length}
                      </TabsCountBadge>) : (selectedOptions.map((option) => (<TabsCountBadge key={option.value} className="max-w-28 justify-start overflow-hidden font-normal">
                          <span className="block min-w-0 truncate text-left">{option.label}</span>
                        </TabsCountBadge>)))}
                  </span>
              </>) : null}
          </PopoverTrigger>
        <PopoverContent minWidth="content" className={cn(facetedFilterContentSizeClassNames[contentSize], "overflow-hidden p-0", contentClassName)} align="start">
          <DataTableFacetedFilterOptions title={title} values={draftValues} onValuesChange={setDraftValues} options={options} emptyLabel={emptyLabel} clearLabel={clearLabel} searchPlaceholder={searchPlaceholder} showClear={false}/>
          <div className="grid grid-cols-2 gap-2 border-t p-2">
            <Button type="button" variant="outline" size="sm" layout="fullWidth" onClick={resetDraft}>
              {tActions("reset")}
            </Button>
            <Button type="button" size="sm" layout="fullWidth" disabled={!hasDraftChanges} onClick={applyDraft}>
              {tDataTable("applyFilter")}
            </Button>
          </div>
        </PopoverContent>
      </Popover>);
}
export function DataTableFacetedFilterGroup({ children, hasSelectedValues, onReset, className, }: DataTableFacetedFilterGroupProps) {
    const tActions = useTranslations("common.actions");
    return (<div className={cn("flex items-center gap-1", className)}>
      {children}
      {hasSelectedValues ? (<Button type="button" variant="ghost" size="sm" aria-label={tActions("reset")} onClick={onReset}>
          {tActions("reset")}
          <XIcon className="size-4"/>
        </Button>) : null}
    </div>);
}
export function DataTableFacetPanel({ title, facets, activeCount, contentClassName, }: DataTableFacetPanelProps) {
  const tActions = useTranslations("common.actions");
  const tDataTable = useTranslations("dataTable");
  const [open, setOpen] = useState(false);
  const [activeFacetId, setActiveFacetId] = useState(facets[0]?.id ?? "");
  const [draftValues, setDraftValues] = useState(() => createFacetDraft(facets));
  const draftFacets = facets.map((facet) => ({
    ...facet,
    values: draftValues[facet.id] ?? [],
    onValuesChange: (values: string[]) => setDraftValues((current) => ({ ...current, [facet.id]: values })),
  }));
  const isSingleFacet = draftFacets.length === 1;
  const activeFacet = draftFacets.find((facet) => facet.id === activeFacetId) ?? draftFacets[0];
  const hasDraftChanges = facets.some((facet) => !haveSameFacetValues(facet.values, draftValues[facet.id] ?? []));

  const handleOpenChange = (nextOpen: boolean) => {
    setOpen(nextOpen);
    if (nextOpen) {
      setDraftValues(createFacetDraft(facets));
    }
  };
  const resetDraft = () => setDraftValues(Object.fromEntries(facets.map((facet) => [facet.id, []])));
  const applyDraft = () => {
    for (const facet of facets) {
      const nextValues = draftValues[facet.id] ?? [];
      if (!haveSameFacetValues(facet.values, nextValues)) {
        facet.onValuesChange(nextValues);
      }
    }
    setOpen(false);
  };

  useEffect(() => {
    if (!facets.some((facet) => facet.id === activeFacetId)) {
      setActiveFacetId(facets[0]?.id ?? "");
    }
  }, [activeFacetId, facets]);

  return (<Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverTrigger render={<Button variant="outline" size="sm"/>}>
        <Filter className="size-4"/>
        {title}
        {activeCount > 0 ? (<TabsCountBadge className="font-normal">{activeCount}</TabsCountBadge>) : null}
      </PopoverTrigger>
      <PopoverContent minWidth="content" className={cn(isSingleFacet ? "w-72" : "w-80 sm:w-96", "overflow-hidden p-0", contentClassName)} align="start">
        <div className="flex h-80 overflow-hidden" data-facet-layout={isSingleFacet ? "single" : "multiple"}>
          {!isSingleFacet ? (<div className="flex h-full w-40 shrink-0 flex-col border-r">
            <div className={cn("border-b px-3 py-2", textRole.bodyStrong)}>{tDataTable("filterDimensions")}</div>
            <div className="p-2">
              <div className="grid gap-1">
                {draftFacets.map((facet) => (
                  <Button key={facet.id} type="button" variant={facet.id === activeFacet?.id ? "secondary" : "ghost"} size="sm" layout="between" onClick={() => setActiveFacetId(facet.id)}>
                    <span className="min-w-0 truncate">{facet.label}</span>
                    {facet.values.length > 0 ? <TabsCountBadge className="font-normal">{facet.values.length}</TabsCountBadge> : null}
                  </Button>
                ))}
              </div>
            </div>
          </div>) : null}
          <div className="flex min-w-0 flex-1 flex-col">
            {activeFacet ? <>
              <div className={cn("border-b px-3 py-2", textRole.bodyStrong)}>{activeFacet.label}</div>
              <div className="min-h-0 flex-1 overflow-hidden">
                <DataTableFacetedFilterOptions title={activeFacet.label} values={activeFacet.values} onValuesChange={activeFacet.onValuesChange} options={activeFacet.options} emptyLabel={activeFacet.emptyLabel} clearLabel={activeFacet.clearLabel} searchPlaceholder={activeFacet.searchPlaceholder} showClear={false}/>
              </div>
            </> : null}
          </div>
        </div>
        <div className="grid grid-cols-2 gap-2 border-t p-2">
          <Button type="button" variant="outline" size="sm" layout="fullWidth" onClick={resetDraft}>
            {tActions("reset")}
          </Button>
          <Button type="button" size="sm" layout="fullWidth" disabled={!hasDraftChanges} onClick={applyDraft}>
            {tDataTable("applyFilter")}
          </Button>
        </div>
      </PopoverContent>
    </Popover>);
}
