"use client"

import { FileText, IconBook } from "@/components/icons"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { ScrollArea } from "@/components/ui/scroll-area"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import {
  GLOBAL_ASSET_SEARCH_FEATURED_GUIDANCE,
  GLOBAL_ASSET_SEARCH_FIELDS,
  GLOBAL_ASSET_SEARCH_FIELD_LABEL_KEYS,
  GLOBAL_ASSET_SEARCH_GUIDANCE,
  GLOBAL_ASSET_SEARCH_GUIDANCE_CATEGORIES,
  GLOBAL_ASSET_SEARCH_OPERATORS,
  type GlobalAssetSearchField,
} from "@/lib/global-asset-search-query"

export const GLOBAL_ASSET_SEARCH_SYNTAX_EXAMPLES = GLOBAL_ASSET_SEARCH_GUIDANCE.map((entry) => entry.query)

type SearchGuidanceTranslator = (key: string) => string

type SearchSyntaxGuidanceProps = {
  t: SearchGuidanceTranslator
  onSelectField: (field: GlobalAssetSearchField) => void
  onSelectExample: (example: string) => void
  onOpenManual: () => void
}

export type SearchSyntaxManualContentProps = {
  t: SearchGuidanceTranslator
  onSelectExample: (example: string) => void
}

export function appendGlobalAssetSearchCondition(
  draft: string,
  field: GlobalAssetSearchField
): string {
  const current = draft.trim()
  return `${current ? `${current} && ` : ""}${field}=""`
}

const globalAssetSearchFieldPattern = GLOBAL_ASSET_SEARCH_FIELDS.join("|")
const incompleteQuotedValuePattern = new RegExp(
  `^(?:${globalAssetSearchFieldPattern})(?:==|=)"(?:\\\\.|[^"\\\\])+$`
)
const completedConditionPattern = new RegExp(
  `(?:^|&&\\s*)(?:${globalAssetSearchFieldPattern})(?:==|=)"(?:\\\\.|[^"\\\\])*"\\s*$`
)

export function getGlobalAssetSearchInlineCompletion(draft: string): string {
  if (!draft) return ""

  if (completedConditionPattern.test(draft)) return " && "

  const connectorIndex = draft.lastIndexOf("&&")
  const currentCondition = (connectorIndex === -1 ? draft : draft.slice(connectorIndex + 2)).trimStart()
  if (!currentCondition) return ""

  const matchingField = GLOBAL_ASSET_SEARCH_FIELDS.find((field) => (
    field.startsWith(currentCondition) && field !== currentCondition
  ))
  if (matchingField) return `${matchingField.slice(currentCondition.length)}="`

  if (GLOBAL_ASSET_SEARCH_FIELDS.includes(currentCondition as GlobalAssetSearchField)) {
    return '="'
  }

  if (/^(?:url|host|title|statusCode|tech)(?:==|=)$/.test(currentCondition)) {
    return '"'
  }

  if (incompleteQuotedValuePattern.test(currentCondition)) return '"'

  return ""
}

function SearchExampleButton({
  entry,
  t,
  onSelectExample,
  compact = false,
}: {
  entry: (typeof GLOBAL_ASSET_SEARCH_GUIDANCE)[number]
  t: SearchGuidanceTranslator
  onSelectExample: (example: string) => void
  compact?: boolean
}) {
  return (
    <Badge
      size="tag"
      variant={compact ? "secondary" : "outline"}
      className={cn("max-w-full cursor-pointer font-mono hover:bg-accent", !compact && "justify-start")}
      render={(
        <button
          type="button"
          onClick={() => onSelectExample(entry.query)}
          aria-label={entry.query}
          title={t(entry.labelKey)}
        />
      )}
    >
      <span className="truncate">{entry.query}</span>
    </Badge>
  )
}

export function SearchSyntaxGuidance({
  t,
  onSelectField,
  onSelectExample,
  onOpenManual,
}: SearchSyntaxGuidanceProps) {
  return (
    <div className="space-y-3 p-3">
      <section className="space-y-1.5">
        <h2 className={textRole.metadataLabel}>{t("syntax.fields")}</h2>
        <div className="flex flex-wrap gap-2">
          {GLOBAL_ASSET_SEARCH_FIELDS.map((field) => (
            <Badge
              size="tag"
              key={field}
              variant="outline"
              className="cursor-pointer font-mono hover:bg-accent"
              render={(
                <button
                  type="button"
                  onClick={() => onSelectField(field)}
                  aria-label={`${field}=""`}
                />
              )}
            >
              {field}{'=""'}
            </Badge>
          ))}
        </div>
      </section>

      <section className="space-y-1.5">
        <h2 className={textRole.metadataLabel}>{t("syntax.operators")}</h2>
        <div className="flex flex-wrap gap-x-4 gap-y-1 text-muted-foreground">
          {GLOBAL_ASSET_SEARCH_OPERATORS.map((operator) => (
            <span key={operator.value} className="inline-flex items-center gap-1.5">
              <code className="radius-badge bg-muted px-1.5 py-0.5 font-mono text-xs text-foreground">{operator.value}</code>
              <span className={textRole.bodySubtle}>{t(operator.labelKey)}</span>
            </span>
          ))}
        </div>
      </section>

      {GLOBAL_ASSET_SEARCH_GUIDANCE_CATEGORIES.map((category) => (
        <section key={category.id} className="space-y-1.5">
          <h2 className={textRole.metadataLabel}>{t(category.titleKey)}</h2>
          <div className="flex flex-wrap gap-2">
            {GLOBAL_ASSET_SEARCH_GUIDANCE
              .filter((entry) => entry.category === category.id)
              .map((entry) => (
                <SearchExampleButton
                  key={entry.id}
                  entry={entry}
                  t={t}
                  onSelectExample={onSelectExample}
                  compact
                />
              ))}
          </div>
        </section>
      ))}

      <Button
        type="button"
        variant="link"
        size="sm"
        className="h-auto px-0 text-xs"
        onClick={onOpenManual}
        aria-label={t("syntax.manualAriaLabel")}
      >
        <FileText className="h-3.5 w-3.5" />
        {t("syntax.openManual")}
      </Button>
    </div>
  )
}

