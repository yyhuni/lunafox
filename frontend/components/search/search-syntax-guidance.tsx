"use client"

import { Badge } from "@/components/ui/badge"
import { textRole } from "@/lib/typography"
import {
  GLOBAL_ASSET_SEARCH_FIELDS,
  type GlobalAssetSearchField,
} from "@/lib/global-asset-search-query"

export const GLOBAL_ASSET_SEARCH_SYNTAX_EXAMPLES = [
  'host="api" && tech="nginx"',
  'statusCode=="200"',
  'title=="Admin" && tech="vue"',
] as const

type SearchSyntaxGuidanceProps = {
  t: (key: string) => string
  onSelectField: (field: GlobalAssetSearchField) => void
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

export function SearchSyntaxGuidance({
  t,
  onSelectField,
  onSelectExample,
}: SearchSyntaxGuidanceProps) {
  return (
    <div className="space-y-3 p-3">
      <section className="space-y-1.5">
        <h2 className={textRole.metadataLabel}>{t("syntax.fields")}</h2>
        <div className="flex flex-wrap gap-2">
          {GLOBAL_ASSET_SEARCH_FIELDS.map((field) => (
            <Badge
              key={field}
              variant="outline"
              className="cursor-pointer px-2 py-0.5 font-mono hover:bg-accent"
              render={<button type="button" onClick={() => onSelectField(field)} />}
            >
              {field}{'=""'}
            </Badge>
          ))}
        </div>
      </section>

      <section className="space-y-1.5">
        <h2 className={textRole.metadataLabel}>{t("syntax.operators")}</h2>
        <div className="flex flex-wrap gap-x-4 gap-y-1 text-muted-foreground">
          <span className="inline-flex items-center gap-1.5">
            <code className="radius-badge bg-muted px-1.5 py-0.5 font-mono text-xs text-foreground">=</code>
            <span className={textRole.bodySubtle}>{t("syntax.contains")}</span>
          </span>
          <span className="inline-flex items-center gap-1.5">
            <code className="radius-badge bg-muted px-1.5 py-0.5 font-mono text-xs text-foreground">==</code>
            <span className={textRole.bodySubtle}>{t("syntax.exact")}</span>
          </span>
          <span className="inline-flex items-center gap-1.5">
            <code className="radius-badge bg-muted px-1.5 py-0.5 font-mono text-xs text-foreground">&&</code>
            <span className={textRole.bodySubtle}>{t("syntax.all")}</span>
          </span>
        </div>
      </section>

      <section className="space-y-1.5">
        <h2 className={textRole.metadataLabel}>{t("syntax.examples")}</h2>
        <div className="flex flex-wrap gap-2">
          {GLOBAL_ASSET_SEARCH_SYNTAX_EXAMPLES.map((example) => (
            <Badge
              key={example}
              variant="secondary"
              className="cursor-pointer px-2 py-0.5 font-mono hover:bg-secondary/80"
              render={<button type="button" onClick={() => onSelectExample(example)} />}
            >
              {example}
            </Badge>
          ))}
        </div>
      </section>
    </div>
  )
}
