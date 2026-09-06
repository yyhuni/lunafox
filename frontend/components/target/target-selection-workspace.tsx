"use client"

import React from "react"
import { useTranslations } from "next-intl"

import { IconChevronLeft, IconChevronsLeft, IconChevronRight, semanticIcons } from "@/components/icons"
import { SearchInput } from "@/components/shared/search-input"
import { Spinner } from "@/components/shared/loading/spinner"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import type { Target } from "@/types/target.types"

export interface TargetSelectionWorkspaceProps {
  id: string
  title: string
  hint: string
  targets: Target[]
  selectedTargetId: number | null
  totalCount: number
  canFirstPage: boolean
  canPreviousPage: boolean
  canNextPage: boolean
  pageSize: number
  pageSizeOptions: readonly number[]
  searchQuery: string
  isLoading: boolean
  disabled?: boolean
  onSearchQueryChange: (value: string) => void
  onFirstPage: () => void
  onPreviousPage: () => void
  onNextPage: () => void
  onPageSizeChange: (pageSize: number) => void
  onToggleTarget: (target: Target) => void
  onClearTarget: () => void
}

type TargetScope = "all" | "selected"

const TargetIcon = semanticIcons.concept.target

interface TargetWorkspacePaginationProps {
  totalCount: number
  canFirstPage: boolean
  canPreviousPage: boolean
  canNextPage: boolean
  pageSize: number
  pageSizeOptions: readonly number[]
  disabled: boolean
  onFirstPage: () => void
  onPreviousPage: () => void
  onNextPage: () => void
  onPageSizeChange: (pageSize: number) => void
}