export function SearchSyntaxManualContent({ t, onSelectExample }: SearchSyntaxManualContentProps) {
  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="space-y-1 border-b px-4 py-3">
        <div className="flex items-center gap-2">
          <IconBook className="h-4 w-4 text-primary" aria-hidden="true" />
          <h2 className={textRole.panelTitle}>{t("manual.title")}</h2>
        </div>
        <p className={textRole.bodySubtle}>{t("manual.description")}</p>
      </div>

      <ScrollArea
        className="min-h-0 flex-1"
        viewportClassName="max-h-[min(70dvh,36rem)]"
        contentClassName="space-y-5 px-4 py-4"
      >
        <section className="space-y-2">
          <h3 className={textRole.sectionTitle}>{t("manual.fieldsTitle")}</h3>
          <div className="grid gap-2 sm:grid-cols-2">
            {GLOBAL_ASSET_SEARCH_FIELDS.map((field) => (
              <div key={field} className="flex min-w-0 items-baseline gap-2">
                <code className="shrink-0 font-mono text-xs text-foreground">{field}</code>
                <span className={cn(textRole.bodySubtle, "truncate")}>{t(GLOBAL_ASSET_SEARCH_FIELD_LABEL_KEYS[field])}</span>
              </div>
            ))}
          </div>
        </section>

        <section className="space-y-2">
          <h3 className={textRole.sectionTitle}>{t("manual.operatorsTitle")}</h3>
          <div className="space-y-1.5">
            {GLOBAL_ASSET_SEARCH_OPERATORS.map((operator) => (
              <div key={operator.value} className="flex min-w-0 items-baseline gap-2">
                <code className="shrink-0 radius-badge bg-muted px-1.5 py-0.5 font-mono text-xs text-foreground">{operator.value}</code>
                <span className={textRole.bodySubtle}>{t(operator.labelKey)}</span>
              </div>
            ))}
          </div>
        </section>

        <section className="space-y-2">
          <h3 className={textRole.sectionTitle}>{t("manual.semanticsTitle")}</h3>
          <ul className={cn(textRole.bodySubtle, "list-disc space-y-1 pl-5")}>
            <li>{t("manual.textSemantics")}</li>
            <li>{t("manual.typedSemantics")}</li>
          </ul>
        </section>

        <section className="space-y-2">
          <h3 className={textRole.sectionTitle}>{t("manual.limitsTitle")}</h3>
          <ul className={cn(textRole.bodySubtle, "list-disc space-y-1 pl-5")}>
            <li>{t("manual.urlLimit")}</li>
            <li>{t("manual.textLimit")}</li>
            <li>{t("manual.statusLimit")}</li>
            <li>{t("manual.conditionLimit")}</li>
            <li>{t("manual.quoting")}</li>
          </ul>
        </section>

        <section className="space-y-2">
          <h3 className={textRole.sectionTitle}>{t("manual.unsupportedTitle")}</h3>
          <p className={textRole.bodySubtle}>{t("manual.unsupported")}</p>
        </section>

        <section className="space-y-3">
          <h3 className={textRole.sectionTitle}>{t("manual.examplesTitle")}</h3>
          {GLOBAL_ASSET_SEARCH_GUIDANCE_CATEGORIES.map((category) => (
            <div key={category.id} className="space-y-2">
              <h4 className={textRole.metadataLabel}>{t(category.titleKey)}</h4>
              <div className="space-y-2">
                {GLOBAL_ASSET_SEARCH_GUIDANCE
                  .filter((entry) => entry.category === category.id)
                  .map((entry) => (
                    <div key={entry.id} className="min-w-0 space-y-1">
                      <SearchExampleButton entry={entry} t={t} onSelectExample={onSelectExample} />
                      <p className={cn(textRole.caption, "break-words")}>{t(entry.descriptionKey)}</p>
                    </div>
                  ))}
              </div>
            </div>
          ))}
        </section>
      </ScrollArea>
    </div>
  )
}

export { GLOBAL_ASSET_SEARCH_FEATURED_GUIDANCE }
