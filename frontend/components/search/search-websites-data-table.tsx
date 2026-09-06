"use client"

import * as React from "react"

import { IconFileExport } from "@/components/icons"
import { buildExportOptions } from "@/components/shared/data-table"
import { ToolbarActionMenu } from "@/components/shared/data-table/menu-owners"
import { Checkbox } from "@/components/ui/checkbox"
import { DropdownMenuItem } from "@/components/ui/dropdown-menu"
import {
  RelationEvidenceListFrame,
  RelationScreenshotPreviewDialog,
  WebsiteRelationEvidenceLoadingState,
  WebsiteRelationEvidenceRow,
  toWebsiteEvidence,
  type RelationScreenshotPreview,
} from "@/components/websites/website-relation-evidence-view"
import { escapeCSV, formatArrayForCSV, formatDateForCSV } from "@/lib/csv-utils"
import { saveBlobAsFile } from "@/lib/file-save-utils"
import { textRole } from "@/lib/typography"
import { useTranslations } from "next-intl"
import type { ExportOption } from "@/types/data-table.types"
import type { WebsiteSearchResult } from "@/types/search.types"
import type { WebSite } from "@/types/website.types"

function toWebSite(result: WebsiteSearchResult): WebSite {
  return {
    id: result.id,
    target: result.targetId,
    url: result.url,
    host: result.host,
    location: result.location,
    title: result.title,
    webserver: result.webserver,
    contentType: result.contentType,
    statusCode: result.statusCode,
    contentLength: result.contentLength,
    responseBody: result.responseBody,
    responseHeaders: result.responseHeaders,
    tech: result.tech,
    vhost: result.vhost,
    subdomain: "",
    createdAt: result.createdAt,
  }
}

function generateCSV(items: WebSite[]): string {
  const headers = [
    "url",
    "host",
    "location",
    "title",
    "status_code",
    "content_length",
    "content_type",
    "webserver",
    "tech",
    "response_headers",
    "response_body",
    "vhost",
    "created_at",
  ]
  const rows = items.map((item) => [
    escapeCSV(item.url),
    escapeCSV(item.host),
    escapeCSV(item.location),
    escapeCSV(item.title),
    escapeCSV(item.statusCode),
    escapeCSV(item.contentLength),
    escapeCSV(item.contentType),
    escapeCSV(item.webserver),
    escapeCSV(formatArrayForCSV(item.tech)),
    escapeCSV(item.responseHeaders),
    escapeCSV(item.responseBody),
    escapeCSV(item.vhost),
    escapeCSV(formatDateForCSV(item.createdAt)),
  ].join(","))
  return "\ufeff" + [headers.join(","), ...rows].join("\n")
}

export interface SearchWebsitesResultModel {
  websites: WebSite[]
  selectedIds: ReadonlySet<number>
  selectedCount: number
  allRowsSelected: boolean
  exportOptions: ExportOption[]
  onRowSelectionChange: (website: WebSite, checked: boolean) => void
  onCurrentPageSelectionChange: (checked: boolean) => void
}

