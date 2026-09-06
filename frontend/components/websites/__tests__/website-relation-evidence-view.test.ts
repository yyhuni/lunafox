import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

import { filterRelationEvidence, resolveVisibleTechnologyCount } from "../website-relation-evidence-view"

import type { WebsiteEvidence } from "../website-relation-evidence-view"

const source = readFileSync(path.resolve(process.cwd(), "components/websites/website-relation-evidence-view.tsx"), "utf8")
const zhMessages = JSON.parse(readFileSync(path.resolve(process.cwd(), "messages/zh.json"), "utf8"))
const enMessages = JSON.parse(readFileSync(path.resolve(process.cwd(), "messages/en.json"), "utf8"))

const records: WebsiteEvidence[] = [
  {
    id: 1,
    url: "https://admin.acme.com",
    host: "admin.acme.com",
    title: "Admin Panel",
    webserver: "nginx",
    contentType: "text/html",
    statusCode: 200,
    contentLength: 1024,
    tech: ["React", "Node.js"],
    vhost: false,
    responseHeaders: "server: nginx",
    responseBody: "",
    createdAt: "2026-07-21T00:00:00Z",
  },
  {
    id: 2,
    url: "https://api.acme.com",
    host: "api.acme.com",
    title: "Acme API",
    webserver: "caddy",
    contentType: "application/json",
    statusCode: 301,
    contentLength: 512,
    tech: ["Django", "PostgreSQL"],
    vhost: true,
    responseHeaders: "server: caddy",
    responseBody: "{}",
    createdAt: "2026-07-21T00:01:00Z",
  },
]

const emptyFilters = {
  statusCode: [],
  tech: [],
  webserver: [],
  contentType: [],
  vhost: [],
}

describe("filterRelationEvidence", () => {
  it("searches the same website identity fields shown in the relation list", () => {
    expect(filterRelationEvidence(records, "admin", emptyFilters).map((record) => record.id)).toEqual([1])
    expect(filterRelationEvidence(records, "API", emptyFilters).map((record) => record.id)).toEqual([2])
  })

  it("applies website facet values with OR inside a facet and AND across facets", () => {
    expect(filterRelationEvidence(records, "", {
      ...emptyFilters,
      tech: ["React", "Django"],
      statusCode: ["301"],
      vhost: ["true"],
    }).map((record) => record.id)).toEqual([2])
  })
})

describe("resolveVisibleTechnologyCount", () => {
  it("shows every technology when the available width fits all tags", () => {
    expect(resolveVisibleTechnologyCount(240, [60, 60, 60], [32, 32])).toBe(3)
  })

  it("reserves space for the remaining-count tag when the full list does not fit", () => {
    expect(resolveVisibleTechnologyCount(170, [60, 60, 60], [32, 32])).toBe(2)
  })

  it("uses the full remaining-count width when no technology tag fits", () => {
    expect(resolveVisibleTechnologyCount(34, [60, 60, 60], [24, 28, 34])).toBe(0)
  })

  it("uses the available card rows before collapsing technologies into a remaining-count tag", () => {
    expect(resolveVisibleTechnologyCount(170, [60, 60, 60, 60, 60], [24, 24, 24, 24, 24], 2)).toBe(4)
  })
})

describe("WebsiteRelationEvidenceLoadingState", () => {
  it("uses the resolved list frame instead of duplicating its structural shell", () => {
    const loadingStateSource = source.slice(source.indexOf("export function WebsiteRelationEvidenceLoadingState"))

    expect(loadingStateSource).toContain("<RelationEvidenceListFrame")
    expect(loadingStateSource).not.toContain("grid-cols-12 gap-4 border-b border-border bg-secondary")
    expect(loadingStateSource).toContain("<CompactPaginationSkeleton")
    expect(loadingStateSource).toContain("rowCount = 3")
    expect(loadingStateSource).toContain("Array.from({ length: rowCount }")
    expect(loadingStateSource).toContain("ResponseEvidencePanelLoadingState")
  })

  it("keeps the identity column's three-row technology budget aligned with its loading row", () => {
    expect(source).toContain("RELATION_EVIDENCE_IDENTITY_MIN_HEIGHT_PX = 211")
    expect(source).toContain("style={relationEvidenceIdentityMinHeightStyle}")
  })

  it("keeps query loading inside a handoff owner with paired stable slots", () => {
    expect(source).toContain('import { ContentHandoff } from "@/components/shared/loading/content-handoff"')
    expect(source).toContain('owner="target-website-evidence-view-content"')
    expect(source).toContain("isLoading={isInitialLoading}")
    expect(source).toContain('data-loading-slot="website-relation-evidence-surface"')
    expect(source).toContain('data-loading-slot="website-relation-evidence-list"')
    expect(source).not.toContain("if (isInitialLoading) return <WebsiteRelationEvidenceLoadingState />")
  })
})

