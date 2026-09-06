"use client"

import { useRef } from "react"
import { useTranslations } from "next-intl"

import {
  lineNumberedTextareaRowClassName,
  getLineNumberGutterWidth,
} from "@/components/common/line-numbered-textarea"
import { BulkLineValidationInput } from "@/components/common/bulk-line-validation-input"
import { PageHeader } from "@/components/common/page-header"
import { ActionSkeleton } from "@/components/shared/loading/action-skeleton"
import {
  getLoadingOwnerAttributes,
  getLoadingStructureSlotAttributes,
} from "@/components/shared/loading/loading-owner"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import {
  BLACKLIST_CONTENT_SHELL_CLASS,
  BLACKLIST_EMBEDDED_CONTENT_SHELL_CLASS,
  BLACKLIST_EMBEDDED_PAGE_SHELL_CLASS,
  BLACKLIST_EDITOR_ACTION_GROUP_CLASS,
  BLACKLIST_EDITOR_ACTION_ROW_CLASS,
  BLACKLIST_EDITOR_CARD_CLASS,
  BLACKLIST_EDITOR_CONTENT_CLASS,
  BLACKLIST_EDITOR_EXAMPLES_ROW_CLASS,
  BLACKLIST_EDITOR_HEADER_CLASS,
  BLACKLIST_EDITOR_HEADER_ROW_CLASS,
  BLACKLIST_EDITOR_STATUS_SLOT_CLASS,
  BLACKLIST_EDITOR_TITLE_GROUP_CLASS,
  BLACKLIST_EDITOR_VIEWPORT_CLASS,
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

export const EXAMPLE_RULES = ["example.com", "*.example.com", "192.0.2.1", "192.0.2.0/24"] as const

const LOADING_RULE_GROUPS = ["domain", "ipv4", "cidr"] as const

interface BlacklistSettingsLoadingStateProps {
  owner?: string
  className?: string
  embedded?: boolean
  scope?: "global" | "target"
  pageTitle?: string
  pageDescription?: string
}

function LoadingRuleGroup({
  title,
  rowWidths,
}: {
  title: string
  rowWidths: string[]
}) {
  return (
    <section>
      <div className={BLACKLIST_RULE_GROUP_TRIGGER_CLASS}>
        <Skeleton className="h-4 w-4 rounded-sm" />
        <h3 className={textRole.sectionTitle}>{title}</h3>
        <Badge variant="secondary" className="rounded-full px-1.5 py-0">
          <span className="inline-flex">
            <Skeleton className="h-4 w-4 rounded-full" />
          </span>
        </Badge>
      </div>
      <div className={BLACKLIST_RULE_GROUP_TABLE_CLASS}>
        {rowWidths.map((width, index) => (
          <div
            key={`${title}-${width}-${index}`}
            className={BLACKLIST_RULE_ROW_CLASS}
          >
            <div className={BLACKLIST_RULE_ICON_CELL_CLASS}>
              <Skeleton className="h-4 w-4 rounded-sm" />
            </div>
            <div className={BLACKLIST_RULE_VALUE_CELL_CLASS}>
              <Skeleton className={cn("h-4 rounded-md", width)} />
            </div>
            <span className={BLACKLIST_RULE_STATUS_CELL_CLASS}>
              <Skeleton className="h-2 w-2 rounded-full" />
            </span>
          </div>
        ))}
      </div>
    </section>
  )
}

export function BlacklistSettingsLoadingState({
  owner,
  className,
  embedded = false,
  scope = "global",
  pageTitle,
  pageDescription,
}: BlacklistSettingsLoadingStateProps) {
  const t = useTranslations("pages.settings.blacklist")
  const editorLabel = scope === "target" ? t("editor.targetInputLabel") : t("editor.inputLabel")
  const lineNumbersRef = useRef<HTMLDivElement | null>(null)
  const textareaRef = useRef<HTMLTextAreaElement | null>(null)

  if (owner !== undefined && !owner.trim()) {
    throw new Error("BlacklistSettingsLoadingState requires a non-empty owner.")
  }

  return (
    <div
      {...(owner ? getLoadingOwnerAttributes({ owner, layer: "route", intent: "route" }) : {})}
      data-slot="blacklist-settings-loading-state"
      className={cn(
        embedded ? BLACKLIST_EMBEDDED_PAGE_SHELL_CLASS : BLACKLIST_PAGE_SHELL_CLASS,
        className
      )}
    >
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
            className={BLACKLIST_RULE_LIST_CARD_CLASS}
          >
            <CardHeader className={BLACKLIST_RULE_LIST_HEADER_CLASS}>
              <CardTitle>{scope === "target" ? t("ruleList.targetTitle") : t("ruleList.title")}</CardTitle>
              <p className={textRole.bodySubtle}>
                {scope === "target" ? t("ruleList.targetDescription") : t("ruleList.description")}
              </p>
            </CardHeader>
            <CardContent className={BLACKLIST_RULE_LIST_CONTENT_CLASS}>
              {scope === "target" ? (
                <LoadingRuleGroup title={t("groups.inherited")} rowWidths={["w-28", "w-36"]} />
              ) : null}
              <LoadingRuleGroup title={t(`groups.${LOADING_RULE_GROUPS[0]}`)} rowWidths={["w-28", "w-36"]} />
              <LoadingRuleGroup title={t(`groups.${LOADING_RULE_GROUPS[1]}`)} rowWidths={["w-24"]} />
              <LoadingRuleGroup title={t(`groups.${LOADING_RULE_GROUPS[2]}`)} rowWidths={["w-32", "w-20"]} />
            </CardContent>
          </Card>

          <Card className={BLACKLIST_EDITOR_CARD_CLASS}>
            <CardHeader
              {...getLoadingStructureSlotAttributes("blacklist-controls")}
              className={BLACKLIST_EDITOR_HEADER_CLASS}
            >
              <div className={BLACKLIST_EDITOR_HEADER_ROW_CLASS}>
                <div className={BLACKLIST_EDITOR_TITLE_GROUP_CLASS}>
                  <CardTitle>{scope === "target" ? t("editor.targetTitle") : t("editor.title")}</CardTitle>
                  <p className={textRole.bodySubtle}>
                    {scope === "target" ? t("editor.targetDescription") : t("editor.description")}
                  </p>
                  <div className={BLACKLIST_EDITOR_EXAMPLES_ROW_CLASS}>
                    <span className={textRole.helperText}>{t("editor.examples")}</span>
                    {EXAMPLE_RULES.map((example) => (
                      <Badge
                        key={example}
                        variant={example.includes("/") ? "success" : "secondary"}
                        className={cn("rounded-md px-2 py-0", textRole.code)}
                      >
                        {example}
                      </Badge>
                    ))}
                  </div>
                </div>
              </div>
            </CardHeader>
            <CardContent className={BLACKLIST_EDITOR_CONTENT_CLASS}>
              {/* The shared field owns both branches; loading only masks volatile editor contents. */}
              <div className="relative min-h-0 flex-1">
                <BulkLineValidationInput
                  id={scope === "target" ? "target-blacklist-rules-loading" : "global-blacklist-rules-loading"}
                  name="blacklistRules"
                  label={editorLabel}
                  labelClassName="sr-only"
                  placeholder=""
                  value=""
                  lineCount={10}
                  lineNumbersRef={lineNumbersRef}
                  textareaRef={textareaRef}
                  onValueChange={() => undefined}
                  onScroll={() => undefined}
                  disabled
                  helper=""
                  example=""
                  emptySummary=""
                  validSummary=""
                  blockingSummary=""
                  collapseDetails=""
                  expandDetails=""
                  validationResult={null}
                  showEmptySummary={false}
                  showSuccessSummary={false}
                  viewportClassName={BLACKLIST_EDITOR_VIEWPORT_CLASS}
                  fillHeight
                />
                <div
                  aria-hidden="true"
                  className="pointer-events-none absolute inset-y-0 right-0 z-30 overflow-hidden bg-background px-3 py-3"
                  style={{ left: getLineNumberGutterWidth(10) }}
                >
                  {["w-28", "w-36", "w-24", "w-32", "w-20"].map((width, index) => (
                    <div
                      key={`${width}-${index}`}
                      className={cn(lineNumberedTextareaRowClassName, "flex items-center")}
                    >
                      <Skeleton className={cn("h-4 rounded-md", width)} />
                    </div>
                  ))}
                </div>
              </div>
              <div className={BLACKLIST_EDITOR_ACTION_ROW_CLASS}>
                <Badge variant="secondary" className={BLACKLIST_EDITOR_STATUS_SLOT_CLASS}>
                  <span className="inline-flex">
                    <Skeleton className="h-4 w-28 rounded-full" />
                  </span>
                </Badge>
                <div className={BLACKLIST_EDITOR_ACTION_GROUP_CLASS}>
                  <ActionSkeleton />
                </div>
              </div>
            </CardContent>
          </Card>
        </div>

      </div>
    </div>
  )
}