// The page-level export menu and row checkboxes share this state so global search stays the only query control.
export function useSearchWebsitesResultModel(results: WebsiteSearchResult[]): SearchWebsitesResultModel {
  const tExport = useTranslations("common.export")
  const [selectedRows, setSelectedRows] = React.useState<WebSite[]>([])

  const websites = React.useMemo(() => results.map(toWebSite), [results])
  const selectedIds = React.useMemo(() => new Set(selectedRows.map((website) => website.id)), [selectedRows])

  React.useEffect(() => {
    setSelectedRows([])
  }, [results])

  const exportItems = React.useCallback((items: WebSite[], suffix: string) => {
    saveBlobAsFile(
      new Blob([generateCSV(items)], { type: "text/csv;charset=utf-8" }),
      `search-websites-${suffix}-${Date.now()}.csv`
    )
  }, [])

  const exportOptions = React.useMemo(
    () => buildExportOptions(tExport, {
      onExportAll: () => exportItems(websites, "all"),
      onExportSelected: () => exportItems(selectedRows, "selected"),
    }),
    [exportItems, selectedRows, tExport, websites]
  )

  const onRowSelectionChange = React.useCallback((website: WebSite, checked: boolean) => {
    setSelectedRows((current) => checked
      ? [...current.filter((item) => item.id !== website.id), website]
      : current.filter((item) => item.id !== website.id))
  }, [])

  const onCurrentPageSelectionChange = React.useCallback((checked: boolean) => {
    setSelectedRows((current) => {
      const visibleIdSet = new Set(websites.map((website) => website.id))
      const retained = current.filter((website) => !visibleIdSet.has(website.id))
      return checked ? [...retained, ...websites] : retained
    })
  }, [websites])

  return {
    websites,
    selectedIds,
    selectedCount: selectedRows.length,
    allRowsSelected: websites.length > 0 && websites.every((website) => selectedIds.has(website.id)),
    exportOptions,
    onRowSelectionChange,
    onCurrentPageSelectionChange,
  }
}

export function SearchWebsiteExportMenu({ model }: { model: SearchWebsitesResultModel }) {
  const tActions = useTranslations("common.actions")

  if (model.exportOptions.length === 0) return null

  return (
    <ToolbarActionMenu
      label={tActions("export")}
      icon={<IconFileExport aria-hidden="true" className="size-4" />}
      buttonSize="default"
    >
      {model.exportOptions.map((option) => {
        const disabled = typeof option.disabled === "function" ? option.disabled(model.selectedCount) : option.disabled
        return (
          <DropdownMenuItem key={option.key} onClick={option.onClick} disabled={disabled}>
            {option.label}
          </DropdownMenuItem>
        )
      })}
    </ToolbarActionMenu>
  )
}

export function SearchWebsitesDataTableLoadingState({
  pagination,
  rowCount,
}: {
  pagination: React.ReactNode
  rowCount: number
}) {
  return <WebsiteRelationEvidenceLoadingState toolbar={null} footer={pagination} rowCount={rowCount} />
}

interface SearchWebsitesDataTableProps {
  model: SearchWebsitesResultModel
  pagination: React.ReactNode
}

export function SearchWebsitesDataTable({ model, pagination }: SearchWebsitesDataTableProps) {
  const tDataTable = useTranslations("dataTable")
  const [screenshotPreview, setScreenshotPreview] = React.useState<RelationScreenshotPreview | null>(null)
  const { websites, selectedIds } = model

  return (
    <div className="min-w-0">
      <RelationEvidenceListFrame
        toolbar={null}
        selectionControl={(
          <Checkbox
            checked={model.allRowsSelected}
            onCheckedChange={(checked) => model.onCurrentPageSelectionChange(checked === true)}
            aria-label={model.allRowsSelected ? tDataTable("deselectAll") : tDataTable("selectAll")}
          />
        )}
        footer={pagination}
      >
        {websites.length > 0 ? websites.map((website) => (
          <WebsiteRelationEvidenceRow
            key={website.resourceName ?? website.id}
            targetId={website.target}
            item={toWebsiteEvidence(website)}
            website={website}
            selected={selectedIds.has(website.id)}
            onSelectedChange={(checked) => model.onRowSelectionChange(website, checked)}
            onPreviewScreenshot={setScreenshotPreview}
          />
        )) : (
          <div className="flex min-h-52 items-center justify-center px-4 text-center">
            <p className={textRole.bodySubtle}>{tDataTable("noResults")}</p>
          </div>
        )}
      </RelationEvidenceListFrame>
      <RelationScreenshotPreviewDialog
        preview={screenshotPreview}
        onOpenChange={(open) => { if (!open) setScreenshotPreview(null) }}
      />
    </div>
  )
}
