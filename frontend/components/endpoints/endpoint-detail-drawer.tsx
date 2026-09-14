"use client"

import { useTranslations } from "next-intl"

import {
  DetailDrawer,
  DETAIL_DRAWER_COMPACT_BODY_CLASS,
  DETAIL_DRAWER_COMPACT_FIELD_GRID_CLASS,
  DETAIL_DRAWER_COMPACT_SECTION_STACK_CLASS,
} from "@/components/shared/detail-drawer"
import { CopyButton } from "@/components/shared/feedback/copy-button"
import { HttpStatusBadge } from "@/components/shared/status/http-status-badge"
import { Badge } from "@/components/ui/badge"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

import type { Endpoint } from "@/types/endpoint.types"

interface EndpointDetailDrawerProps {
  endpoint: Endpoint | null
  open: boolean
  onOpenChange: (open: boolean) => void
  formatDate: (value: string) => string
}

export function EndpointDetailDrawer({
  endpoint,
  open,
  onOpenChange,
  formatDate,
}: EndpointDetailDrawerProps) {
  const tActions = useTranslations("common.actions")
  const tColumns = useTranslations("columns")
  const tStatus = useTranslations("common.status")
  const tTooltips = useTranslations("tooltips")
  const title = endpoint?.title || endpoint?.host || endpoint?.url || ""

  return (
    <DetailDrawer
      open={open}
      onOpenChange={onOpenChange}
      title={title}
      description={endpoint?.url}
      titleMeta={endpoint ? <HttpStatusBadge statusCode={endpoint.statusCode} /> : null}
      headerMeta={endpoint ? (
        <div className={cn("min-w-0 truncate", textRole.caption)} title={endpoint.url}>
          {endpoint.url}
        </div>
      ) : null}
    >
      {endpoint ? (
        <div className={DETAIL_DRAWER_COMPACT_BODY_CLASS}>
          <div className={DETAIL_DRAWER_COMPACT_SECTION_STACK_CLASS}>
            <EndpointDrawerSection label={tActions("details")}>
              <dl className={DETAIL_DRAWER_COMPACT_FIELD_GRID_CLASS}>
                <EndpointDetailField label={tColumns("common.url")} value={endpoint.url} className="sm:col-span-2" />
                <EndpointDetailField label={tColumns("endpoint.host")} value={endpoint.host} />
                <EndpointDetailField label={tColumns("endpoint.title")} value={endpoint.title} />
                <EndpointDetailField label={tColumns("endpoint.contentLength")} value={endpoint.contentLength ?? "-"} mono />
                <EndpointDetailField label={tColumns("endpoint.webServer")} value={endpoint.webserver} />
                <EndpointDetailField label={tColumns("endpoint.contentType")} value={endpoint.contentType} />
                <EndpointDetailField label={tColumns("endpoint.location")} value={endpoint.location} />
                <EndpointDetailField label={tColumns("endpoint.vhost")} value={endpoint.vhost == null ? "-" : String(endpoint.vhost)} />
                <EndpointDetailField label={tColumns("common.createdAt")} value={endpoint.createdAt ? formatDate(endpoint.createdAt) : "-"} />
              </dl>
            </EndpointDrawerSection>

            <EndpointDrawerSection label={tColumns("endpoint.technologies")}>
              {endpoint.tech?.length ? (
                <div className="flex flex-wrap gap-1.5">
                  {endpoint.tech.map((technology) => (
                    <Badge key={technology} variant="outline">{technology}</Badge>
                  ))}
                </div>
              ) : (
                <p className={textRole.bodySubtle}>{tStatus("noData")}</p>
              )}
            </EndpointDrawerSection>

            <EndpointDrawerSection label={tColumns("endpoint.responseHeaders")}>
              <ResponseCodeRegion
                value={endpoint.responseHeaders}
                emptyLabel={tStatus("noData")}
                copyLabel={tActions("copy")}
                copiedLabel={tTooltips("copied")}
                toastId={`endpoint-response-headers-${endpoint.id}`}
              />
            </EndpointDrawerSection>

            <EndpointDrawerSection label={tColumns("endpoint.responseBody")}>
              <ResponseCodeRegion
                value={endpoint.responseBody}
                emptyLabel={tStatus("noData")}
                copyLabel={tActions("copy")}
                copiedLabel={tTooltips("copied")}
                toastId={`endpoint-response-body-${endpoint.id}`}
              />
            </EndpointDrawerSection>
          </div>
        </div>
      ) : null}
    </DetailDrawer>
  )
}

function EndpointDrawerSection({ label, children }: React.PropsWithChildren<{ label: string }>) {
  return (
    <section className="min-w-0 space-y-3">
      <h3 className={textRole.sectionTitle}>{label}</h3>
      {children}
    </section>
  )
}

function EndpointDetailField({
  label,
  value,
  mono = false,
  className,
}: {
  label: string
  value: string | number | null | undefined
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
    <div className="flex min-h-0 flex-col gap-2">
      <div className="flex justify-end">
        <CopyButton value={value} copyLabel={copyLabel} copiedLabel={copiedLabel} toastId={toastId} />
      </div>
      <pre className="max-h-96 overflow-auto rounded-md border bg-muted/30 p-3 font-mono text-xs leading-5 whitespace-pre-wrap break-words">
        {value}
      </pre>
    </div>
  )
}
