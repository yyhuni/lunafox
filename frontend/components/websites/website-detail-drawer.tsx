"use client"

import { useTranslations } from "next-intl"

import {
  DetailDrawer,
} from "@/components/shared/detail-drawer"
import { CopyButton } from "@/components/shared/feedback/copy-button"
import { HttpStatusBadge } from "@/components/shared/status/http-status-badge"
import { Badge } from "@/components/ui/badge"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

import type { WebSite } from "@/types/website.types"

interface WebsiteDetailDrawerProps {
  website: WebSite | null
  open: boolean
  onOpenChange: (open: boolean) => void
  formatDate: (value: string) => string
}

export function WebsiteDetailDrawer({
  website,
  open,
  onOpenChange,
  formatDate,
}: WebsiteDetailDrawerProps) {
  const tActions = useTranslations("common.actions")
  const tColumns = useTranslations("columns")
  const tStatus = useTranslations("common.status")
  const tTooltips = useTranslations("tooltips")

  const title = website?.title || website?.host || website?.url || ""

  return (
    <DetailDrawer
      open={open}
      onOpenChange={onOpenChange}
      title={title}
      description={website?.url}
      titleMeta={website ? <HttpStatusBadge statusCode={website.statusCode} /> : null}
      headerMeta={website ? (
        <div className={cn("min-w-0 truncate", textRole.caption)} title={website.url}>
          {website.url}
        </div>
      ) : null}
      headerClassName="py-3"
    >
      {website ? (
        <div className="min-h-0 flex-1 overflow-y-auto px-6 py-5">
          <div className="space-y-7">
            <WebsiteDrawerSection label={tActions("details")}>
              <dl className="grid grid-cols-1 gap-x-6 gap-y-4 sm:grid-cols-2">
                <WebsiteDetailField label={tColumns("website.url")} value={website.url} className="sm:col-span-2" />
                <WebsiteDetailField label={tColumns("website.host")} value={website.host} />
                <WebsiteDetailField label={tColumns("website.title")} value={website.title} />
                <WebsiteDetailField label={tColumns("website.contentLength")} value={website.contentLength || "-"} mono />
                <WebsiteDetailField label={tColumns("website.webServer")} value={website.webserver} />
                <WebsiteDetailField label={tColumns("website.contentType")} value={website.contentType} />
                <WebsiteDetailField label={tColumns("website.location")} value={website.location} />
                <WebsiteDetailField label={tColumns("website.vhost")} value={website.vhost === null ? "-" : String(website.vhost)} />
                <WebsiteDetailField label={tColumns("website.createdAt")} value={formatDate(website.createdAt)} />
              </dl>
            </WebsiteDrawerSection>

            <WebsiteDrawerSection label={tColumns("website.technologies")}>
              {website.tech.length > 0 ? (
                <div className="flex flex-wrap gap-2">
                  {website.tech.map((technology) => (
                    <Badge key={technology} variant="outline">{technology}</Badge>
                  ))}
                </div>
              ) : (
                <p className={textRole.bodySubtle}>{tStatus("noData")}</p>
              )}
            </WebsiteDrawerSection>

            <WebsiteDrawerSection label={tColumns("website.responseHeaders")}>
              <ResponseCodeRegion
                value={website.responseHeaders}
                emptyLabel={tStatus("noData")}
                copyLabel={tActions("copy")}
                copiedLabel={tTooltips("copied")}
                toastId={`website-response-headers-${website.id}`}
              />
            </WebsiteDrawerSection>

            <WebsiteDrawerSection label={tColumns("website.responseBody")}>
              <ResponseCodeRegion
                value={website.responseBody}
                emptyLabel={tStatus("noData")}
                copyLabel={tActions("copy")}
                copiedLabel={tTooltips("copied")}
                toastId={`website-response-body-${website.id}`}
              />
            </WebsiteDrawerSection>
          </div>
        </div>
      ) : null}
    </DetailDrawer>
  )
}

function WebsiteDrawerSection({ label, children }: React.PropsWithChildren<{ label: string }>) {
  return (
    <section className="min-w-0 space-y-3">
      <h3 className={textRole.sectionTitle}>{label}</h3>
      {children}
    </section>
  )
}

function WebsiteDetailField({
  label,
  value,
  mono = false,
  className,
}: {
  label: string
  value: string | number
  mono?: boolean
  className?: string
}) {
  return (
    <div className={cn("min-w-0 space-y-1", className)}>
      <dt className={textRole.metadataLabel}>{label}</dt>
      <dd className={cn(textRole.metadataValueStrong, "break-words", mono && "font-mono tabular-nums")}>
        {value || "-"}
      </dd>
    </div>
  )
}

function ResponseCodeRegion({
  value,
  emptyLabel,
  copyLabel,
  copiedLabel,
  toastId,
}: {
  value?: string
  emptyLabel: string
  copyLabel: string
  copiedLabel: string
  toastId: string
}) {
  if (!value) {
    return <p className={textRole.bodySubtle}>{emptyLabel}</p>
  }

  return (
    <div className="relative min-h-0">
      <div className="absolute top-2 right-2 z-10">
        <CopyButton value={value} copyLabel={copyLabel} copiedLabel={copiedLabel} toastId={toastId} />
      </div>
      <pre className="max-h-96 overflow-auto rounded-md border bg-muted/30 p-3 pr-12 font-mono text-xs leading-5 whitespace-pre-wrap break-words">
        {value}
      </pre>
    </div>
  )
}
