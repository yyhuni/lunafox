"use client"

import * as React from "react"
import { useLocale, useTranslations } from "next-intl"

import {
  ChevronDown,
  Clock,
  RefreshCw,
} from "@/components/icons"
import { Button } from "@/components/ui/button"
import { FieldError } from "@/components/ui/field"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import {
  WheelPicker,
  WheelPickerWrapper,
  type WheelPickerOption,
} from "@ncdai/react-wheel-picker"
import "@ncdai/react-wheel-picker/style.css"

const SECONDS_PER_MINUTE = 60
const SECONDS_PER_HOUR = 60 * SECONDS_PER_MINUTE
const SECONDS_PER_DAY = 24 * SECONDS_PER_HOUR

export interface DurationParts {
  days: number
  hours: number
  minutes: number
  seconds: number
}

export interface EngineDurationInputProps {
  id: string
  value: number
  defaultValue?: number
  minimum?: number
  maximum?: number
  disabled?: boolean
  onChange: (value: number) => void
}

export function durationPartsFromSeconds(value: number): DurationParts {
  const total = Math.max(0, Math.trunc(value))
  const days = Math.floor(total / SECONDS_PER_DAY)
  const dayRemainder = total % SECONDS_PER_DAY
  const hours = Math.floor(dayRemainder / SECONDS_PER_HOUR)
  const hourRemainder = dayRemainder % SECONDS_PER_HOUR
  const minutes = Math.floor(hourRemainder / SECONDS_PER_MINUTE)
  const seconds = hourRemainder % SECONDS_PER_MINUTE
  return { days, hours, minutes, seconds }
}

export function durationSecondsFromParts(parts: DurationParts): number {
  return (
    parts.days * SECONDS_PER_DAY
    + parts.hours * SECONDS_PER_HOUR
    + parts.minutes * SECONDS_PER_MINUTE
    + parts.seconds
  )
}

function isWithinBounds(value: number, minimum?: number, maximum?: number): boolean {
  if (!Number.isInteger(value)) return false
  if (minimum !== undefined && value < minimum) return false
  if (maximum !== undefined && value > maximum) return false
  return true
}

function optionsForRange(maximum: number, pad = true): WheelPickerOption<number>[] {
  return Array.from({ length: maximum + 1 }, (_, value) => ({
    value,
    label: pad ? String(value).padStart(2, "0") : String(value),
  }))
}

function dayMaximumFor(
  parts: DurationParts,
  value: number,
  defaultValue: number | undefined,
  minimum: number | undefined,
  maximum: number | undefined,
): number {
  if (maximum !== undefined) {
    return Math.max(0, Math.floor(maximum / SECONDS_PER_DAY))
  }

  const knownDays = [value, defaultValue, minimum]
    .filter((candidate): candidate is number => candidate !== undefined && candidate >= 0)
    .map((candidate) => durationPartsFromSeconds(candidate).days)

  // Default to at least 7 days for balanced column density and no awkward empty void,
  // growing dynamically if the user moves past it.
  return Math.max(parts.days + 3, ...knownDays, 7)
}

function formatSeconds(value: number, locale: string): string {
  return new Intl.NumberFormat(locale).format(value)
}

function formatHumanDuration(seconds: number, locale: string): string {
  if (seconds <= 0) return locale === "zh" ? "0 秒" : "0s"
  const d = Math.floor(seconds / SECONDS_PER_DAY)
  const h = Math.floor((seconds % SECONDS_PER_DAY) / SECONDS_PER_HOUR)
  const m = Math.floor((seconds % SECONDS_PER_HOUR) / SECONDS_PER_MINUTE)
  const s = seconds % SECONDS_PER_MINUTE

  const parts: string[] = []
  if (locale === "zh") {
    if (d > 0) parts.push(`${d} 天`)
    if (h > 0) parts.push(`${h} 小时`)
    if (m > 0 && d === 0) parts.push(`${m} 分钟`)
    if (s > 0 && d === 0 && h === 0) parts.push(`${s} 秒`)
  } else {
    if (d > 0) parts.push(`${d}d`)
    if (h > 0) parts.push(`${h}h`)
    if (m > 0 && d === 0) parts.push(`${m}m`)
    if (s > 0 && d === 0 && h === 0) parts.push(`${s}s`)
  }
  return parts.slice(0, 2).join(" ") || (locale === "zh" ? `${s} 秒` : `${s}s`)
}

