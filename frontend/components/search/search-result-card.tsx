"use client"

import { Badge } from "@/components/ui/badge"
import { Card, CardContent } from "@/components/ui/card"
import { ExpandableTagList } from "@/components/shared/data-table/expandable-cell"
import { ResponseEvidencePanel } from "@/components/shared/response-evidence"
import { HttpStatusBadge } from "@/components/shared/status/http-status-badge"
import { useTranslations } from "next-intl"
import type { WebsiteSearchResult } from "@/types/search.types"

interface SearchResultCardProps {
  result: WebsiteSearchResult
}

function formatBytes(value: number | null): string | null {
  if (value === null || value === undefined) return null
  if (value < 1024) return `${value} B`
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`
  return `${(value / (1024 * 1024)).toFixed(1)} MB`
}

export function SearchResultCard({ result }: SearchResultCardProps) {
  const t = useTranslations("search.card")
  const contentLength = formatBytes(result.contentLength)

  return (
    <Card className="gap-0 overflow-hidden py-0">
      <CardContent className="p-0">
        <div className="space-y-2 border-b bg-muted/30 px-4 py-2">
          <h3 className="break-all font-mono text-sm">{result.url || result.host}</h3>
          <div className="flex flex-wrap items-center gap-2">
            <HttpStatusBadge statusCode={result.statusCode} className="text-xs" />
            {result.webserver ? <Badge variant="outline" className="font-mono text-xs">{result.webserver}</Badge> : null}
            {result.contentType ? <Badge variant="outline" className="font-mono text-xs">{result.contentType.split(";")[0]}</Badge> : null}
            {contentLength ? <Badge variant="outline" className="font-mono text-xs">{contentLength}</Badge> : null}
          </div>
        </div>

        <div className="flex flex-col md:flex-row">
          <div className="flex w-full flex-col border-b px-4 py-3 md:w-80 md:shrink-0 md:border-r md:border-b-0">
            <div className="space-y-1.5 text-sm">
              <div className="flex items-baseline">
                <span className="w-12 shrink-0 text-muted-foreground">{t("title")}</span>
                <span className="truncate" title={result.title}>{result.title || "-"}</span>
              </div>
              <div className="flex items-baseline">
                <span className="w-12 shrink-0 text-muted-foreground">{t("host")}</span>
                <span className="truncate font-mono" title={result.host}>{result.host || "-"}</span>
              </div>
            </div>
            <div className="mt-3">
              <ExpandableTagList items={result.tech} maxVisible={5} variant="secondary" />
            </div>
          </div>

          <div className="flex w-full flex-col md:flex-1">
            <ResponseEvidencePanel
              responseBody={result.responseBody}
              responseHeaders={result.responseHeaders}
              location={result.location}
              labels={{
                body: t("body"),
                headers: t("header"),
                location: t("location"),
                emptyLabel: t("noResponseContent"),
              }}
            />
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
