"use client"

import React from "react"

import { IconCheck } from "@/components/icons"
import { ScanSearchablePicker } from "@/components/scan/scan-searchable-picker"
import { CommandGroup, CommandItem } from "@/components/ui/command"
import { Label } from "@/components/ui/label"
import { getSupportedTimeZones } from "@/lib/scheduled-scan-helpers"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

type ScheduledScanTimeZoneFieldProps = {
  id: string
  value: string
  onChange: (value: string) => void
  disabled?: boolean
  label: string
  placeholder: string
  searchPlaceholder: string
  emptyLabel: string
  description: string
}

export function ScheduledScanTimeZoneField({
  id,
  value,
  onChange,
  disabled = false,
  label,
  placeholder,
  searchPlaceholder,
  emptyLabel,
  description,
}: ScheduledScanTimeZoneFieldProps) {
  const timeZones = React.useMemo(() => getSupportedTimeZones(value), [value])

  return (
    <div className="space-y-2">
      <Label htmlFor={id}>{label} *</Label>
      <ScanSearchablePicker
        id={id}
        ariaLabel={label}
        disabled={disabled}
        trigger={(
          <span className={cn("min-w-0 flex-1 truncate", value ? textRole.body : textRole.bodySubtle)}>
            {value || placeholder}
          </span>
        )}
        searchPlaceholder={searchPlaceholder}
        emptyLabel={emptyLabel}
      >
        <CommandGroup>
          {timeZones.map((timeZone) => (
            <CommandItem
              key={timeZone}
              value={timeZone}
              keywords={[timeZone]}
              onSelect={() => onChange(timeZone)}
              className="gap-2"
            >
              <span className={cn("min-w-0 flex-1 truncate", textRole.body)}>{timeZone}</span>
              {value === timeZone && <IconCheck className="size-4 shrink-0" />}
            </CommandItem>
          ))}
        </CommandGroup>
      </ScanSearchablePicker>
      <p className={cn("text-muted-foreground", textRole.helperText)}>{description}</p>
    </div>
  )
}
