"use client"

import { useState, type MouseEvent, type ReactNode } from "react"
import Link from "next/link"
import { useRouter, useSearchParams } from "next/navigation"
import { useTranslations } from "next-intl"

import { DirectoriesView } from "@/components/directories/directories-view"
import { EndpointsDetailView } from "@/components/endpoints/endpoints-detail-view"
import { ArrowLeft, ExternalLink, semanticIcons } from "@/components/icons"
import { IPAddressesView } from "@/components/ip-addresses/ip-addresses-view"
import { AppErrorState } from "@/components/shared/feedback/app-error-state"
import { ContentHandoff } from "@/components/shared/loading/content-handoff"
import { useDetailShellReadySignal } from "@/components/shared/loading/detail-shell-ready-context"
import { HttpStatusBadge } from "@/components/shared/status/http-status-badge"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { VulnerabilitiesDetailView } from "@/components/vulnerabilities/vulnerabilities-detail-view"
import { useWebsite } from "@/hooks/use-websites"
import { createAppError } from "@/lib/errors/app-error"
import { normalizeError } from "@/lib/errors/normalize-error"
import { parseWebsiteName } from "@/lib/resource-name"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

import {
  WEBSITE_RELATION_DETAIL_SECTIONS,
  type WebsiteRelationDetailSection,
} from "./website-relation-detail-sections"
import {
  RelationScreenshotPreviewDialog,
  WebsiteResponseEvidence,
  WebsiteScreenshot,
  WebsiteTechnologySummary,
  toWebsiteEvidence,
  type RelationScreenshotPreview,
  type WebsiteEvidence,
} from "./website-relation-evidence-view"
import type { WebsiteAssetScope, WebSite } from "@/types/website.types"

const WebsiteIcon = semanticIcons.concept.website
const FingerprintIcon = semanticIcons.concept.fingerprint

export function WebsiteRelationDetailView({
  targetId,
  websiteId,
  section,
}: {
  targetId: number
  websiteId: number
  section: WebsiteRelationDetailSection
}) {
  const websiteQuery = useWebsite(websiteId)
  const isInitialLoading = websiteQuery.isLoading && !websiteQuery.data
  const detailShellReady = useDetailShellReadySignal(!isInitialLoading)
  const t = useTranslations("pages.targetDetail.relations.detail")

  if (detailShellReady?.deferInitialSkeleton && isInitialLoading) return null
  if (websiteQuery.error) return <AppErrorState error={normalizeError(websiteQuery.error)} onRetry={() => websiteQuery.refetch()} />

  const websiteRecord = websiteQuery.data
  const website = websiteRecord && belongsToTargetWebsiteRoute(websiteRecord, targetId, websiteId)
    ? toWebsiteEvidence(websiteRecord)
    : undefined
  if (!isInitialLoading && !website) {
    return (
      <AppErrorState
        error={createAppError("resource-not-found", { retryable: false })}
        title={t("notFound.title")}
        description={t("notFound.description", { id: websiteId })}
        actionHref={`/targets/${targetId}/websites/`}
        resourceLabel={t("backToList")}
      />
    )
  }

  return (
    <ContentHandoff
      owner="website-relation-detail-view-content"
      isLoading={isInitialLoading}
      skeleton={<WebsiteRelationDetailLoadingState />}
    >
      {website ? <ResolvedWebsiteRelationDetail targetId={targetId} website={website} section={section} /> : null}
    </ContentHandoff>
  )
}

export function belongsToTargetWebsiteRoute(website: WebSite, targetId: number, websiteId: number) {
  try {
    const resource = parseWebsiteName(website.resourceName ?? "")
    return resource.targetId === targetId && resource.websiteId === websiteId
  } catch {
    return false
  }
}

function WebsiteRelationDetailFrame({
  header,
  children,
  ariaHidden = false,
}: {
  header: ReactNode
  children: ReactNode
  ariaHidden?: boolean
}) {
  return (
    <div className="space-y-5" data-loading-slot="website-relation-detail-surface" aria-hidden={ariaHidden || undefined}>
      <div className="space-y-4" data-loading-slot="website-relation-detail-header">
        {header}
      </div>
      <div data-loading-slot="website-relation-detail-region">
        {children}
      </div>
    </div>
  )
}

function WebsiteRelationDetailIdentityFrame({
  identity,
  technologies,
}: {
  identity: ReactNode
  technologies: ReactNode
}) {
  return (
    <section className="min-w-0 border-b border-border pb-4" aria-labelledby="relation-detail-website-title">
      <div className="flex min-w-0 flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
        <div className="min-w-0 space-y-2">
          {identity}
        </div>
        <div className="min-w-0 space-y-2 lg:w-full lg:max-w-xl">
          {technologies}
        </div>
      </div>
    </section>
  )
}