function TargetWorkspacePagination({
  totalCount,
  canFirstPage,
  canPreviousPage,
  canNextPage,
  pageSize,
  pageSizeOptions,
  disabled,
  onFirstPage,
  onPreviousPage,
  onNextPage,
  onPageSizeChange,
}: TargetWorkspacePaginationProps) {
  const t = useTranslations("targetSelection")
  return (
    <div className="flex flex-col gap-2 p-4 sm:h-full sm:flex-row sm:items-center">
      <span className={cn("sm:flex-1", textRole.metadataLabel)}>
        {t("total", { count: totalCount })}
      </span>
      <nav className="flex items-center justify-center gap-1" aria-label={t("paginationLabel")}>
        <Button type="button" variant="outline" size="icon-sm" aria-label={t("firstPage")} onClick={onFirstPage} disabled={disabled || !canFirstPage}>
          <IconChevronsLeft />
        </Button>
        <Button type="button" variant="outline" size="icon-sm" aria-label={t("previousPage")} onClick={onPreviousPage} disabled={disabled || !canPreviousPage}>
          <IconChevronLeft />
        </Button>
        <Button type="button" variant="outline" size="icon-sm" aria-label={t("nextPage")} onClick={onNextPage} disabled={disabled || !canNextPage}>
          <IconChevronRight />
        </Button>
      </nav>
      <div className="flex items-center justify-end gap-2 sm:flex-1">
        <Label className={textRole.metadataLabel}>{t("perPage")}</Label>
        <Select value={String(pageSize)} onValueChange={(value) => onPageSizeChange(Number(value))} disabled={disabled}>
          <SelectTrigger size="sm" className="w-24"><SelectValue /></SelectTrigger>
          <SelectContent width="content-fit">
            {pageSizeOptions.map((size) => (
              <SelectItem key={size} value={String(size)}>{t("perPageOption", { count: size })}</SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </div>
  )
}

export function TargetSelectionWorkspace({
  id,
  title,
  hint,
  targets,
  selectedTargetId,
  totalCount,
  canFirstPage,
  canPreviousPage,
  canNextPage,
  pageSize,
  pageSizeOptions,
  searchQuery,
  isLoading,
  disabled = false,
  onSearchQueryChange,
  onFirstPage,
  onPreviousPage,
  onNextPage,
  onPageSizeChange,
  onToggleTarget,
  onClearTarget,
}: TargetSelectionWorkspaceProps) {
  const t = useTranslations("targetSelection")
  const tTarget = useTranslations("target")
  const [scope, setScope] = React.useState<TargetScope>("all")

  if (pageSizeOptions.length === 0 || pageSizeOptions.some((size) => !Number.isFinite(size) || size <= 0)) {
    throw new Error("TargetSelectionWorkspace requires positive page-size options.")
  }

  const visibleTargets = scope === "all"
    ? targets
    : targets.filter((target) => target.id === selectedTargetId)
  const targetTypeLabels = {
    domain: tTarget("types.domain"),
    ip: tTarget("types.ip"),
    cidr: tTarget("types.cidr"),
  } as const
  const titleId = `${id}-title`

  const handleSearchChange = React.useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
    onSearchQueryChange(event.target.value)
  }, [onSearchQueryChange])

  return (
    <section className="grid gap-3" aria-labelledby={titleId} data-selection-mode="single" data-selection-entity="target">
      <div className="radius-surface overflow-hidden border bg-card">
        <div className="grid gap-3 border-b p-4 sm:min-h-18 sm:flex sm:min-w-0 sm:items-center sm:justify-between sm:gap-4">
          <div className="flex min-w-0 items-center gap-3">
            <span className="radius-surface flex size-8 shrink-0 items-center justify-center bg-muted text-muted-foreground">
              <TargetIcon className="size-4" />
            </span>
            <div className="grid min-w-0 gap-0.5">
              <p id={titleId} className={textRole.sectionTitle}>{title}</p>
              <p className={textRole.helperText}>{hint}</p>
            </div>
          </div>
          <div className="flex items-center justify-end gap-2 sm:shrink-0">
            <Button type="button" variant="link" size="sm" onClick={onClearTarget} disabled={disabled || selectedTargetId === null}>
              {t("clear")}
            </Button>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2 border-b p-3">
          <div className="min-w-48 flex-1">
            <SearchInput value={searchQuery} onChange={handleSearchChange} placeholder={t("searchPlaceholder")} disabled={disabled} toolbarDensity="compact" className="min-w-0" />
          </div>
          <Select value={scope} onValueChange={(value) => {
            if (value === "all" || value === "selected") setScope(value)
          }} disabled={disabled}>
            <SelectTrigger size="sm" className="w-32"><SelectValue>{scope === "all" ? t("scopeAll") : t("scopeSelected")}</SelectValue></SelectTrigger>
            <SelectContent width="content-fit">
              <SelectItem value="all">{t("scopeAll")}</SelectItem>
              <SelectItem value="selected">{t("scopeSelected")}</SelectItem>
            </SelectContent>
          </Select>
        </div>

        {isLoading ? (
          <div className="flex h-72 items-center justify-center"><Spinner /></div>
        ) : visibleTargets.length === 0 ? (
          <div className="p-4">
            <div className="radius-surface grid justify-items-center gap-3 bg-muted/10 px-4 py-8 text-center">
              <span className="radius-round flex size-12 items-center justify-center border bg-background text-muted-foreground"><TargetIcon className="size-5" /></span>
              <div className="grid gap-1">
                <p className={textRole.bodyStrong}>{scope === "selected" ? t("noSelected") : t("noAvailable")}</p>
                <p className={cn("max-w-72", textRole.helperText)}>{scope === "selected" ? t("selectedEmptyHint") : t("emptyHint")}</p>
              </div>
            </div>
          </div>
        ) : (
          <div className="grid max-h-[360px] grid-cols-1 gap-2 overflow-y-auto p-3 sm:grid-cols-2">
            {visibleTargets.map((target) => {
              const isSelected = selectedTargetId === target.id
              const organizationNames = target.organizations?.map((organization) => organization.name).join(", ")
              const TargetTypeIcon = semanticIcons.concept[target.type]
              return (
                <label key={target.id} className={cn("radius-surface flex cursor-pointer items-start gap-3 border p-3", isSelected && "border-interaction-accent bg-primary/5")}>
                  <Checkbox className="mt-2" checked={isSelected} onCheckedChange={() => onToggleTarget(target)} aria-label={t("selectTarget", { name: target.name })} disabled={disabled} />
                  <span title={targetTypeLabels[target.type]} className="radius-surface flex size-8 shrink-0 items-center justify-center bg-muted text-muted-foreground"><TargetTypeIcon aria-hidden="true" className="size-4" /></span>
                  <span className="grid min-w-0 gap-1">
                    <span className={textRole.tableCellPrimary}>{target.name}</span>
                    <span className={cn("line-clamp-2", textRole.tableCellSecondary)}>{organizationNames || t("unassociated")}</span>
                    <span className={textRole.metadataLabel}>{targetTypeLabels[target.type]}</span>
                  </span>
                </label>
              )
            })}
          </div>
        )}

        <div className="border-t bg-muted/30 sm:h-18">
          <TargetWorkspacePagination totalCount={totalCount} canFirstPage={canFirstPage} canPreviousPage={canPreviousPage} canNextPage={canNextPage} pageSize={pageSize} pageSizeOptions={pageSizeOptions} disabled={disabled || isLoading} onFirstPage={onFirstPage} onPreviousPage={onPreviousPage} onNextPage={onNextPage} onPageSizeChange={onPageSizeChange} />
        </div>
      </div>
    </section>
  )
}
