"use client"

import type { ReactNode } from "react"

import { ScrollArea } from "@/components/ui/scroll-area"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { textRole } from "@/lib/typography"
import { cn, formatBytes } from "@/lib/utils"

export type ResponseEvidenceLabels = {
  body: string
  headers: string
  location: string
  emptyLabel: string
}

export type ResponseEvidenceMetadata = {
  contentType: string | null | undefined
  contentLength: number | null | undefined
  labels: {
    contentType: string
    responseSize: string
  }
}

export type ResponseEvidencePanelProps = {
  responseBody: string | null | undefined
  responseHeaders: string | null | undefined
  location?: string | null
  labels: ResponseEvidenceLabels
  showMetadata?: boolean
  metadata?: ResponseEvidenceMetadata
  className?: string
}

function ResponseValue({
  value,
  emptyLabel,
}: {
  value: string | null | undefined
  emptyLabel: string
}) {
  return (
    <pre className={cn("min-w-0 whitespace-pre-wrap break-words p-3", textRole.code)}>
      {value || emptyLabel}
    </pre>
  )
}

function ResponseEvidencePanelFrame({
  className,
  children,
}: {
  className?: string
  children: ReactNode
}) {
  return (
    <section className={cn("radius-surface flex h-full min-w-0 flex-col overflow-hidden border border-border bg-muted/20", className)}>
      {children}
    </section>
  )
}

function ResponseEvidencePanelLoadingMetadata() {
  return (
    <dl className="grid grid-cols-2 gap-3 border-t border-border bg-card/50 px-3 py-2.5">
      <div className="min-w-0">
        <div aria-hidden="true" className={textRole.caption}>
          <Skeleton className="inline-block h-4 w-4/5 align-middle" />
        </div>
      </div>
      <div className="min-w-0 text-right">
        <div aria-hidden="true" className={textRole.caption}>
          <Skeleton className="inline-block h-4 w-2/3 align-middle" />
        </div>
      </div>
    </dl>
  )
}

export function ResponseEvidencePanelLoadingState({
  showMetadata = false,
  className,
}: {
  showMetadata?: boolean
  className?: string
}) {
  return (
    <ResponseEvidencePanelFrame className={className}>
      <Tabs defaultValue="body" className="flex h-full w-full flex-col gap-0">
        <div className="border-b border-border px-3">
          <TabsList variant="minimal" size="sm">
            <TabsTrigger value="body" variant="minimal" activeIndicator="fixed" size="sm">
              <Skeleton className="h-4 w-10" />
            </TabsTrigger>
            <TabsTrigger value="headers" variant="minimal" activeIndicator="fixed" size="sm">
              <Skeleton className="h-4 w-10" />
            </TabsTrigger>
          </TabsList>
        </div>

        <TabsContent value="body" className="min-h-0 flex-1">
          <ScrollArea className="min-h-0 max-h-52 flex-1 bg-muted/30" type="always" contentClassName="min-w-0">
            <div className="p-3"><Skeleton className="h-4 w-full" /></div>
          </ScrollArea>
        </TabsContent>
      </Tabs>

      {showMetadata ? <ResponseEvidencePanelLoadingMetadata /> : null}
    </ResponseEvidencePanelFrame>
  )
}

export function ResponseEvidencePanel({
  responseBody,
  responseHeaders,
  location,
  labels,
  showMetadata = false,
  metadata,
  className,
}: ResponseEvidencePanelProps) {
  const hasLocation = Boolean(location?.trim())
  const shouldRenderMetadata = showMetadata && metadata !== undefined

  return (
    <ResponseEvidencePanelFrame className={className}>
      <Tabs defaultValue="body" className="flex h-full w-full flex-col gap-0">
        <div className="border-b border-border px-3">
          <TabsList variant="minimal" size="sm">
            <TabsTrigger value="body" variant="minimal" activeIndicator="fixed" size="sm">
              {labels.body}
            </TabsTrigger>
            <TabsTrigger value="headers" variant="minimal" activeIndicator="fixed" size="sm">
              {labels.headers}
            </TabsTrigger>
            {hasLocation ? (
              <TabsTrigger value="location" variant="minimal" activeIndicator="fixed" size="sm">
                {labels.location}
              </TabsTrigger>
            ) : null}
          </TabsList>
        </div>

        <TabsContent value="body" className="min-h-0 flex-1">
          <ScrollArea className="min-h-0 max-h-52 flex-1 bg-muted/30" type="always" contentClassName="min-w-0">
            <ResponseValue value={responseBody} emptyLabel={labels.emptyLabel} />
          </ScrollArea>
        </TabsContent>
        <TabsContent value="headers" className="min-h-0 flex-1">
          <ScrollArea className="min-h-0 max-h-52 flex-1 bg-muted/30" type="always" contentClassName="min-w-0">
            <ResponseValue value={responseHeaders} emptyLabel={labels.emptyLabel} />
          </ScrollArea>
        </TabsContent>
        {hasLocation ? (
          <TabsContent value="location" className="min-h-0 flex-1">
            <ScrollArea className="min-h-0 max-h-52 flex-1 bg-muted/30" type="always" contentClassName="min-w-0">
              <ResponseValue value={location} emptyLabel={labels.emptyLabel} />
            </ScrollArea>
          </TabsContent>
        ) : null}
      </Tabs>

      {shouldRenderMetadata ? (
        <dl className="grid grid-cols-2 gap-3 border-t border-border bg-card/50 px-3 py-2.5">
          <div className="min-w-0">
            <dt className="sr-only">{metadata.labels.contentType}</dt>
            <dd className={cn("truncate", textRole.caption)} title={metadata.contentType ?? undefined}>
              {metadata.contentType || "-"}
            </dd>
          </div>
          <div className="min-w-0 text-right">
            <dt className="sr-only">{metadata.labels.responseSize}</dt>
            <dd className={cn("tabular-nums", textRole.caption)}>
              {formatBytes(metadata.contentLength ?? 0)}
            </dd>
          </div>
        </dl>
      ) : null}
    </ResponseEvidencePanelFrame>
  )
}