function ResolvedWebsiteRelationDetail({
  targetId,
  website,
  section,
}: {
  targetId: number
  website: WebsiteEvidence
  section: WebsiteRelationDetailSection
}) {
  const t = useTranslations("pages.targetDetail.relations.detail")
  const [screenshotPreview, setScreenshotPreview] = useState<RelationScreenshotPreview | null>(null)
  const router = useRouter()
  const searchParams = useSearchParams()
  const basePath = `/targets/${targetId}/websites/${website.id}`
  const websiteListPath = `/targets/${targetId}/websites/`
  const restoresWebsiteListState = searchParams.get("returnTo") === "website-list"
  const returnContext = restoresWebsiteListState ? "?returnTo=website-list" : ""
  const sectionPaths: Record<WebsiteRelationDetailSection, string> = {
    overview: `${basePath}/${returnContext}`,
    "ip-addresses": `${basePath}/ip-addresses/${returnContext}`,
    urls: `${basePath}/urls/${returnContext}`,
    directories: `${basePath}/directories/${returnContext}`,
    vulnerabilities: `${basePath}/vulnerabilities/${returnContext}`,
  }

  const handleBackToList = (event: MouseEvent<HTMLAnchorElement>) => {
    if (!restoresWebsiteListState) return
    event.preventDefault()
    router.push(`${websiteListPath}?restoreWebsiteState=1`)
  }

  return (
    <>
      <WebsiteRelationDetailFrame
        header={(
          <>
            <Button variant="ghost" size="sm" render={<Link href={websiteListPath} onClick={handleBackToList} />} className="-ml-2 w-fit">
              <ArrowLeft data-icon="inline-start" />
              {t("backToList")}
            </Button>

            <WebsiteRelationDetailIdentityFrame
              identity={(
                <>
                  <div className="flex min-w-0 items-center gap-2">
                    <HttpStatusBadge statusCode={website.statusCode} size="default" />
                    <a href={website.url} target="_blank" rel="noreferrer" className={cn("flex min-w-0 items-center gap-1.5 text-primary hover:underline", textRole.pageTitle)}>
                      <WebsiteIcon aria-hidden="true" className="size-4 shrink-0" />
                      <span id="relation-detail-website-title" className="truncate">{website.url}</span>
                      <ExternalLink aria-hidden="true" className="size-3.5 shrink-0" />
                    </a>
                  </div>
                  <p className={cn("truncate", textRole.bodyStrong)}>{website.title || t("unknownTitle")}</p>
                  <p className={textRole.bodySubtle}>{website.host} · {website.webserver || t("unknownServer")}</p>
                </>
              )}
              technologies={(
                <>
                  <div className="flex items-center gap-1.5">
                    <FingerprintIcon aria-hidden="true" className="size-3.5 text-muted-foreground" />
                    <span className={textRole.metadataLabel}>{t("technology")}</span>
                  </div>
                  <WebsiteTechnologySummary technologies={website.tech} maxRows={3} />
                </>
              )}
            />

            <Tabs value={section}>
              <TabsList variant="content">
                {WEBSITE_RELATION_DETAIL_SECTIONS.map((item) => (
                  <TabsTrigger key={item} value={item} variant="content" render={<Link href={sectionPaths[item]} />}>
                    {t(`tabs.${item}`)}
                  </TabsTrigger>
                ))}
              </TabsList>
            </Tabs>
          </>
        )}
      >
        {section === "overview" ? (
          <WebsiteRelationOverview website={website} onPreviewScreenshot={setScreenshotPreview} />
        ) : (
          <WebsiteRelationAssetWorkspace targetId={targetId} website={website} section={section} />
        )}
      </WebsiteRelationDetailFrame>
      <RelationScreenshotPreviewDialog preview={screenshotPreview} onOpenChange={(open) => { if (!open) setScreenshotPreview(null) }} />
    </>
  )
}

function WebsiteRelationOverview({
  website,
  onPreviewScreenshot,
}: {
  website: WebsiteEvidence
  onPreviewScreenshot: (preview: RelationScreenshotPreview) => void
}) {
  return (
    <div className="grid min-w-0 items-stretch gap-4 xl:grid-cols-12">
      <div className="min-w-0 xl:col-span-5">
        <WebsiteScreenshot item={website} onPreview={onPreviewScreenshot} />
      </div>
      <div className="min-w-0 xl:col-span-7">
        <WebsiteResponseEvidence item={website} />
      </div>
    </div>
  )
}

function WebsiteRelationAssetWorkspace({
  targetId,
  website,
  section,
}: {
  targetId: number
  website: WebsiteEvidence
  section: Exclude<WebsiteRelationDetailSection, "overview">
}) {
  const websiteScope: WebsiteAssetScope = {
    url: website.url,
    host: website.host,
    readOnly: true,
  }

  if (section === "ip-addresses") return <IPAddressesView targetId={targetId} websiteScope={websiteScope} />
  if (section === "urls") return <EndpointsDetailView targetId={targetId} websiteScope={websiteScope} />
  if (section === "directories") return <DirectoriesView targetId={targetId} websiteScope={websiteScope} />
  return <VulnerabilitiesDetailView targetId={targetId} websiteScope={websiteScope} />
}

export function WebsiteRelationDetailLoadingState() {
  return (
    <WebsiteRelationDetailFrame
      ariaHidden
      header={(
        <>
          <Skeleton className="h-8 w-32" />
          <WebsiteRelationDetailIdentityFrame
            identity={(
              <>
                <div className="flex min-w-0 items-center gap-2">
                  <Skeleton className="h-7 w-10 shrink-0" />
                  <Skeleton className="h-7 w-2/3 max-w-xl" />
                </div>
                <Skeleton className="h-5 w-2/3 max-w-lg" />
                <Skeleton className="h-6 w-1/3 max-w-sm" />
              </>
            )}
            technologies={(
              <>
                <div className="flex items-center gap-1.5">
                  <Skeleton className="size-3.5 shrink-0" />
                  <Skeleton className="h-5 w-20" />
                </div>
                <div className="flex min-w-0 flex-wrap gap-1.5">
                  <Skeleton className="h-6 w-12" />
                  <Skeleton className="h-6 w-16" />
                  <Skeleton className="h-6 w-14" />
                  <Skeleton className="h-6 w-24" />
                </div>
              </>
            )}
          />
          <div className="flex h-9 items-end gap-4 border-b border-border">
            {Array.from({ length: 5 }, (_, index) => <Skeleton key={index} className="mb-2 h-4 w-16" />)}
          </div>
        </>
      )}
    >
      <Skeleton className="h-64 w-full" />
    </WebsiteRelationDetailFrame>
  )
}
