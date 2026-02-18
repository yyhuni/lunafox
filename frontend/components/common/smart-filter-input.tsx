"use client"

import { IconSearch } from "@/components/icons"
import { useTranslations } from "next-intl"
import {
  Popover,
  PopoverContent,
  PopoverAnchor,
} from "@/components/ui/popover"
import { Button } from "@/components/ui/button"

import {
  parseFilterExpression,
  PREDEFINED_FIELDS,
  type FilterField,
  type ParsedFilter,
  getTranslatedFields,
} from "@/lib/smart-filter"
import { useSmartFilterInputState } from "@/components/common/smart-filter-input-state"
import { SmartFilterInputField } from "@/components/common/smart-filter-input-field"
import { SmartFilterInputMenu } from "@/components/common/smart-filter-input-menu"

// Default fields (IP Addresses page)
const DEFAULT_FIELDS: FilterField[] = [
  PREDEFINED_FIELDS.ip,
  PREDEFINED_FIELDS.port,
  PREDEFINED_FIELDS.host,
]

export type { FilterField, ParsedFilter }
export { PREDEFINED_FIELDS, getTranslatedFields }

interface SmartFilterInputProps {
  /** Available filter fields, uses default fields if not provided */
  fields?: FilterField[]
  /** Combination examples (complete examples using logical operators) */
  examples?: string[]
  placeholder?: string
  /** Controlled mode: current filter value */
  value?: string
  onSearch?: (filters: ParsedFilter[], rawQuery: string) => void
  className?: string
}

export function SmartFilterInput({
  fields = DEFAULT_FIELDS,
  examples,
  placeholder,
  value,
  onSearch,
  className,
}: SmartFilterInputProps) {
  const t = useTranslations("filter")
  const {
    open,
    setOpen,
    inputValue,
    inputRef,
    ghostRef,
    listRef,
    ghostText,
    parsedFilters,
    defaultPlaceholder,
    currentWord,
    showFieldSuggestions,
    handleSelectSuggestion,
    handleSearch,
    handleKeyDown,
    handleAppendExample,
    handleInputChange,
    handleBlur,
  } = useSmartFilterInputState({
    fields,
    examples,
    value,
    onSearch,
  })

  return (
    <div className={className}>
      <div className="flex items-center gap-2">
        <Popover open={open} onOpenChange={setOpen} modal={false}>
          <PopoverAnchor asChild>
            <SmartFilterInputField
              value={inputValue}
              ghostText={ghostText}
              inputRef={inputRef}
              ghostRef={ghostRef}
              placeholder={placeholder || defaultPlaceholder}
              onChange={handleInputChange}
              onFocus={() => setOpen(true)}
              onBlur={handleBlur}
              onKeyDown={handleKeyDown}
            />
          </PopoverAnchor>
        <PopoverContent
          className="w-[var(--radix-popover-trigger-width)] p-0"
          align="start"
          side="bottom"
          sideOffset={4}
          collisionPadding={16}
          onOpenAutoFocus={(e) => e.preventDefault()}
          onCloseAutoFocus={(e) => e.preventDefault()}
          onPointerDownOutside={(e) => {
            // If clicking on input box, don't close popover
            if (inputRef.current?.contains(e.target as Node)) {
              e.preventDefault()
            }
          }}
        >
          <SmartFilterInputMenu
            t={t}
            listRef={listRef}
            parsedFilters={parsedFilters}
            showFieldSuggestions={showFieldSuggestions}
            fields={fields}
            currentWord={currentWord}
            examples={examples}
            onSelectSuggestion={handleSelectSuggestion}
            onAppendExample={handleAppendExample}
          />
        </PopoverContent>
      </Popover>
        <Button variant="outline" size="sm" onClick={handleSearch}>
          <IconSearch className="h-4 w-4" />
        </Button>
      </div>
    </div>
  )
}

export { parseFilterExpression, DEFAULT_FIELDS }
