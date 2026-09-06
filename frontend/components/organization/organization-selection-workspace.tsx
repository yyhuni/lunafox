"use client"

import React from "react"
import { useLocale, useTranslations } from "next-intl"

import { IconChevronLeft, IconChevronsLeft, IconChevronRight, semanticIcons } from "@/components/icons"
import { SearchInput } from "@/components/shared/search-input"
import { Spinner } from "@/components/shared/loading/spinner"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import type { Organization } from "@/types/organization.types"

export type OrganizationSelectionMode = "single" | "multiple"

export interface OrganizationSelectionWorkspaceProps {
  id: string
  label?: string
  title: string
  hint: string
  selectionMode: OrganizationSelectionMode
  organizations: Organization[]
  selectedOrganizationIds: readonly string[]
  totalCount: number
  canFirstPage: boolean
  canPreviousPage: boolean
  canNextPage: boolean
  pageSize: number
  pageSizeOptions: readonly number[]
  searchQuery: string
  isLoading: boolean
  disabled?: boolean
  showSelectionCount?: boolean
  emptyAction?: React.ReactNode
  className?: string
  onSearchQueryChange: (value: string) => void
  onFirstPage: () => void
  onPreviousPage: () => void
  onNextPage: () => void
  onPageSizeChange: (pageSize: number) => void
  onToggleOrganization: (organization: Organization) => void
  onClearOrganizations: () => void
}

type OrganizationScope = "all" | "selected"

const OrganizationIcon = semanticIcons.concept.organization

function getOrganizationTargetCount(organization: Organization) {
  return organization.stats?.totalTargets
    ?? organization.targetCount
    ?? organization.targets?.length
    ?? 0
}

function getOrganizationInitials(name: string) {
  const firstCharacter = Array.from(name.trim())[0]
  return firstCharacter ? firstCharacter.toUpperCase() : ""
}

function formatOrganizationCreatedAt(value: string, locale: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value

  return date.toLocaleString(locale === "zh" ? "zh-CN" : "en-US", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  })
}

function OrganizationAvatar({ name, isSelected }: { name: string; isSelected: boolean }) {
  return (
    <Avatar className={cn(
      "radius-surface border border-transparent transition-colors group-hover:border-border",
      isSelected && "border-border"
    )}>
      <AvatarFallback className={cn(
        "radius-surface bg-muted font-medium text-muted-foreground transition-colors group-hover:text-foreground",
        isSelected && "text-foreground"
      )}>
        {getOrganizationInitials(name)}
      </AvatarFallback>
    </Avatar>
  )
}

interface OrganizationWorkspacePaginationProps {
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

function OrganizationWorkspacePagination({
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
}: OrganizationWorkspacePaginationProps) {
  const t = useTranslations("organizationSelection")
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
          <SelectTrigger size="sm" className="w-24">
            <SelectValue />
          </SelectTrigger>
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

export function OrganizationSelectionWorkspace({
  id,
  label,
  title,
  hint,
  selectionMode,
  organizations,
  selectedOrganizationIds,
  totalCount,
  canFirstPage,
  canPreviousPage,
  canNextPage,
  pageSize,
  pageSizeOptions,
  searchQuery,
  isLoading,
  disabled = false,
  showSelectionCount = true,
  emptyAction,
  className,
  onSearchQueryChange,
  onFirstPage,
  onPreviousPage,
  onNextPage,
  onPageSizeChange,
  onToggleOrganization,
  onClearOrganizations,
}: OrganizationSelectionWorkspaceProps) {
  const t = useTranslations("organizationSelection")
  const locale = useLocale()
  const [scope, setScope] = React.useState<OrganizationScope>("all")
  const selectedIdSet = React.useMemo(
    () => new Set(selectedOrganizationIds),
    [selectedOrganizationIds]
  )

  if (selectionMode === "single" && selectedOrganizationIds.length > 1) {
    throw new Error("OrganizationSelectionWorkspace single mode accepts at most one selected organization.")
  }
  if (pageSizeOptions.length === 0 || pageSizeOptions.some((size) => !Number.isFinite(size) || size <= 0)) {
    throw new Error("OrganizationSelectionWorkspace requires positive page-size options.")
  }

  const visibleOrganizations = scope === "all"
    ? organizations
    : organizations.filter((organization) => selectedIdSet.has(String(organization.id)))
  const titleId = `${id}-title`
  const labelId = `${id}-label`

  const handleSearchChange = React.useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
    onSearchQueryChange(event.target.value)
  }, [onSearchQueryChange])

