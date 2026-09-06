"use client"

import React, { useEffect, useMemo, useRef, useState } from "react"
import { useTranslations } from "next-intl"

import { BulkLineValidationInput, type BulkLineValidationResult } from "@/components/common/bulk-line-validation-input"
import { countLineNumberedTextareaLines } from "@/components/common/line-numbered-textarea"
import { PageHeader } from "@/components/common/page-header"
import {
  BlacklistSettingsLoadingState,
  EXAMPLE_RULES,
} from "@/components/settings/blacklist/blacklist-settings-loading-state"
import { ChevronDown, semanticIcons } from "@/components/icons"
import { AppErrorState } from "@/components/shared/feedback/app-error-state"
import { ContentHandoff } from "@/components/shared/loading/content-handoff"
import { useDetailShellReadySignal } from "@/components/shared/loading/detail-shell-ready-context"
import { getLoadingStructureSlotAttributes } from "@/components/shared/loading/loading-owner"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  useGlobalBlacklistPolicy,
  useTargetBlacklistPolicy,
  useUpdateGlobalBlacklistPolicy,
  useUpdateTargetBlacklistPolicy,
} from "@/hooks/use-blacklist-policy"
import { normalizeError } from "@/lib/errors/normalize-error"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import {
  parseBlacklistRules,
  submittedBlacklistPatterns,
  type BlacklistRuleKind,
  type ParsedBlacklistRule,
} from "./blacklist-rule-parser"
import {
  BLACKLIST_CONTENT_SHELL_CLASS,
  BLACKLIST_EDITOR_ACTION_GROUP_CLASS,
  BLACKLIST_EDITOR_ACTION_ROW_CLASS,
  BLACKLIST_EDITOR_CARD_CLASS,
  BLACKLIST_EDITOR_CONTENT_CLASS,
  BLACKLIST_EDITOR_EXAMPLES_ROW_CLASS,
  BLACKLIST_EDITOR_HEADER_CLASS,
  BLACKLIST_EDITOR_HEADER_ROW_CLASS,
  BLACKLIST_EDITOR_STATUS_SLOT_CLASS,
  BLACKLIST_EDITOR_TITLE_GROUP_CLASS,
  BLACKLIST_EMBEDDED_CONTENT_SHELL_CLASS,
  BLACKLIST_EMBEDDED_PAGE_SHELL_CLASS,
  BLACKLIST_PAGE_SHELL_CLASS,
  BLACKLIST_RULE_GROUP_TABLE_CLASS,
  BLACKLIST_RULE_GROUP_TRIGGER_CLASS,
  BLACKLIST_RULE_ICON_CELL_CLASS,
  BLACKLIST_RULE_LIST_CARD_CLASS,
  BLACKLIST_RULE_LIST_CONTENT_CLASS,
  BLACKLIST_RULE_LIST_HEADER_CLASS,
  BLACKLIST_RULE_ROW_CLASS,
  BLACKLIST_RULE_STATUS_CELL_CLASS,
  BLACKLIST_RULE_VALUE_CELL_CLASS,
  BLACKLIST_WORKBENCH_GRID_CLASS,
} from "./blacklist-settings-layout"

type RuleGroupKey = BlacklistRuleKind | "inherited"

function groupRules(rules: ParsedBlacklistRule[]): Record<BlacklistRuleKind, ParsedBlacklistRule[]> {
  return {
    domain: rules.filter((rule) => rule.kind === "domain"),
    ipv4: rules.filter((rule) => rule.kind === "ipv4"),
    cidr: rules.filter((rule) => rule.kind === "cidr"),
    invalid: rules.filter((rule) => rule.kind === "invalid"),
  }
}

interface BlacklistSettingsWorkspaceProps {
  embedded?: boolean
  targetId?: number
  owner?: string
  pageTitle?: string
  pageDescription?: string
}

