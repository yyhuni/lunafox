import * as React from "react"
import { Badge } from "@/components/ui/badge"
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandItem,
  CommandList,
} from "@/components/ui/command"
import type { FilterField, ParsedFilter } from "@/lib/smart-filter"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

interface SmartFilterInputMenuProps {
  t: TranslationFn
  listRef: React.RefObject<HTMLDivElement | null>
  parsedFilters: ParsedFilter[]
  showFieldSuggestions: boolean
  fields: FilterField[]
  currentWord: string
  examples?: string[]
  onSelectSuggestion: (value: string) => void
  onAppendExample: (example: string) => void
}

export function SmartFilterInputMenu({
  t,
  listRef,
  parsedFilters,
  showFieldSuggestions,
  fields,
  currentWord,
  examples,
  onSelectSuggestion,
  onAppendExample,
}: SmartFilterInputMenuProps) {
  return (
    <Command>
      <CommandList ref={listRef}>
        {parsedFilters.length > 0 && (
          <CommandGroup heading={t("groups.activeFilters")}>
            <div className="flex flex-wrap gap-1 px-2 py-1">
              {parsedFilters.map((filter, index) => (
                <Badge key={index} variant="secondary" className="text-xs font-mono">
                  {filter.raw}
                </Badge>
              ))}
            </div>
          </CommandGroup>
        )}

        {showFieldSuggestions && (
          <CommandGroup heading={t("groups.availableFields")}>
            <div className="flex flex-wrap gap-1 px-2 py-1">
              {fields
                .filter((field) => !currentWord || field.key.startsWith(currentWord.toLowerCase()))
                .map((field) => (
                  <Badge
                    asChild
                    key={field.key}
                    variant="outline"
                    className="text-xs font-mono hover:bg-accent"
                  >
                    <button
                      type="button"
                      onClick={() => onSelectSuggestion(`${field.key}="`)}
                    >
                      {field.key}
                    </button>
                  </Badge>
                ))}
            </div>
          </CommandGroup>
        )}

        <CommandGroup heading={t("groups.syntax")}>
          <div className="px-2 py-1.5 text-xs text-muted-foreground space-y-2">
            <div className="space-y-1">
              <div className="font-medium text-foreground/80">{t("syntax.operators")}</div>
              <div className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-0.5">
                <code className="bg-muted px-1 rounded">=</code>
                <span>{t("syntax.containsFuzzy")}</span>
                <code className="bg-muted px-1 rounded">==</code>
                <span>{t("syntax.exactMatch")}</span>
                <code className="bg-muted px-1 rounded">!=</code>
                <span>{t("syntax.notEquals")}</span>
              </div>
            </div>
            <div className="space-y-1 pt-1 border-t border-muted">
              <div className="font-medium text-foreground/80">{t("syntax.logic")}</div>
              <div className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-0.5">
                <span>
                  <code className="bg-muted px-1 rounded">||</code>{" "}
                  <code className="bg-muted px-1 rounded">or</code>
                </span>
                <span>{t("syntax.matchAny")}</span>
                <span>
                  <code className="bg-muted px-1 rounded">&&</code>{" "}
                  <code className="bg-muted px-1 rounded">and</code>{" "}
                  <code className="bg-muted px-1 rounded">space</code>
                </span>
                <span>{t("syntax.matchAll")}</span>
              </div>
            </div>
          </div>
        </CommandGroup>

        {examples && examples.length > 0 && (
          <CommandGroup heading={t("groups.examples")}>
            {examples.map((example, index) => (
              <CommandItem
                key={index}
                value={example}
                onSelect={() => onAppendExample(example)}
              >
                <code className="text-xs">{example}</code>
              </CommandItem>
            ))}
          </CommandGroup>
        )}

        <CommandEmpty>{t("empty")}</CommandEmpty>
      </CommandList>
    </Command>
  )
}
