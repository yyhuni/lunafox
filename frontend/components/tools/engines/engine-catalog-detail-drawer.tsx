"use client"

import * as React from "react"
import { useTranslations } from "next-intl"

import {
  DetailDrawer,
  DetailDrawerTabs,
  DetailDrawerTabsContent,
  DetailDrawerTabsList,
  DetailDrawerTabsTrigger,
} from "@/components/shared/detail-drawer"
import { AppErrorState } from "@/components/shared/feedback/app-error-state"
import { CopyButton } from "@/components/shared/feedback/copy-button"
import { getLoadingOwnerAttributes } from "@/components/shared/loading/loading-owner"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { useEngineCatalogDetail } from "@/hooks/use-engine-catalog"
import {
  getLocalizedEngineSummaryDisplay,
  localizeEngineCatalogDetail,
} from "@/lib/engine-catalog"
import { normalizeError } from "@/lib/errors/normalize-error"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import type { Locale } from "@/i18n/config"
import type { EngineParamDefinition, EngineParamValue } from "@/types/engine-config.types"
import type { EngineCatalogDetail, EngineCatalogSummary } from "@/types/engine-catalog.types"

interface EngineCatalogDetailDrawerProps {
  engine: EngineCatalogSummary | null
  locale: Locale
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function EngineCatalogDetailDrawer({
  engine,
  locale,
  open,
  onOpenChange,
}: EngineCatalogDetailDrawerProps) {
  const t = useTranslations("pages.engines")
  const detailQuery = useEngineCatalogDetail(engine?.engineId ?? "", open && engine !== null)
  const summaryDisplay = engine ? getLocalizedEngineSummaryDisplay(engine, locale) : null

  return (
    <DetailDrawer
      open={open}
      onOpenChange={onOpenChange}
      title={summaryDisplay?.displayName ?? ""}
      description={summaryDisplay?.description ?? t("detailDescription")}
      titleMeta={engine ? <Badge variant="count" className="font-mono">{engine.packageVersion}</Badge> : null}
      headerMeta={summaryDisplay ? (
        <div data-slot="engine-detail-description" className={cn("min-w-0 line-clamp-2", textRole.bodySubtle)}>
          {summaryDisplay.description}
        </div>
      ) : null}
    >
      {open && engine ? (
        <EngineCatalogDetailBody
          detail={detailQuery.data}
          error={detailQuery.error}
          isPending={detailQuery.isPending}
          locale={locale}
          onRetry={detailQuery.refetch}
        />
      ) : null}
    </DetailDrawer>
  )
}

function EngineCatalogDetailBody({
  detail,
  error,
  isPending,
  locale,
  onRetry,
}: {
  detail: EngineCatalogDetail | undefined
  error: Error | null
  isPending: boolean
  locale: Locale
  onRetry: () => Promise<unknown>
}) {
  const t = useTranslations("pages.engines")

  if (isPending || !detail && !error) {
    return <EngineCatalogDetailLoadingState />
  }

  if (error || !detail) {
    const normalizedError = normalizeError(error ?? new Error("Engine detail is unavailable"), {
      notFoundKind: "resource-not-found",
    })
    return (
      <div className="min-h-0 flex-1 overflow-y-auto px-6 py-5">
        <AppErrorState
          error={{ ...normalizedError, retryable: true }}
          title={t("detailUnavailable")}
          onRetry={onRetry}
          variant="section"
        />
      </div>
    )
  }

  const localized = localizeEngineCatalogDetail(detail, locale)

  return (
    <DetailDrawerTabs defaultValue="overview">
      <DetailDrawerTabsList className="px-6">
        <DetailDrawerTabsTrigger value="overview">{t("overview")}</DetailDrawerTabsTrigger>
        <DetailDrawerTabsTrigger value="configuration">{t("configuration")}</DetailDrawerTabsTrigger>
      </DetailDrawerTabsList>

      <DetailDrawerTabsContent value="overview" className="min-h-0 flex-1 overflow-y-auto px-6 py-5">
        <div className="space-y-6">
          <EngineDrawerSection label={t("identity")}>
            <dl className="grid grid-cols-1 gap-x-6 gap-y-4 sm:grid-cols-2">
              <EngineDetailField label={t("engineId")} value={detail.engineId} mono className="sm:col-span-2" />
              <EngineDetailField label={t("publisher")} value={detail.publisher} />
              <EngineDetailField label={t("packageVersion")} value={detail.packageVersion} mono />
              <EngineDetailField label={t("manifestVersion")} value={detail.manifestVersion} mono />
              <EngineDetailField label={t("engineApiVersion")} value={`v${detail.execution.engineApiMajor}`} mono />
            </dl>
          </EngineDrawerSection>

          <EngineDrawerSection label={t("capabilities")}>
            <EngineBadgeField label={t("supportedTargets")} values={detail.execution.supportedTargetTypes} emptyLabel={t("none")} />
            <EngineBadgeField label={t("executionResources")} values={detail.execution.executionResources ?? []} emptyLabel={t("none")} />
          </EngineDrawerSection>

          <EngineDrawerSection label={t("packageProvenance")}>
            <EngineCopyField label={t("artifactRef")} value={detail.artifactRef} toastId={`engine-artifact-${detail.engineId}`} />
            <EngineCopyField label={t("packageDigest")} value={detail.packageDigest} toastId={`engine-package-${detail.engineId}`} />
          </EngineDrawerSection>
        </div>
      </DetailDrawerTabsContent>

      <DetailDrawerTabsContent value="configuration" className="min-h-0 flex-1 overflow-y-auto px-6 py-5">
        {localized.execution.configSections.length === 0 ? (
          <p className={textRole.bodySubtle}>{t("noConfiguration")}</p>
        ) : (
          <div className="space-y-7">
            {localized.execution.configSections.map((section) => (
              <section key={section.id} className="min-w-0 space-y-4 border-b border-border/70 pb-7 last:border-b-0 last:pb-0">
                <div className="flex min-w-0 items-start justify-between gap-3">
                  <div className="min-w-0 space-y-1">
                    <h3 className={textRole.sectionTitle}>{section.name}</h3>
                    {section.description ? <p className={textRole.bodySubtle}>{section.description}</p> : null}
                  </div>
                  <Badge variant="outline">
                    {section.defaultEnabled ? t("defaultEnabled") : t("defaultDisabled")}
                  </Badge>
                </div>

                <div className="divide-y divide-border/70 border-y border-border/70">
                  {section.params.map((param) => (
                    <EngineParameterDetail key={param.key} param={param} />
                  ))}
                </div>
              </section>
            ))}
          </div>
        )}
      </DetailDrawerTabsContent>
    </DetailDrawerTabs>
  )
}

function EngineCatalogDetailLoadingState() {
  return (
    <div
      {...getLoadingOwnerAttributes({ owner: "engine-catalog-detail", layer: "interaction", intent: "interaction" })}
      data-loading-phase="loading"
      aria-busy="true"
      className="min-h-0 flex-1 overflow-y-auto px-6 py-5"
    >
      <div className="space-y-6">
        <Skeleton className="h-9 w-48 radius-control" />
        <div className="grid gap-4 sm:grid-cols-2">
          {Array.from({ length: 4 }, (_, index) => (
            <div key={index} className="space-y-2">
              <Skeleton className="h-4 w-24 radius-control" />
              <Skeleton className="h-5 w-full radius-control" />
            </div>
          ))}
        </div>
        <div className="space-y-3 border-y border-border/70 py-5">
          <Skeleton className="h-4 w-28 radius-control" />
          <Skeleton className="h-16 w-full radius-control" />
          <Skeleton className="h-16 w-full radius-control" />
        </div>
      </div>
    </div>
  )
}

function EngineDrawerSection({ label, children }: React.PropsWithChildren<{ label: string }>) {
  return (
    <section className="min-w-0 space-y-4 border-b border-border/70 pb-6 last:border-b-0 last:pb-0">
      <h3 className={textRole.sectionTitle}>{label}</h3>
      {children}
    </section>
  )
}

function EngineDetailField({
  label,
  value,
  mono = false,
  className,
}: {
  label: string
  value: string
  mono?: boolean
  className?: string
}) {
  return (
    <div className={cn("min-w-0 space-y-1", className)}>
      <dt className={textRole.metadataLabel}>{label}</dt>
      <dd className={cn("break-words", textRole.metadataValueStrong, mono && textRole.code)}>{value}</dd>
    </div>
  )
}

function EngineBadgeField({
  label,
  values,
  emptyLabel,
  formatValue = (value) => value,
}: {
  label: string
  values: string[]
  emptyLabel: string
  formatValue?: (value: string) => string
}) {
  return (
    <div className="grid min-w-0 grid-cols-1 gap-2 sm:grid-cols-[8rem_minmax(0,1fr)] sm:items-start">
      <div className={textRole.metadataLabel}>{label}</div>
      <div className="flex min-w-0 flex-wrap gap-1.5">
        {values.length === 0 ? (
          <span className={textRole.metadataValue}>{emptyLabel}</span>
        ) : (
          values.map((value) => <Badge key={value} variant="outline">{formatValue(value)}</Badge>)
        )}
      </div>
    </div>
  )
}

function EngineCopyField({ label, value, toastId }: { label: string; value: string; toastId: string }) {
  const tActions = useTranslations("common.actions")
  const tTooltips = useTranslations("tooltips")

  return (
    <div className="min-w-0 space-y-1.5">
      <div className={textRole.metadataLabel}>{label}</div>
      <div className="flex min-w-0 items-start gap-2 border-y border-border/70 py-2">
        <code className={cn("min-w-0 flex-1 break-all", textRole.code)}>{value}</code>
        <CopyButton
          value={value}
          copyLabel={tActions("copy")}
          copiedLabel={tTooltips("copied")}
          toastId={toastId}
        />
      </div>
    </div>
  )
}

function EngineParameterDetail({ param }: { param: EngineParamDefinition }) {
  const t = useTranslations("pages.engines")
  const range = formatParameterRange(param)

  return (
    <div className="min-w-0 space-y-3 py-4">
      <div className="flex min-w-0 items-start justify-between gap-3">
        <div className="min-w-0 space-y-1">
          <div className={textRole.monoLabel}>{param.key}</div>
          {param.description ? <p className={textRole.bodySubtle}>{param.description}</p> : null}
        </div>
        <Badge variant="outline">{param.type}</Badge>
      </div>
      <dl className="grid grid-cols-1 gap-x-6 gap-y-3 sm:grid-cols-2">
        <ParameterMetadataField label={t("defaultValue")} value={formatParameterValue(param.default, t("none"))} />
        {range ? <ParameterMetadataField label={t("allowedRange")} value={range} /> : null}
        {param.enum?.length ? <ParameterMetadataField label={t("allowedValues")} value={param.enum.join(", ")} /> : null}
        {param.resource ? <ParameterMetadataField label={t("resourceBinding")} value={param.resource.kind} /> : null}
      </dl>
    </div>
  )
}

function ParameterMetadataField({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0 space-y-1">
      <dt className={textRole.helperText}>{label}</dt>
      <dd className={cn("break-words", textRole.metadataValue)}>{value}</dd>
    </div>
  )
}

function formatParameterValue(value: EngineParamValue | undefined, emptyLabel: string): string {
  if (value === undefined) return emptyLabel
  if (Array.isArray(value)) return value.length > 0 ? value.join(", ") : emptyLabel
  return String(value)
}

function formatParameterRange(param: EngineParamDefinition): string | null {
  const minimum = param.minimum ?? param.minLength
  const maximum = param.maximum ?? param.maxLength
  if (minimum === undefined && maximum === undefined) return null
  if (minimum === undefined) return `<= ${maximum}`
  if (maximum === undefined) return `>= ${minimum}`
  return `${minimum} - ${maximum}`
}
