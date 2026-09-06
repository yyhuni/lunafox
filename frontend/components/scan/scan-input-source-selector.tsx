"use client"

import { useTranslations } from "next-intl"

import { Badge } from "@/components/ui/badge"
import { Label } from "@/components/ui/label"
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import { isScanInputSource, type ScanInputSource } from "@/types/scan.types"

interface ScanInputSourceSelectorProps {
  value: ScanInputSource
  onValueChange: (value: ScanInputSource) => void
  disabled?: boolean
  id?: string
}

export function ScanInputSourceSelector({
  value,
  onValueChange,
  disabled = false,
  id = "scan-input-source",
}: ScanInputSourceSelectorProps) {
  const t = useTranslations("scan.inputSource")
  const labelId = `${id}-label`
  const options = [
    {
      value: "scanSnapshot" as const,
      label: t("scanSnapshot"),
      description: t("scanSnapshotDescription"),
      recommended: true,
    },
    {
      value: "targetInventory" as const,
      label: t("targetInventory"),
      description: t("targetInventoryDescription"),
      recommended: false,
    },
  ]

  return (
    <div className="space-y-2">
      <Label id={labelId}>{t("label")}</Label>
      <RadioGroup
        id={id}
        value={value}
        onValueChange={(nextValue) => {
          if (!isScanInputSource(nextValue)) {
            throw new Error("Scan input source selector received an invalid value")
          }
          onValueChange(nextValue)
        }}
        disabled={disabled}
        aria-labelledby={labelId}
        className="gap-2"
      >
        {options.map((option) => {
          const optionId = `${id}-${option.value}`
          const isSelected = value === option.value

          return (
            <label
              key={option.value}
              htmlFor={optionId}
              className={cn(
                "radius-control flex min-w-0 cursor-pointer items-start gap-3 border px-3 py-2.5 transition-colors",
                isSelected ? "border-primary/60 bg-primary/10" : "border-border",
                disabled && "cursor-not-allowed opacity-60"
              )}
            >
              <RadioGroupItem id={optionId} value={option.value} disabled={disabled} className="mt-1 translate-y-px" />
              <span className="min-w-0">
                <span className="flex flex-wrap items-center gap-2">
                  <span className={textRole.bodyStrong}>{option.label}</span>
                  {option.recommended ? <Badge variant="outline">{t("recommended")}</Badge> : null}
                </span>
                <span className={cn("block", textRole.helperText)}>{option.description}</span>
              </span>
            </label>
          )
        })}
      </RadioGroup>
    </div>
  )
}