export function BlacklistSettingsWorkspace({
  embedded = false,
  targetId,
  owner,
  pageTitle,
  pageDescription,
}: BlacklistSettingsWorkspaceProps) {
  const t = useTranslations("pages.settings.blacklist")
  const isTargetScope = embedded && targetId !== undefined
  const [blacklistText, setBlacklistText] = useState("")
  const [hasChanges, setHasChanges] = useState(false)
  const [collapsedGroups, setCollapsedGroups] = useState<Record<RuleGroupKey, boolean>>({
    domain: false,
    ipv4: false,
    cidr: false,
    invalid: false,
    inherited: true,
  })
  const lineNumbersRef = useRef<HTMLDivElement | null>(null)
  const textareaRef = useRef<HTMLTextAreaElement | null>(null)

  const globalPolicy = useGlobalBlacklistPolicy()
  const targetPolicy = useTargetBlacklistPolicy(targetId ?? 0)
  const updateGlobalPolicy = useUpdateGlobalBlacklistPolicy()
  const updateTargetPolicy = useUpdateTargetBlacklistPolicy()
  const editablePolicy = isTargetScope ? targetPolicy.data : globalPolicy.data
  const inheritedRules = useMemo(
    () => (isTargetScope ? parseBlacklistRules(globalPolicy.data?.patterns.join("\n") ?? "") : []),
    [globalPolicy.data?.patterns, isTargetScope]
  )
  const policyError = globalPolicy.error ?? (isTargetScope ? targetPolicy.error : undefined)
  const isInitialLoading = isTargetScope
    ? (globalPolicy.isLoading && !globalPolicy.data) || (targetPolicy.isLoading && !targetPolicy.data)
    : globalPolicy.isLoading && !globalPolicy.data
  const isSaving = isTargetScope ? updateTargetPolicy.isPending : updateGlobalPolicy.isPending
  const detailShellReady = useDetailShellReadySignal(!isInitialLoading)
  const resolvedOwner = owner ?? (embedded ? "target-settings-content" : "blacklist-page-content")

  useEffect(() => {
    if (editablePolicy) {
      setBlacklistText(editablePolicy.patterns.join("\n"))
      setHasChanges(false)
    }
  }, [editablePolicy])

  const parsedRules = useMemo(() => parseBlacklistRules(blacklistText), [blacklistText])
  const ruleGroups = useMemo(() => groupRules(parsedRules), [parsedRules])
  const errorRules = useMemo(() => parsedRules.filter((rule) => !rule.valid), [parsedRules])
  const errorCount = errorRules.length
  const validCount = parsedRules.length - errorCount
  const lineCount = Math.max(countLineNumberedTextareaLines(blacklistText), 10)
  const validationResult = useMemo<BulkLineValidationResult | null>(() => {
    if (parsedRules.length === 0) return null
    return {
      validCount,
      blockingIssueCount: errorCount,
      advisoryIssueCount: 0,
      lineIssues: errorRules.map((rule) => ({
        id: `blacklist-rule-${rule.line}`,
        lineNumber: rule.line,
        tone: "error" as const,
        badgeLabel: t("editor.invalidBadge"),
        message: t("editor.invalidRule", { line: rule.line }),
      })),
    }
  }, [errorCount, errorRules, parsedRules.length, t, validCount])

  const handleTextChange = (value: string) => {
    setBlacklistText(value)
    setHasChanges(true)
  }

  const handleTextareaScroll = (event: React.UIEvent<HTMLTextAreaElement>) => {
    if (lineNumbersRef.current) {
      lineNumbersRef.current.scrollTop = event.currentTarget.scrollTop
    }
  }

  const toggleRuleGroup = (group: RuleGroupKey) => {
    setCollapsedGroups((current) => ({
      ...current,
      [group]: !current[group],
    }))
  }

  const handleSave = () => {
    if (!editablePolicy?.etag) return
    const input = {
      patterns: submittedBlacklistPatterns(parsedRules),
      etag: editablePolicy.etag,
    }
    const onSuccess = () => setHasChanges(false)

    if (isTargetScope && targetId !== undefined) {
      updateTargetPolicy.mutate({ targetId, ...input }, { onSuccess })
      return
    }
    updateGlobalPolicy.mutate(input, { onSuccess })
  }

  const handleRefetch = () => {
    void globalPolicy.refetch()
    if (isTargetScope) {
      void targetPolicy.refetch()
    }
  }

  if (policyError) {
    return (
      <AppErrorState
        error={normalizeError(policyError, { notFoundKind: "unexpected-error" })}
        onRetry={handleRefetch}
        actionHref={embedded ? undefined : "/overview/"}
        variant={embedded ? "section" : "page"}
        className={embedded ? undefined : "flex-1"}
      />
    )
  }

  if (embedded && detailShellReady?.deferInitialSkeleton && isInitialLoading) {
    return null
  }

  const editorTitle = isTargetScope ? t("editor.targetTitle") : t("editor.title")
  const editorDescription = isTargetScope ? t("editor.targetDescription") : t("editor.description")
  const editorLabel = isTargetScope ? t("editor.targetInputLabel") : t("editor.inputLabel")

  return (
    <ContentHandoff
      owner={resolvedOwner}
      layer={embedded ? "section" : "workspace"}
      isLoading={isInitialLoading}
      skeleton={(
        <BlacklistSettingsLoadingState
          embedded={embedded}
          scope={isTargetScope ? "target" : "global"}
          pageTitle={pageTitle}
          pageDescription={pageDescription}
        />
      )}
      className="flex min-h-0 flex-1 flex-col"
      skeletonClassName="flex min-h-0 flex-1 flex-col"
      contentClassName="flex min-h-0 flex-1 flex-col"
      prepareContentBeforeHandoff
    >
      <div className={embedded ? BLACKLIST_EMBEDDED_PAGE_SHELL_CLASS : BLACKLIST_PAGE_SHELL_CLASS}>
        {!embedded ? (
          <header {...getLoadingStructureSlotAttributes("blacklist-header")}>
            <PageHeader
              code="BLK-01"
              title={pageTitle ?? t("title")}
              description={pageDescription ?? t("description")}
            />
          </header>
        ) : null}

        <div className={embedded ? BLACKLIST_EMBEDDED_CONTENT_SHELL_CLASS : BLACKLIST_CONTENT_SHELL_CLASS}>
          <div className={BLACKLIST_WORKBENCH_GRID_CLASS}>
            <Card
              {...getLoadingStructureSlotAttributes("blacklist-list")}
              data-testid="blacklist-rule-list"
              className={BLACKLIST_RULE_LIST_CARD_CLASS}
            >
              <CardHeader className={BLACKLIST_RULE_LIST_HEADER_CLASS}>
                <CardTitle>{isTargetScope ? t("ruleList.targetTitle") : t("ruleList.title")}</CardTitle>
                <p className={textRole.bodySubtle}>
                  {isTargetScope ? t("ruleList.targetDescription") : t("ruleList.description")}
                </p>
              </CardHeader>
              <CardContent className={BLACKLIST_RULE_LIST_CONTENT_CLASS}>
                {isTargetScope ? (
                  <RuleGroup
                    groupKey="inherited"
                    title={t("groups.inherited")}
                    description={t("ruleList.inheritedDescription")}
                    rules={inheritedRules}
                    collapsed={collapsedGroups.inherited}
                    onToggle={toggleRuleGroup}
                    emptyText={t("ruleList.emptyInherited")}
                    validLabel={t("ruleList.valid")}
                    errorLabel={t("ruleList.error")}
                    readOnly
                  />
                ) : null}
                <RuleGroup
                  groupKey="domain"
                  title={t("groups.domain")}
                  rules={ruleGroups.domain}
                  collapsed={collapsedGroups.domain}
                  onToggle={toggleRuleGroup}
                  emptyText={t("ruleList.emptyDomain")}
                  validLabel={t("ruleList.valid")}
                  errorLabel={t("ruleList.error")}
                />
                <RuleGroup
                  groupKey="ipv4"
                  title={t("groups.ipv4")}
                  rules={ruleGroups.ipv4}
                  collapsed={collapsedGroups.ipv4}
                  onToggle={toggleRuleGroup}
                  emptyText={t("ruleList.emptyIPv4")}
                  validLabel={t("ruleList.valid")}
                  errorLabel={t("ruleList.error")}
                />
                <RuleGroup
                  groupKey="cidr"
                  title={t("groups.cidr")}
                  rules={ruleGroups.cidr}
                  collapsed={collapsedGroups.cidr}
                  onToggle={toggleRuleGroup}
                  emptyText={t("ruleList.emptyCidr")}
                  validLabel={t("ruleList.valid")}
                  errorLabel={t("ruleList.error")}
                />
                {errorCount > 0 ? (
                  <RuleGroup
                    groupKey="invalid"
                    title={t("groups.invalid")}
                    rules={ruleGroups.invalid}
                    collapsed={collapsedGroups.invalid}
                    onToggle={toggleRuleGroup}
                    emptyText={t("ruleList.emptyInvalid")}
                    validLabel={t("ruleList.valid")}
                    errorLabel={t("ruleList.error")}
                  />
                ) : null}
              </CardContent>
            </Card>

            <Card data-testid="blacklist-rule-editor" className={BLACKLIST_EDITOR_CARD_CLASS}>
              <CardHeader
                {...getLoadingStructureSlotAttributes("blacklist-controls")}
                className={BLACKLIST_EDITOR_HEADER_CLASS}
              >
                <div className={BLACKLIST_EDITOR_HEADER_ROW_CLASS}>
                  <div className={BLACKLIST_EDITOR_TITLE_GROUP_CLASS}>
                    <CardTitle>{editorTitle}</CardTitle>
                    <p className={textRole.bodySubtle}>{editorDescription}</p>
                    <div className={BLACKLIST_EDITOR_EXAMPLES_ROW_CLASS}>
                      <span className={textRole.helperText}>{t("editor.examples")}</span>
                      {EXAMPLE_RULES.map((example) => (
                        <RuleKindPill key={example} kind={parseBlacklistRules(example)[0]?.kind ?? "invalid"} label={example} />
                      ))}
                    </div>
                  </div>
                </div>
              </CardHeader>
              <CardContent className={BLACKLIST_EDITOR_CONTENT_CLASS}>
                <BulkLineValidationInput
                  id={isTargetScope ? "target-blacklist-rules" : "global-blacklist-rules"}
                  name="blacklistRules"
                  label={editorLabel}
                  labelClassName="sr-only"
                  placeholder={t("placeholder")}
                  value={blacklistText}
                  lineCount={lineCount}
                  lineNumbersRef={lineNumbersRef}
                  textareaRef={textareaRef}
                  onValueChange={handleTextChange}
                  onScroll={handleTextareaScroll}
                  disabled={isSaving}
                  helper={t("editor.helper")}
                  example={t("editor.example")}
                  emptySummary={t("editor.emptySummary")}
                  validSummary={t("editor.validSummary", { count: validCount })}
                  blockingSummary={t("editor.blockingSummary", { count: errorCount })}
                  collapseDetails={t("editor.collapseDetails")}
                  expandDetails={t("editor.expandDetails")}
                  validationResult={validationResult}
                  showSuccessSummary={false}
                  fillHeight
                />
                <div className={BLACKLIST_EDITOR_ACTION_ROW_CLASS}>
                  <div className={BLACKLIST_EDITOR_STATUS_SLOT_CLASS}>
                    <Badge variant={errorCount > 0 ? "error" : "success"}>
                      {errorCount > 0
                        ? t("editor.validation", { valid: validCount, errors: errorCount })
                        : t("editor.validSummary", { count: validCount })}
                    </Badge>
                  </div>
                  <div className={BLACKLIST_EDITOR_ACTION_GROUP_CLASS}>
                    {hasChanges ? (
                      <span className={cn("flex items-center gap-2", textRole.helperText)}>
                        <span className="h-2 w-2 rounded-full bg-warning" aria-hidden="true" />
                        {t("editor.modified")}
                      </span>
                    ) : null}
                    <Button
                      type="button"
                      onClick={handleSave}
                      disabled={isSaving || errorCount > 0 || !editablePolicy?.etag}
                      loading={isSaving}
                      loadingLabel={t("save")}
                    >
                      {t("save")}
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>
        </div>
      </div>
    </ContentHandoff>
  )
}

