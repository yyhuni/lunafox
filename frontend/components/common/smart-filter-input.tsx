"use client";
import { IconSearch } from "@/components/icons";
import { useTranslations } from "next-intl";
import { Popover, PopoverContent, PopoverAnchor, } from "@/components/ui/popover";
import { Button } from "@/components/ui/button";
import { parseFilterExpression, type FilterField, type ParsedFilter, getTranslatedFields, } from "@/lib/smart-filter";
import { useSmartFilterInputState } from "@/components/common/smart-filter-input-state";
import { SmartFilterInputField } from "@/components/common/smart-filter-input-field";
import { SmartFilterInputMenu } from "@/components/common/smart-filter-input-menu";
import { cn } from "@/lib/utils";
import type { DataTableToolbarDensity } from "@/types/data-table.types";
export type { FilterField, ParsedFilter };
export { getTranslatedFields };
interface SmartFilterInputProps {
    /** Available filter fields, uses default fields if not provided */
    fields?: FilterField[];
    /** Combination examples (complete examples using logical operators) */
    examples?: string[];
    placeholder?: string;
    /** Controlled mode: current filter value */
    value?: string;
    onSearch?: (filters: ParsedFilter[], rawQuery: string) => void;
    toolbarDensity?: DataTableToolbarDensity;
    density?: DataTableToolbarDensity;
    className?: string;
    fieldClassName?: string;
    toolbarClassName?: string;
    inputClassName?: string;
    searchButtonLabel?: string;
    searchButtonMode?: "inline" | "icon" | "attached-primary";
}
export function SmartFilterInput({ fields, examples, placeholder, value, onSearch, toolbarDensity, density, className, fieldClassName, toolbarClassName, inputClassName, searchButtonLabel, searchButtonMode = "icon", }: SmartFilterInputProps) {
    const t = useTranslations("filter");
    const translatedFields = getTranslatedFields(t);
    const resolvedFields = fields ?? [
        translatedFields.ip,
        translatedFields.port,
        translatedFields.host,
    ];
    const resolvedToolbarDensity = toolbarDensity ?? density ?? "compact";
    const isStandardDensity = resolvedToolbarDensity === "standard";
    const { open, setOpen, inputValue, inputRef, ghostRef, listRef, ghostText, parsedFilters, defaultPlaceholder, currentWord, showFieldSuggestions, handleSelectSuggestion, handleSearch, handleKeyDown, handleAppendExample, handleInputChange, handleBlur, } = useSmartFilterInputState({
        fields: resolvedFields,
        examples,
        value,
        onSearch,
    });
    const resolvedSearchButtonSize = isStandardDensity ? "icon" : "icon-sm";
    const defaultSearchButtonLabel = placeholder || defaultPlaceholder || t("search");
    const resolvedSearchButtonLabel = searchButtonLabel ?? defaultSearchButtonLabel;
    const searchButtonContent = (<>
          <IconSearch data-icon="inline-start"/>
          {searchButtonLabel ? <span>{searchButtonLabel}</span> : null}
        </>);
    const handlePopoverOpenChange = (nextOpen: boolean, eventDetails: { reason?: string; event?: Event; cancel?: () => void }) => {
        if (!nextOpen && eventDetails.reason === "outside-press" && inputRef.current?.contains(eventDetails.event?.target as Node)) {
            eventDetails.cancel?.();
            return;
        }
        setOpen(nextOpen);
    };
    return (<div className={className}>
      <div className={cn("flex gap-2 items-center", toolbarClassName)}>
        <Popover open={open} onOpenChange={handlePopoverOpenChange} modal={false}>
          <PopoverAnchor render={<SmartFilterInputField value={inputValue} ghostText={ghostText} inputRef={inputRef} ghostRef={ghostRef} placeholder={placeholder || defaultPlaceholder} onChange={handleInputChange} onFocus={() => setOpen(true)} onClick={() => setOpen(true)} onBlur={handleBlur} onKeyDown={handleKeyDown} showIcon={searchButtonMode === "inline"} toolbarDensity={resolvedToolbarDensity} className={fieldClassName} inputClassName={inputClassName}/>}></PopoverAnchor>
        <PopoverContent className="p-0 w-[var(--anchor-width)]" align="start" side="bottom" sideOffset={4} collisionPadding={16} initialFocus={false} finalFocus={false}>
          <SmartFilterInputMenu t={t} listRef={listRef} parsedFilters={parsedFilters} showFieldSuggestions={showFieldSuggestions} fields={resolvedFields} currentWord={currentWord} examples={examples} onSelectSuggestion={handleSelectSuggestion} onAppendExample={handleAppendExample}/>
        </PopoverContent>
      </Popover>
        {searchButtonMode === "attached-primary" ? (<Button variant="primary" size="default" className="radius-none border-l border-l-primary-foreground/20 shadow-none focus-visible:z-10" onClick={handleSearch} aria-label={resolvedSearchButtonLabel}>
            {searchButtonContent}
          </Button>) : searchButtonMode === "icon" ? (<Button variant="outline" size={resolvedSearchButtonSize} onClick={handleSearch} aria-label={resolvedSearchButtonLabel}>
            {searchButtonContent}
          </Button>) : null}
      </div>
    </div>);
}
export { parseFilterExpression };
