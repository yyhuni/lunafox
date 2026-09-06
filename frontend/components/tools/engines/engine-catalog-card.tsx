"use client"

import { useTranslations } from "next-intl"

import { ChevronRight } from "@/components/icons"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, cardVariants } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { getLocalizedEngineSummaryDisplay } from "@/lib/engine-catalog"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import type { Locale } from "@/i18n/config"
import type { EngineCatalogSummary } from "@/types/engine-catalog.types"

const ENGINE_CARD_SURFACE_CLASS = cn(
  cardVariants({ variant: "shell" }),
  "min-h-56 w-full min-w-0 items-stretch justify-start gap-0 overflow-hidden py-0 text-left whitespace-normal"
)
const ENGINE_CARD_HEADER_CLASS = "flex min-w-0 flex-col gap-2 px-4 pt-4 pb-3"
const ENGINE_CARD_CAPABILITIES_CLASS = "grid gap-2.5 border-y border-border/60 bg-muted/15 px-4 py-3"
const ENGINE_CARD_CAPABILITY_ROW_CLASS = "grid min-w-0 grid-cols-[5rem_minmax(0,1fr)] items-center gap-2"
const ENGINE_CARD_FOOTER_CLASS = "mt-auto flex min-w-0 items-center justify-between gap-3 px-4 py-3"

interface EngineCatalogCardProps {
  engine: EngineCatalogSummary
  locale: Locale
  onSelect: (engine: EngineCatalogSummary) => void
}

export function EngineCatalogCard({ engine, locale, onSelect }: EngineCatalogCardProps) {
  const t = useTranslations("pages.engines")
  const display = getLocalizedEngineSummaryDisplay(engine, locale)
  const executionResourceCount = engine.execution.executionResources?.length ?? 0

  return (
    <Button
      type="button"
      variant="outline"
      size="content"
      className={cn(
        ENGINE_CARD_SURFACE_CLASS,
        "group hover:border-foreground/30 hover:bg-card focus-visible:border-ring focus-visible:ring-ring/40"
      )}
      aria-label={`${t("viewDetails")}: ${display.displayName}`}
      onClick={() => onSelect(engine)}
    >
      <span className={ENGINE_CARD_HEADER_CLASS}>
        <span className="flex min-w-0 items-start justify-between gap-3">
          <span className={cn("min-w-0 truncate", textRole.panelTitle)}>{display.displayName}</span>
          <Badge variant="count" className="font-mono">{engine.packageVersion}</Badge>
        </span>
        <span className={cn("line-clamp-2 min-h-10", textRole.bodySubtle)}>{display.description}</span>
      </span>

      <span className={ENGINE_CARD_CAPABILITIES_CLASS}>
        <EngineCapabilityRow label={t("supportedTargets")}>
          {engine.execution.supportedTargetTypes.map((targetType) => (
            <Badge key={targetType} variant="secondary">{targetType}</Badge>
          ))}
        </EngineCapabilityRow>
      </span>

      <span className={ENGINE_CARD_FOOTER_CLASS}>
        <span className={cn("min-w-0 truncate", textRole.caption)}>
          {t("publisher")}: <span className="text-foreground">{engine.publisher}</span>
        </span>
        <span className={cn("flex shrink-0 items-center gap-1.5", textRole.caption)}>
          {t("executionResources")}: <span className="tabular-nums text-foreground">{executionResourceCount}</span>
          <ChevronRight aria-hidden="true" className="size-4" />
        </span>
      </span>
    </Button>
  )
}

function EngineCapabilityRow({ label, children }: React.PropsWithChildren<{ label: string }>) {
  return (
    <span className={ENGINE_CARD_CAPABILITY_ROW_CLASS}>
      <span className={textRole.helperText}>{label}</span>
      <span className="flex min-h-5 min-w-0 flex-wrap items-center gap-1">{children}</span>
    </span>
  )
}

export function EngineCatalogCardLoadingState() {
  const t = useTranslations("pages.engines")

  return (
    <Card data-slot="engine-card-loading-state" className={ENGINE_CARD_SURFACE_CLASS}>
      <div className={ENGINE_CARD_HEADER_CLASS}>
        <div className="flex items-start justify-between gap-3">
          <EngineTextPlaceholder roleClassName={textRole.panelTitle} className="w-32" />
          <Badge aria-hidden="true" variant="count" className="relative font-mono text-transparent">
            0.0.0
            <Skeleton className="absolute inset-0 radius-badge" />
          </Badge>
        </div>
        <EngineTextPlaceholder roleClassName={textRole.bodySubtle} lineWidths={["w-full", "w-3/4"]} />
      </div>

      <div className={ENGINE_CARD_CAPABILITIES_CLASS}>
        <LoadingCapabilityRow label={t("supportedTargets")} widths={["w-14", "w-10", "w-12"]} />
      </div>

      <div className={ENGINE_CARD_FOOTER_CLASS}>
        <EngineTextPlaceholder roleClassName={textRole.caption} className="w-28" />
        <EngineTextPlaceholder roleClassName={textRole.caption} className="w-24" />
      </div>
    </Card>
  )
}

function EngineTextPlaceholder({
  roleClassName,
  className,
  lineWidths = ["w-full"],
}: {
  roleClassName: string
  className?: string
  lineWidths?: string[]
}) {
  return (
    <span aria-hidden="true" className={cn("relative block min-w-0", roleClassName, className)}>
      <span className="invisible whitespace-pre-line">
        {lineWidths.map(() => "\u00a0").join("\n")}
      </span>
      <span className="absolute inset-0 flex flex-col">
        {lineWidths.map((width, index) => (
          <span key={`${width}-${index}`} className="relative min-h-0 flex-1">
            <Skeleton className={cn("absolute top-1/2 left-0 h-4 -translate-y-1/2 radius-control", width)} />
          </span>
        ))}
      </span>
    </span>
  )
}

function LoadingCapabilityRow({ label, widths }: { label: string; widths: string[] }) {
  return (
    <div className={ENGINE_CARD_CAPABILITY_ROW_CLASS}>
      <span className={textRole.helperText}>{label}</span>
      <div className="flex min-h-5 flex-wrap items-center gap-1">
        {widths.map((width, index) => (
          <Skeleton key={`${width}-${index}`} className={cn("h-5 radius-badge", width)} />
        ))}
      </div>
    </div>
  )
}