function RuleGroup({
  groupKey,
  title,
  description,
  rules,
  collapsed,
  onToggle,
  emptyText,
  validLabel,
  errorLabel,
  readOnly = false,
}: {
  groupKey: RuleGroupKey
  title: string
  description?: string
  rules: ParsedBlacklistRule[]
  collapsed: boolean
  onToggle: (group: RuleGroupKey) => void
  emptyText: string
  validLabel?: string
  errorLabel?: string
  readOnly?: boolean
}) {
  return (
    <section data-testid={readOnly ? "blacklist-inherited-rules" : undefined}>
      <Button
        type="button"
        variant="ghost"
        aria-expanded={!collapsed}
        onClick={() => onToggle(groupKey)}
        className={BLACKLIST_RULE_GROUP_TRIGGER_CLASS}
      >
        <ChevronDown className={cn("h-4 w-4", collapsed && "-rotate-90")} aria-hidden="true" />
        <span className={textRole.sectionTitle}>{title}</span>
        <Badge variant="secondary" className="rounded-full px-1.5 py-0">
          {rules.length}
        </Badge>
        {readOnly ? <span className={textRole.helperText}>{description}</span> : null}
      </Button>
      {!collapsed && rules.length > 0 ? (
        <div className={BLACKLIST_RULE_GROUP_TABLE_CLASS}>
          {rules.map((rule) => (
            <div key={`${groupKey}-${rule.line}-${rule.value}`} className={BLACKLIST_RULE_ROW_CLASS}>
              <div className={BLACKLIST_RULE_ICON_CELL_CLASS}>
                <RuleKindIcon kind={rule.kind} />
              </div>
              <div className={BLACKLIST_RULE_VALUE_CELL_CLASS}>
                <span className={cn("truncate px-3 py-3", textRole.code)}>{rule.value}</span>
              </div>
              <span className={BLACKLIST_RULE_STATUS_CELL_CLASS}>
                <span
                  className={cn("h-2 w-2 rounded-full", rule.valid ? "bg-success" : "bg-destructive")}
                  aria-label={rule.valid ? (validLabel ?? "Valid") : (errorLabel ?? "Error")}
                  role="status"
                />
              </span>
            </div>
          ))}
        </div>
      ) : !collapsed ? (
        <div className="rounded-md border border-dashed px-4 py-4 text-center">
          <p className={textRole.helperText}>{emptyText}</p>
        </div>
      ) : null}
    </section>
  )
}

function RuleKindIcon({ kind }: { kind: BlacklistRuleKind }) {
  const Icon = kind === "domain"
    ? semanticIcons.concept.domain
    : kind === "ipv4"
      ? semanticIcons.concept.ip
      : kind === "cidr"
        ? semanticIcons.concept.cidr
        : semanticIcons.status.error

  return <Icon className="h-4 w-4" aria-hidden="true" />
}

function RuleKindPill({ kind, label }: { kind: BlacklistRuleKind; label: string }) {
  return (
    <Badge variant={kind === "cidr" ? "success" : "secondary"} className={cn("rounded-md px-2 py-0", textRole.code)}>
      {label}
    </Badge>
  )
}
