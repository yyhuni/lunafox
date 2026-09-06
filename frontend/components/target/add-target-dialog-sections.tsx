import Link from "next/link"
import React from "react"
import { ChevronRight, Plus } from "@/components/icons"
import { BulkLineValidationInput, type BulkLineValidationIssue } from "@/components/common/bulk-line-validation-input"
import {
  countLineNumberedTextareaLines,
  lineNumberedTextareaResponsiveViewportClassName,
} from "@/components/common/line-numbered-textarea"
import { OrganizationSelectionWorkspace } from "@/components/organization/organization-selection-workspace"
import { Button } from "@/components/ui/button"
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import type { CursorPaginationNavigation } from "@/types/data-table.types"
import type { Organization } from "@/types/organization.types"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

const ORGANIZATION_WORKSPACE_PAGE_SIZE_OPTIONS = [10, 20, 50]

interface InvalidTargetInfo {
  index: number
  lineNumber: number
  originalTarget: string
  error: string
}

interface AddTargetInputSectionProps {
  t: TranslationFn
  formTargets: string
  targetCount: number
  invalidTargets: InvalidTargetInfo[]
  lineNumbersRef: React.RefObject<HTMLDivElement | null>
  textareaRef: React.RefObject<HTMLTextAreaElement | null>
  onInputChange: (value: string) => void
  onScroll: (event: React.UIEvent<HTMLTextAreaElement>) => void
  isPending: boolean
}

export function AddTargetInputSection({
  t,
  formTargets,
  targetCount,
  invalidTargets,
  lineNumbersRef,
  textareaRef,
  onInputChange,
  onScroll,
  isPending,
}: AddTargetInputSectionProps) {
  const lineIssues = React.useMemo<BulkLineValidationIssue[]>(() => invalidTargets.map((target) => ({
    id: `invalid-${target.index}`,
    lineNumber: target.lineNumber,
    tone: "error",
    badgeLabel: t("invalidBadge"),
    message: t("invalidIssue", {
      line: target.lineNumber,
      target: target.originalTarget,
      error: target.error,
    }),
  })), [invalidTargets, t])

  const validationResult = formTargets.trim().length > 0
    ? {
      validCount: targetCount,
      blockingIssueCount: invalidTargets.length,
      advisoryIssueCount: 0,
      lineIssues,
    }
    : null

  return (
    <section className="grid gap-3" aria-labelledby="targets">
      <BulkLineValidationInput
        id="targets"
        name="targets"
        label={t("targetList")}
        required
        placeholder={t("targetPlaceholder")}
        value={formTargets}
        lineCount={Math.max(countLineNumberedTextareaLines(formTargets), 8)}
        lineNumbersRef={lineNumbersRef}
        textareaRef={textareaRef}
        onValueChange={onInputChange}
        onScroll={onScroll}
        viewportClassName={lineNumberedTextareaResponsiveViewportClassName}
        disabled={isPending}
        helper={t("targetHelper")}
        example={t("targetExample")}
        emptySummary={t("emptySummary")}
        validSummary={t("validSummary", { count: targetCount })}
        blockingSummary={t("blockingSummary", { count: invalidTargets.length })}
        collapseDetails={t("collapseDetails")}
        expandDetails={t("expandDetails")}
        validationResult={validationResult}
      />
    </section>
  )
}

interface AddTargetOrganizationPickerProps {
  t: TranslationFn
  isLoading: boolean
  formOrganizationIds: string[]
  organizations: Organization[]
  organizationTotalCount: number
  organizationPaginationNavigation: CursorPaginationNavigation
  onFirstOrgPage: () => void
  orgSearchQuery: string
  setOrgSearchQuery: (value: string) => void
  orgPageSize: number
  setOrgPageSize: React.Dispatch<React.SetStateAction<number>>
  onPreviousOrgPage: () => void
  onNextOrgPage: () => void
  onToggleOrganization: (id: string) => void
  onClearOrganizations: () => void
  isPending: boolean
}

export function AddTargetOrganizationPicker({
  t,
  isLoading,
  formOrganizationIds,
  organizations,
  organizationTotalCount,
  organizationPaginationNavigation,
  onFirstOrgPage,
  orgSearchQuery,
  setOrgSearchQuery,
  orgPageSize,
  setOrgPageSize,
  onPreviousOrgPage,
  onNextOrgPage,
  onToggleOrganization,
  onClearOrganizations,
  isPending,
}: AddTargetOrganizationPickerProps) {
  return (
    <Collapsible defaultOpen={false} className="border-t border-border/60 pt-4">
      <CollapsibleTrigger
        render={(
          <Button
            type="button"
            variant="ghost"
            className="group h-auto min-w-0 justify-start px-0 py-0 text-left hover:bg-transparent hover:text-foreground dark:hover:bg-transparent dark:hover:text-foreground"
            disabled={isPending}
          />
        )}
      >
        <span className="flex min-w-0 items-center gap-2">
          <span className={cn(textRole.bodyStrong, "min-w-0 text-muted-foreground transition-colors group-hover:text-foreground")}>
            {t("linkOrganization")}
          </span>
          <ChevronRight className="size-4 shrink-0 text-muted-foreground transition-[color,transform] duration-200 motion-reduce:transition-none group-hover:text-foreground group-data-[panel-open]:rotate-90" />
        </span>
      </CollapsibleTrigger>
      <CollapsibleContent className="pt-4">
        <OrganizationSelectionWorkspace
          id="add-target-organization-workspace"
          title={t("organizationWorkspaceTitle")}
          hint={t("organizationWorkspaceHint")}
          selectionMode="multiple"
          organizations={organizations}
          selectedOrganizationIds={formOrganizationIds}
          totalCount={organizationTotalCount}
          canFirstPage={organizationPaginationNavigation.canFirstPage ?? false}
          canPreviousPage={organizationPaginationNavigation.canPreviousPage}
          canNextPage={organizationPaginationNavigation.canNextPage}
          pageSize={orgPageSize}
          pageSizeOptions={ORGANIZATION_WORKSPACE_PAGE_SIZE_OPTIONS}
          searchQuery={orgSearchQuery}
          isLoading={isLoading}
          disabled={isPending}
          emptyAction={<Button type="button" size="sm" variant="outline" render={<Link href="/organizations/" />}><Plus className="size-3.5" />{t("createOrganization")}</Button>}
          onSearchQueryChange={setOrgSearchQuery}
          onFirstPage={onFirstOrgPage}
          onPreviousPage={onPreviousOrgPage}
          onNextPage={onNextOrgPage}
          onPageSizeChange={setOrgPageSize}
          onToggleOrganization={(organization) => onToggleOrganization(String(organization.id))}
          onClearOrganizations={onClearOrganizations}
        />
      </CollapsibleContent>
    </Collapsible>
  )
}

interface AddTargetDialogFooterProps {
  tDialog: TranslationFn
  isPending: boolean
  isFormValid: boolean
}

export function AddTargetDialogFooter({ tDialog, isPending, isFormValid }: AddTargetDialogFooterProps) {
  return (
    <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
      <Button type="submit" disabled={isPending || !isFormValid} loading={isPending} loadingLabel={tDialog("creating")}>
        <Plus />
        {tDialog("addTarget")}
      </Button>
    </div>
  )
}