describe("relation evidence inspection affordances", () => {
  it("exports its presentational list owner so read-only consumers can omit target-scoped details", () => {
    expect(source).toContain("export function RelationEvidenceListFrame")
    expect(source).toContain("export function RelationEvidenceToolbar")
    expect(source).toContain("export function WebsiteRelationEvidenceRow")
    expect(source).toContain("targetId?: number")
    expect(source).toContain("targetId === undefined ? undefined")
    expect(source).toContain("{detailHref ? (")
  })

  it("keeps target website facets enabled by default while allowing read-only consumers to fill the search row", () => {
    expect(source).toContain("showFacets = true")
    expect(source).toContain("inputWidthMode = \"fixed\"")
    expect(source).toContain("inputWidthMode={inputWidthMode}")
    expect(source).toContain("{showFacets ? <DataTableFacetPanel")
  })

  it("keeps response selection inside each content panel and lets technology summaries reveal excess tags", () => {
    expect(source).toContain('useTranslations("pages.targetDetail.relations")')
    expect(source).toContain('export function WebsiteResponseEvidence({ item }: { item: WebsiteEvidence })')
    expect(source).toContain('import { ResponseEvidencePanel, ResponseEvidencePanelLoadingState } from "@/components/shared/response-evidence"')
    expect(source).toContain("<ResponseEvidencePanel")
    expect(source).toContain("showMetadata")
    expect(source).toContain('location: t("location")')
    expect(source).toContain('responseSize: t("responseSize")')
    expect(source).not.toContain("ResponsePane")
    expect(source).not.toContain("onResponsePaneChange")
    expect(source).toContain('className="flex min-w-0 flex-wrap gap-1.5"')
    expect(source).toContain("resolveVisibleTechnologyCount")
    expect(source).toContain("ResizeObserver")
    expect(source).toContain("technologies.slice(0, visibleCount)")
    expect(source).toContain("remainingCount")
    expect(source).toContain("technologies.map((technology, index)")
    expect(source).toContain("viewRemainingTechnologies")
    expect(source).toContain('<DialogTitle>{t("technology")}</DialogTitle>')
    expect(source).toContain("<DialogContent className=\"sm:max-w-lg\">")
    expect(source).toContain("<RelationScreenshotPreviewDialog")
    expect(source).toContain("group-hover:scale-105")
    expect(source).toContain("--media-overlay-background")
  })

  it("keeps relation evidence copy localized in both supported locales", () => {
    const zhRelationMessages = zhMessages.pages.targetDetail.relations
    const enRelationMessages = enMessages.pages.targetDetail.relations

    expect(Object.keys(zhRelationMessages).sort()).toEqual(Object.keys(enRelationMessages).sort())
    expect(zhMessages.columns.endpoint.technologies).toBe("指纹")
    expect(zhMessages.columns.website.technologies).toBe("指纹")
    expect(zhMessages.search.fields.tech).toBe("指纹")
    expect(zhMessages.search.table.technologies).toBe("指纹")
    expect(enMessages.columns.endpoint.technologies).toBe("Technology")
    expect(enMessages.columns.website.technologies).toBe("Technology")
    expect(enMessages.search.fields.tech).toBe("Technology")
    expect(enMessages.search.table.technologies).toBe("Technology")
    expect(zhRelationMessages.technology).toBe("指纹")
    expect(enRelationMessages.technology).toBe("Technology")
    expect(zhRelationMessages.location).toBe("跳转")
    expect(enRelationMessages.location).toBe("Location")
    expect(source).not.toMatch(/[\u3400-\u9fff]/)
    expect(source).toContain('{item.host} · {item.webserver || t("unknownServer")}')
    expect(source).not.toContain('{item.host} · {item.webserver || t("unknownServer")} ·')
  })
})
