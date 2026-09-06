"use client"

import * as React from "react"

import { ChevronDown } from "@/components/icons"
import { Button } from "@/components/ui/button"
import { Command, CommandEmpty, CommandInput, CommandList } from "@/components/ui/command"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"

type ScanSearchablePickerProps = {
  id?: string
  ariaLabel: string
  disabled?: boolean
  trigger: React.ReactNode
  searchPlaceholder: string
  emptyLabel?: string
  children: React.ReactNode
  onOpenChange?: (open: boolean) => void
  searchValue?: string
  onSearchValueChange?: (value: string) => void
  shouldFilter?: boolean
  listRef?: React.Ref<HTMLDivElement>
  listBusy?: boolean
}

// Both workflow and Agent selection must keep the form height stable as their lists grow.
export function ScanSearchablePicker({
  id,
  ariaLabel,
  disabled = false,
  trigger,
  searchPlaceholder,
  emptyLabel,
  children,
  onOpenChange,
  searchValue,
  onSearchValueChange,
  shouldFilter = true,
  listRef,
  listBusy = false,
}: ScanSearchablePickerProps) {
  const [open, setOpen] = React.useState(false)

  const handleOpenChange = (nextOpen: boolean) => {
    setOpen(nextOpen)
    onOpenChange?.(nextOpen)
  }

  const closeAfterSelection = (event: React.MouseEvent<HTMLDivElement>) => {
    if (event.target instanceof Element && event.target.closest("[data-command-item]:not([data-disabled=true])")) {
      handleOpenChange(false)
    }
  }

  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverTrigger
        render={
          <Button
            id={id}
            type="button"
            variant="outline"
            layout="between"
            className="h-auto min-h-10 overflow-hidden py-2 text-left font-normal"
            disabled={disabled}
            aria-label={ariaLabel}
          />
        }
      >
        {trigger}
        <ChevronDown className="size-4 shrink-0 text-muted-foreground" />
      </PopoverTrigger>
      <PopoverContent align="start" className="w-[var(--anchor-width)] overflow-hidden p-0">
        <Command shouldFilter={shouldFilter} onClick={closeAfterSelection}>
          <CommandInput
            placeholder={searchPlaceholder}
            {...(searchValue !== undefined ? { value: searchValue } : {})}
            onChange={(event) => onSearchValueChange?.(event.currentTarget.value)}
          />
          <CommandList ref={listRef} aria-busy={listBusy} className="max-h-52 sm:max-h-72">
            {emptyLabel ? <CommandEmpty>{emptyLabel}</CommandEmpty> : null}
            {children}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  )
}
