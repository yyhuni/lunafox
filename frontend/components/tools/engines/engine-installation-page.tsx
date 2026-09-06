"use client"

import * as React from "react"
import { useLocale, useTranslations } from "next-intl"

import { Button } from "@/components/ui/button"
import { IconPlus } from "@/components/icons"
import { Input } from "@/components/ui/input"
import { AlertDialog, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { PageHeader } from "@/components/common/page-header"
import { AppErrorState } from "@/components/shared/feedback/app-error-state"
import { FormDrawer } from "@/components/shared/form-drawer"
import { ContentHandoff } from "@/components/shared/loading/content-handoff"
import { getLoadingStructureSlotAttributes } from "@/components/shared/loading/loading-owner"
import { SearchInput } from "@/components/shared/search-input"
import {
  EngineCatalogCard,
  EngineCatalogCardLoadingState,
} from "@/components/tools/engines/engine-catalog-card"
import { EngineCatalogDetailDrawer } from "@/components/tools/engines/engine-catalog-detail-drawer"
import { useEngineCatalog, useInstallEngine } from "@/hooks/use-engine-catalog"
import { getEngineReplacementConflict, getLocalizedEngineSummaryName, type EngineReplacementConflict } from "@/lib/engine-catalog"
import { normalizeError } from "@/lib/errors/normalize-error"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import type { Locale } from "@/i18n/config"
import type { EngineCatalogSummary } from "@/types/engine-catalog.types"

const ENGINE_CATALOG_GRID_CLASS =
  "grid grid-cols-1 gap-4 @xl/main:grid-cols-2 @5xl/main:grid-cols-3 @7xl/main:grid-cols-4"

// The built-in catalog has a stable initial window of eight entries. The
// loading grid reserves that real first-frame card rhythm instead of showing an
// extra placeholder row that disappears when the catalog resolves.
const ENGINE_CATALOG_FIRST_FRAME_CARD_COUNT = 8

function EngineCatalogControls({
  value,
  onChange,
  placeholder,
  disabled = false,
  action,
}: {
  value: string
  onChange: (event: React.ChangeEvent<HTMLInputElement>) => void
  placeholder: string
  disabled?: boolean
  action?: React.ReactNode
}) {
  return (
    <div
      {...getLoadingStructureSlotAttributes("engine-catalog-controls")}
      className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between"
    >
      <div className="min-w-0 w-full sm:max-w-sm">
        <SearchInput
          value={value}
          onChange={onChange}
          placeholder={placeholder}
          disabled={disabled}
          toolbarDensity="compact"
        />
      </div>
      {action ? <div className="flex shrink-0 items-center">{action}</div> : null}
    </div>
  )
}

function EngineCatalogResults({
  children,
  error = false,
}: {
  children: React.ReactNode
  error?: boolean
}) {
  const t = useTranslations("pages.engines")

  return (
    <section
      {...getLoadingStructureSlotAttributes("engine-catalog-grid")}
      aria-label={t("title")}
    >
      <div
        {...getLoadingStructureSlotAttributes("engine-catalog-card-rhythm")}
        className={cn(error ? "min-h-0" : ENGINE_CATALOG_GRID_CLASS)}
      >
        {children}
      </div>
    </section>
  )
}

function EngineCatalogGridLoadingState() {
  return (
    <EngineCatalogResults>
      {Array.from({ length: ENGINE_CATALOG_FIRST_FRAME_CARD_COUNT }, (_, index) => (
        <EngineCatalogCardLoadingState key={index} />
      ))}
    </EngineCatalogResults>
  )
}

function EngineCatalogLoadingState({
  searchPlaceholder,
  installLabel,
}: {
  searchPlaceholder: string
  installLabel: string
}) {
  return (
    <>
      <EngineCatalogControls
        value=""
        onChange={() => {}}
        placeholder={searchPlaceholder}
        disabled
        action={<Button type="button" size="sm" disabled><IconPlus className="size-4" />{installLabel}</Button>}
      />
      <EngineCatalogGridLoadingState />
    </>
  )
}

export function EngineInstallationPage({ embedded = false }: { embedded?: boolean }) {
  const [search, setSearch] = React.useState("")
  const [artifactRef, setArtifactRef] = React.useState("")
  const [isInstallOpen, setIsInstallOpen] = React.useState(false)
  const [selectedEngine, setSelectedEngine] = React.useState<EngineCatalogSummary | null>(null)
  const [conflict, setConflict] = React.useState<EngineReplacementConflict | null>(null)
  const locale = useLocale() as Locale
  const t = useTranslations("pages.engines")
  const tActions = useTranslations("common.actions")
  const catalog = useEngineCatalog()
  const install = useInstallEngine()
  const submitInstall = async (allowReplacement: boolean) => {
    try {
      await install.mutateAsync({ artifactRef, allowReplacement })
      setConflict(null)
      setArtifactRef("")
      setIsInstallOpen(false)
    } catch (error: unknown) {
      const replacementConflict = getEngineReplacementConflict(error)
      if (replacementConflict) setConflict(replacementConflict)
    }
  }
  const engines = React.useMemo(() => {
    const query = search.trim().toLocaleLowerCase()
    return (catalog.data ?? []).filter((engine) => !query || [engine.engineId, engine.publisher, getLocalizedEngineSummaryName(engine, locale), ...engine.execution.supportedTargetTypes].join(" ").toLocaleLowerCase().includes(query))
  }, [catalog.data, locale, search])

  const content = (
    <>
      <div className="px-4 lg:px-6">
        <ContentHandoff
          owner="engine-catalog-content"
          layer="workspace"
          isLoading={catalog.isPending}
          skeleton={<EngineCatalogLoadingState searchPlaceholder={t("searchPlaceholder")} installLabel={t("install")} />}
          skeletonClassName="space-y-6"
          contentClassName="space-y-6"
        >
          <EngineCatalogControls
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder={t("searchPlaceholder")}
            action={
              <FormDrawer
                open={isInstallOpen}
                onOpenChange={setIsInstallOpen}
                trigger={<Button type="button" size="sm"><IconPlus className="size-4" />{t("install")}</Button>}
                title={t("install")}
                description={t("installDescription")}
                closeDisabled={install.isPending}
                formProps={{
                  onSubmit: (event) => {
                    event.preventDefault()
                    void submitInstall(false)
                  },
                }}
                footer={<Button type="submit" disabled={install.isPending || !artifactRef}>{t("install")}</Button>}
              >
                <label className="grid gap-2" htmlFor="engine-artifact-ref">
                  <span className={textRole.metadataLabel}>{t("artifactRef")}</span>
                  <Input
                    id="engine-artifact-ref"
                    value={artifactRef}
                    onChange={(event) => setArtifactRef(event.target.value)}
                    placeholder="registry.example/team/engine@sha256:..."
                    disabled={install.isPending}
                  />
                </label>
              </FormDrawer>
            }
          />
          {catalog.isError ? (
            <EngineCatalogResults error>
              <AppErrorState
                error={normalizeError(catalog.error, { notFoundKind: "unexpected-error" })}
                title={t("catalogUnavailable")}
                onRetry={catalog.refetch}
                variant="section"
              />
            </EngineCatalogResults>
          ) : (
            <EngineCatalogResults>
              {engines.map((engine) => (
                <EngineCatalogCard
                  key={engine.engineId}
                  engine={engine}
                  locale={locale}
                  onSelect={setSelectedEngine}
                />
              ))}
            </EngineCatalogResults>
          )}
        </ContentHandoff>
      </div>
      <EngineCatalogDetailDrawer
        engine={selectedEngine}
        locale={locale}
        open={selectedEngine !== null}
        onOpenChange={(open) => {
          if (!open) setSelectedEngine(null)
        }}
      />
      <AlertDialog open={conflict !== null} onOpenChange={(open) => !open && setConflict(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("replaceTitle")}</AlertDialogTitle>
            <AlertDialogDescription>{t("replaceDescription")}</AlertDialogDescription>
          </AlertDialogHeader>
          {conflict ? (
            <dl className="grid gap-3 font-mono text-xs">
              <div><dt className={textRole.helperText}>{t("engineId")}</dt><dd className="break-all">{conflict.engineId}</dd></div>
              <div><dt className={textRole.helperText}>{t("currentPackageDigest")}</dt><dd className="break-all">{conflict.currentPackageDigest}</dd></div>
              <div><dt className={textRole.helperText}>{t("proposedPackageDigest")}</dt><dd className="break-all">{conflict.proposedPackageDigest}</dd></div>
            </dl>
          ) : null}
          <AlertDialogFooter>
            <Button variant="outline" onClick={() => setConflict(null)} disabled={install.isPending}>{tActions("cancel")}</Button>
            <Button onClick={() => void submitInstall(true)} disabled={install.isPending}>{t("replace")}</Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )

  if (embedded) {
    return content
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-4 py-4 md:gap-6 md:py-6">
      <PageHeader code="TLS-04" title={t("title")} description={t("description")} />
      {content}
    </div>
  )
}
