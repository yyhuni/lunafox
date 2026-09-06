import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

import {
  WEBSITE_RELATION_DETAIL_SECTIONS,
  resolveWebsiteRelationDetailSection,
} from "../website-relation-detail-sections"
import { belongsToTargetWebsiteRoute } from "../website-relation-detail-view"

import type { WebSite } from "@/types/website.types"

const componentSource = readFileSync(path.resolve(process.cwd(), "components/websites/website-relation-detail-view.tsx"), "utf8")
const routeSource = readFileSync(path.resolve(process.cwd(), "app/targets/[id]/websites/[websiteId]/[[...section]]/page.tsx"), "utf8")
const layoutSource = readFileSync(path.resolve(process.cwd(), "app/targets/[id]/layout.tsx"), "utf8")
const evidenceSource = readFileSync(path.resolve(process.cwd(), "components/websites/website-relation-evidence-view.tsx"), "utf8")
const zhMessages = JSON.parse(readFileSync(path.resolve(process.cwd(), "messages/zh.json"), "utf8"))
const enMessages = JSON.parse(readFileSync(path.resolve(process.cwd(), "messages/en.json"), "utf8"))

describe("website relation detail route contract", () => {
  it("supports one overview and four route-backed collection views", () => {
    expect(WEBSITE_RELATION_DETAIL_SECTIONS).toEqual([
      "overview",
      "ip-addresses",
      "urls",
      "directories",
      "vulnerabilities",
    ])
    expect(resolveWebsiteRelationDetailSection(undefined)).toBe("overview")
    expect(resolveWebsiteRelationDetailSection("urls")).toBe("urls")
    expect(resolveWebsiteRelationDetailSection("unknown")).toBe("overview")
    expect(routeSource).toContain("section?: string[]")
    expect(routeSource).toContain("resolveWebsiteRelationDetailSection(section?.[0])")
  })

  it("preserves the Assets primary tab and selects a detail-specific loading state", () => {
    expect(layoutSource).toContain('pathname.includes("/websites")')
    expect(layoutSource).toContain("isWebsiteDetailRoute")
    expect(layoutSource).toContain('showSecondaryNav = primaryTab === "settings" || (primaryTab === "assets" && !isWebsiteDetailRoute)')
    expect(layoutSource).toContain("<WebsiteRelationDetailLoadingState />")
    expect(componentSource).toContain('<TabsList variant="content">')
    expect(componentSource).toContain('render={<Link href={sectionPaths[item]} />}')
  })

  it("keeps website-detail query loading inside a paired handoff surface", () => {
    expect(componentSource).toContain('import { ContentHandoff } from "@/components/shared/loading/content-handoff"')
    expect(componentSource).toContain('owner="website-relation-detail-view-content"')
    expect(componentSource).toContain("isLoading={isInitialLoading}")
    expect(componentSource).toContain('data-loading-slot="website-relation-detail-surface"')
    expect(componentSource).toContain('data-loading-slot="website-relation-detail-header"')
    expect(componentSource).toContain('data-loading-slot="website-relation-detail-region"')
    expect(componentSource.match(/<WebsiteRelationDetailIdentityFrame/g)).toHaveLength(2)
    expect(componentSource).not.toContain("if (isInitialLoading) return <WebsiteRelationDetailLoadingState />")
  })

  it("reuses target detail workspace owners instead of relation fixtures", () => {
    expect(componentSource).toContain('import { IPAddressesView } from "@/components/ip-addresses/ip-addresses-view"')
    expect(componentSource).toContain('import { EndpointsDetailView } from "@/components/endpoints/endpoints-detail-view"')
    expect(componentSource).toContain('import { DirectoriesView } from "@/components/directories/directories-view"')
    expect(componentSource).toContain('import { VulnerabilitiesDetailView } from "@/components/vulnerabilities/vulnerabilities-detail-view"')
    expect(componentSource).toContain('const websiteScope: WebsiteAssetScope')
    expect(componentSource).toContain('<IPAddressesView targetId={targetId} websiteScope={websiteScope} />')
    expect(componentSource).toContain('<EndpointsDetailView targetId={targetId} websiteScope={websiteScope} />')
    expect(componentSource).toContain('<DirectoriesView targetId={targetId} websiteScope={websiteScope} />')
    expect(componentSource).toContain('<VulnerabilitiesDetailView targetId={targetId} websiteScope={websiteScope} />')
    expect(componentSource).not.toContain("createWebsiteRelationDetailFixture")
    expect(componentSource).not.toContain("WebsiteRelationDetailFixture")
    expect(componentSource).not.toContain("TabsCountBadge")
  })

  it("uses a single Website resource and rejects a canonical name from another Target", () => {
    expect(componentSource).toContain("useWebsite(websiteId)")
    expect(componentSource).toContain("belongsToTargetWebsiteRoute(websiteRecord, targetId, websiteId)")
    expect(componentSource).toContain("parseWebsiteName(website.resourceName ?? \"\")")
    expect(componentSource).toContain("toWebsiteEvidence(websiteRecord)")
    expect(componentSource).not.toContain("useTargetWebSites(targetId")
    expect(componentSource).not.toContain("websiteRelations")
  })

  it("does not accept a Website response whose canonical Target differs from the route", () => {
    const website = {
      id: 1,
      resourceName: "targets/2/websites/1",
      url: "https://api.acme.com",
      host: "api.acme.com",
      location: "",
      title: "API",
      webserver: "nginx",
      contentType: "application/json",
      statusCode: 200,
      contentLength: 1,
      responseBody: "",
      tech: [],
      vhost: false,
      subdomain: "api.acme.com",
      createdAt: "2026-01-01T00:00:00Z",
    } satisfies WebSite

    expect(belongsToTargetWebsiteRoute(website, 1, 1)).toBe(false)
    expect(belongsToTargetWebsiteRoute({ ...website, resourceName: "targets/1/websites/1" }, 1, 1)).toBe(true)
    expect(belongsToTargetWebsiteRoute({ ...website, resourceName: "invalid" }, 1, 1)).toBe(false)
  })

  it("leaves relation-detail identity to its own header rather than the target breadcrumb", () => {
    expect(componentSource).not.toContain("TargetDetailBreadcrumbContext")
    expect(componentSource).not.toContain("setRelationBreadcrumb")
    expect(componentSource).not.toContain("WebsiteRelationBreadcrumbPrototype")
  })

  it("uses an explicit detail link instead of making the evidence row a link", () => {
    expect(evidenceSource).toContain('render={<Link href={detailHref} />}')
    expect(evidenceSource).toContain('{t("viewDetails")}')
    expect(evidenceSource).not.toContain("useRouter")
    expect(evidenceSource).not.toContain("router.push(detailHref)")
    expect(evidenceSource).not.toContain('role="link"')
  })

  it("uses the same bounded adaptive fingerprint summary as the evidence list", () => {
    expect(componentSource).toContain('<WebsiteTechnologySummary technologies={website.tech} maxRows={3} />')
    expect(componentSource).not.toContain('className="lg:justify-end"')
    expect(componentSource).toContain('className="min-w-0 space-y-2 lg:w-full lg:max-w-xl"')
    expect(componentSource).not.toContain("lg:text-right")
    expect(componentSource).toContain('className="flex items-center gap-1.5"')
    expect(componentSource).not.toContain("lg:justify-end")
  })

  it("restores website-list state only when the detail page came from the website list", () => {
    expect(evidenceSource).toContain('const detailHref = `/targets/${targetId}/websites/${item.id}/?returnTo=website-list`')
    expect(componentSource).toContain('import { useRouter, useSearchParams } from "next/navigation"')
    expect(componentSource).toContain('const restoresWebsiteListState = searchParams.get("returnTo") === "website-list"')
    expect(componentSource).toContain('router.push(`${websiteListPath}?restoreWebsiteState=1`)')
    expect(componentSource).toContain('href={websiteListPath} onClick={handleBackToList}')
    expect(componentSource).toContain('const returnContext = restoresWebsiteListState ? "?returnTo=website-list" : ""')
    expect(evidenceSource).toContain('const WEBSITE_EVIDENCE_RESTORE_QUERY = "restoreWebsiteState"')
    expect(evidenceSource).toContain("restoreWebsiteEvidenceState(targetId)")
    expect(evidenceSource).toContain("window.sessionStorage.setItem")
  })
})

describe("website relation detail localization", () => {
  it("keeps Chinese and English detail message structures aligned", () => {
    const zhDetail = zhMessages.pages.targetDetail.relations.detail
    const enDetail = enMessages.pages.targetDetail.relations.detail

    expect(Object.keys(zhDetail).sort()).toEqual(Object.keys(enDetail).sort())
    expect(Object.keys(zhDetail.tabs).sort()).toEqual(Object.keys(enDetail.tabs).sort())
    expect(Object.keys(zhDetail.columns).sort()).toEqual(Object.keys(enDetail.columns).sort())
    expect(zhDetail.technology).toBe("指纹")
    expect(enDetail.technology).toBe("Technology")
    expect(componentSource).not.toMatch(/[\u3400-\u9fff]/)
  })
})