export function EngineDurationInput({
  id,
  value,
  defaultValue,
  minimum,
  maximum,
  disabled = false,
  onChange,
}: EngineDurationInputProps) {
  const locale = useLocale()
  const t = useTranslations("scan.initiate.engineConfigForm")
  const [open, setOpen] = React.useState(false)
  const [draft, setDraft] = React.useState<DurationParts>(() => durationPartsFromSeconds(value))

  React.useEffect(() => {
    if (!open) setDraft(durationPartsFromSeconds(value))
  }, [open, value])

  const candidate = durationSecondsFromParts(draft)
  const validCandidate = isWithinBounds(candidate, minimum, maximum)
  const errorId = `${id}-duration-error`
  const formattedSeconds = formatSeconds(value, locale)
  const currentLabel = `${formattedSeconds} ${t("unitSeconds")}`
  const daysMaximum = dayMaximumFor(draft, value, defaultValue, minimum, maximum)
  const dayOptions = React.useMemo(() => optionsForRange(daysMaximum, true), [daysMaximum])
  const hourOptions = React.useMemo(() => optionsForRange(23, true), [])
  const minuteOptions = React.useMemo(() => optionsForRange(59, true), [])
  const secondOptions = React.useMemo(() => optionsForRange(59, true), [])

  const updatePart = React.useCallback((part: keyof DurationParts, next: number) => {
    setDraft((current) => ({ ...current, [part]: next }))
  }, [])

  const closeAndDiscard = React.useCallback(() => {
    setDraft(durationPartsFromSeconds(value))
    setOpen(false)
  }, [value])

  const rangeMessage = React.useMemo(() => {
    if (validCandidate) return null
    const lower = minimum === undefined ? t("unbounded") : formatSeconds(minimum, locale)
    const upper = maximum === undefined ? t("unbounded") : formatSeconds(maximum, locale)
    return t("durationOutOfRange", { min: lower, max: upper })
  }, [locale, maximum, minimum, t, validCandidate])

  const humanValue = React.useMemo(() => formatHumanDuration(value, locale), [locale, value])

  return (
    <div>
      <Popover
        open={open}
        onOpenChange={(nextOpen) => {
          if (nextOpen) {
            setDraft(durationPartsFromSeconds(value))
            setOpen(true)
          } else {
            closeAndDiscard()
          }
        }}
      >
        <PopoverTrigger
          render={(
            <Button
              id={id}
              type="button"
              variant="outline"
              size="sm"
              layout="between"
              className="font-normal"
              disabled={disabled}
              aria-label={`${t("chooseDuration")}: ${currentLabel}`}
              aria-invalid={rangeMessage ? true : undefined}
              aria-describedby={rangeMessage ? errorId : undefined}
              title={`${t("chooseDuration")}: ${currentLabel}`}
              data-duration-picker="seconds"
            />
          )}
        >
          <div className="flex min-w-0 items-center gap-2">
            <Clock aria-hidden className="size-4 shrink-0 text-muted-foreground" />
            <span className="flex min-w-0 items-center gap-1.5 tabular-nums text-foreground font-medium">
              <span>{formattedSeconds}</span>
              <span className={cn(textRole.metadataLabel, "font-normal")}>{t("unitSeconds")}</span>
              <span className="text-muted-foreground text-xs font-normal">
                ({humanValue})
              </span>
            </span>
          </div>
          <ChevronDown aria-hidden className="size-4 shrink-0 text-muted-foreground" />
        </PopoverTrigger>
        <PopoverContent
          align="end"
          aria-label={t("chooseDuration")}
          className="w-80 p-0"
        >
          <div className="border-b border-border/70 bg-muted/30 px-3 py-1.5">
            <div className="grid grid-cols-4 text-center">
              {([
                ["days", "durationDays"],
                ["hours", "durationHours"],
                ["minutes", "durationMinutes"],
                ["seconds", "durationSeconds"],
              ] as const).map(([part, labelKey]) => (
                <span key={part} className={cn("text-[11px] font-medium text-muted-foreground", textRole.compactCaption)}>
                  {t(labelKey)}
                </span>
              ))}
            </div>
          </div>

          {/* Wheel Picker area */}
          <div
            role="group"
            aria-label={t("chooseDuration")}
            className="relative bg-background py-1"
          >
            <WheelPickerWrapper className="h-44 divide-x divide-border/40">
              <WheelPicker
                value={draft.days}
                options={dayOptions}
                optionItemHeight={32}
                classNames={{
                  optionItem: "text-muted-foreground/60 tabular-nums text-xs transition-opacity",
                  highlightWrapper: "border-y border-border/80 bg-accent/40 dark:bg-accent/30",
                  highlightItem: "tabular-nums font-semibold text-foreground",
                }}
                onValueChange={(next) => updatePart("days", next)}
              />
              <WheelPicker
                value={draft.hours}
                options={hourOptions}
                optionItemHeight={32}
                classNames={{
                  optionItem: "text-muted-foreground/60 tabular-nums text-xs transition-opacity",
                  highlightWrapper: "border-y border-border/80 bg-accent/40 dark:bg-accent/30",
                  highlightItem: "tabular-nums font-semibold text-foreground",
                }}
                onValueChange={(next) => updatePart("hours", next)}
              />
              <WheelPicker
                value={draft.minutes}
                options={minuteOptions}
                optionItemHeight={32}
                classNames={{
                  optionItem: "text-muted-foreground/60 tabular-nums text-xs transition-opacity",
                  highlightWrapper: "border-y border-border/80 bg-accent/40 dark:bg-accent/30",
                  highlightItem: "tabular-nums font-semibold text-foreground",
                }}
                onValueChange={(next) => updatePart("minutes", next)}
              />
              <WheelPicker
                value={draft.seconds}
                options={secondOptions}
                optionItemHeight={32}
                classNames={{
                  optionItem: "text-muted-foreground/60 tabular-nums text-xs transition-opacity",
                  highlightWrapper: "border-y border-border/80 bg-accent/40 dark:bg-accent/30",
                  highlightItem: "tabular-nums font-semibold text-foreground",
                }}
                onValueChange={(next) => updatePart("seconds", next)}
              />
            </WheelPickerWrapper>
          </div>

          {rangeMessage ? (
            <div className="border-t border-destructive/20 bg-destructive/5 px-3 py-1.5">
              <FieldError id={errorId}>{rangeMessage}</FieldError>
            </div>
          ) : null}

          {/* Footer action bar */}
          <div className="flex items-center justify-between gap-2 border-t border-border/70 bg-muted/20 px-3 py-1.5">
            <Button
              type="button"
              variant="ghost"
              size="xs"
              disabled={disabled || defaultValue === undefined}
              onClick={() => {
                if (defaultValue !== undefined) setDraft(durationPartsFromSeconds(defaultValue))
              }}
              className="text-muted-foreground hover:text-foreground"
            >
              <RefreshCw aria-hidden className="size-3" />
              <span>{t("restoreDefault")}</span>
            </Button>
            <div className="flex items-center gap-1.5">
              <Button type="button" variant="ghost" size="xs" onClick={closeAndDiscard}>
                {t("cancel")}
              </Button>
              <Button
                type="button"
                variant="primary"
                size="xs"
                disabled={disabled || !validCandidate}
                onClick={() => {
                  if (!validCandidate) return
                  onChange(candidate)
                  setOpen(false)
                }}
              >
                {t("apply")}
              </Button>
            </div>
          </div>
        </PopoverContent>
      </Popover>
    </div>
  )
}