  return (
    <section className={cn("grid gap-3", className)} aria-labelledby={label ? labelId : titleId} data-selection-mode={selectionMode}>
      {label ? <Label id={labelId}>{label}</Label> : null}

      <div className="radius-surface overflow-hidden border bg-card">
        <div className="grid gap-3 border-b p-4 sm:min-h-18 sm:flex sm:min-w-0 sm:items-center sm:justify-between sm:gap-4">
          <div className="flex min-w-0 items-center gap-3">
            <span className="radius-surface flex size-8 shrink-0 items-center justify-center bg-muted text-muted-foreground">
              <OrganizationIcon className="size-4" />
            </span>
            <div className="grid min-w-0 gap-0.5">
              <p id={titleId} className={textRole.sectionTitle}>{title}</p>
              <p className={textRole.helperText}>{hint}</p>
            </div>
          </div>
          <div className="flex items-center justify-end gap-2 sm:shrink-0">
            {showSelectionCount ? (
              <Badge variant="count" className="px-2 py-0.5">{t("selectedCount", { count: selectedOrganizationIds.length })}</Badge>
            ) : null}
            <Button type="button" variant="link" size="sm" onClick={onClearOrganizations} disabled={disabled || selectedOrganizationIds.length === 0}>
              {t("clear")}
            </Button>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2 border-b p-3">
          <div className="min-w-48 flex-1">
            <SearchInput
              value={searchQuery}
              onChange={handleSearchChange}
              placeholder={t("searchPlaceholder")}
              disabled={disabled}
              toolbarDensity="compact"
              className="min-w-0"
            />
          </div>
          <Select value={scope} onValueChange={(value) => {
            if (value === "all" || value === "selected") setScope(value)
          }} disabled={disabled}>
            <SelectTrigger size="sm" className="w-28"><SelectValue>{scope === "all" ? t("scopeAll") : t("scopeSelected")}</SelectValue></SelectTrigger>
            <SelectContent width="content-fit">
              <SelectItem value="all">{t("scopeAll")}</SelectItem>
              <SelectItem value="selected">{t("scopeSelected")}</SelectItem>
            </SelectContent>
          </Select>
        </div>

        {isLoading ? (
          <div className="flex h-72 items-center justify-center"><Spinner /></div>
        ) : visibleOrganizations.length === 0 ? (
          <div className="p-4">
            <div className="radius-surface grid justify-items-center gap-3 bg-muted/10 px-4 py-8 text-center">
              <span className="radius-round flex size-12 items-center justify-center border bg-background text-muted-foreground"><OrganizationIcon className="size-5" /></span>
              <div className="grid gap-1">
                <p className={textRole.bodyStrong}>{scope === "selected" ? t("noSelected") : t("noAvailable")}</p>
                <p className={cn("max-w-72", textRole.helperText)}>{scope === "selected" ? t("selectedEmptyHint") : t("emptyHint")}</p>
              </div>
              {scope === "all" ? emptyAction : null}
            </div>
          </div>
        ) : (
          <div className="grid max-h-90 grid-cols-1 gap-2 overflow-y-auto p-3 sm:grid-cols-2">
            {visibleOrganizations.map((organization) => {
              const organizationId = String(organization.id)
              const isSelected = selectedIdSet.has(organizationId)
              return (
                <label key={organization.id} className={cn("group radius-surface flex cursor-pointer items-start gap-3 border p-3", isSelected && "border-interaction-accent bg-primary/5")}>
                  <Checkbox className="mt-2" checked={isSelected} onCheckedChange={() => onToggleOrganization(organization)} aria-label={t("selectOrganization", { name: organization.name })} disabled={disabled} />
                  <OrganizationAvatar name={organization.name} isSelected={isSelected} />
                  <span className="grid min-w-0 gap-1">
                    <span className={textRole.tableCellPrimary}>{organization.name}</span>
                    {organization.description ? <span className={cn("line-clamp-2", textRole.tableCellSecondary)}>{organization.description}</span> : null}
                    <span className={textRole.metadataLabel}>{t("targetCount", { count: getOrganizationTargetCount(organization) })} · {formatOrganizationCreatedAt(organization.createdAt, locale)}</span>
                  </span>
                </label>
              )
            })}
          </div>
        )}

        <div className="border-t bg-muted/30 sm:h-18">
          <OrganizationWorkspacePagination
            totalCount={totalCount}
            canFirstPage={canFirstPage}
            canPreviousPage={canPreviousPage}
            canNextPage={canNextPage}
            pageSize={pageSize}
            pageSizeOptions={pageSizeOptions}
            disabled={disabled || isLoading}
            onFirstPage={onFirstPage}
            onPreviousPage={onPreviousPage}
            onNextPage={onNextPage}
            onPageSizeChange={onPageSizeChange}
          />
        </div>
      </div>
    </section>
  )
}
