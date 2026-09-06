import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/screenshots/screenshots-gallery-sections.tsx"), "utf8")
const sortControlSource = readFileSync(path.resolve(process.cwd(), "components/screenshots/screenshot-sort-control.tsx"), "utf8")
const pageSource = readFileSync(path.resolve(process.cwd(), "components/screenshots/screenshots-gallery.tsx"), "utf8")
const layoutSource = readFileSync(path.resolve(process.cwd(), "components/screenshots/screenshots-gallery-layout.ts"), "utf8")

describe("screenshots-gallery-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function ScreenshotsGalleryLoadingState")
    expect(source).toContain("export function ScreenshotsGalleryContent")
    expect(pageSource).toContain("<ScreenshotsGalleryLoadingState state={state} />")
    expect(source).not.toContain('from "./screenshots-gallery-skeleton"')
    expect(source).not.toContain("ScreenshotsGallerySkeleton")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
    expect(source).not.toContain("ScreenshotsGalleryErrorState")
  })

  it("routes HTTP status codes through the shared badge owner", () => {
    expect(source).toContain("HttpStatusBadge")
    expect(source).toContain('from "@/components/shared/status/http-status-badge"')
    expect(source).toContain('size="overlay"')
    expect(source).not.toContain("getHttpStatusBadgeVariant")
    expect(source).not.toContain("getHttpStatusBadgeClassName")
    expect(source).not.toContain("bg-green-500")
    expect(source).not.toContain("bg-blue-500")
    expect(source).not.toContain("bg-yellow-500")
    expect(source).not.toContain("bg-red-500")
    expect(source).not.toContain("text-blue-400")
  })

  it("top-aligns gallery toolbar actions with the search controls", () => {
    expect(source).toContain("SCREENSHOTS_GALLERY_TOOLBAR_CLASS")
    expect(layoutSource).toContain('export const SCREENSHOTS_GALLERY_TOOLBAR_CLASS = "flex gap-4 items-start justify-between"')
    expect(source).not.toContain('className="flex gap-4 items-center justify-between"')
  })

  it("shares gallery root, toolbar, and grid layout between content and loading state", () => {
    const sharedLayoutConstants = [
      "SCREENSHOTS_GALLERY_ROOT_CLASS",
      "SCREENSHOTS_GALLERY_TOOLBAR_CLASS",
      "SCREENSHOTS_GALLERY_TOOLBAR_CONTROLS_CLASS",
      "SCREENSHOTS_GALLERY_GRID_CLASS",
    ]

    expect(source).toContain('from "./screenshots-gallery-layout"')
    for (const constant of sharedLayoutConstants) {
      expect(layoutSource).toContain(`export const ${constant}`)
      expect(source).toContain(constant)
    }

    expect(source).not.toContain('className="space-y-4"')
    expect(source).not.toContain('className="gap-4 grid grid-cols-2 lg:grid-cols-4 md:grid-cols-3"')
  })

  it("uses shared control placeholders for the initial toolbar skeleton", () => {
    expect(source).toContain('from "@/components/shared/loading/search-toolbar-skeleton"')
    expect(source).toContain('from "@/components/shared/loading/action-skeleton"')
    expect(source).toContain("<SearchToolbarSkeleton")
    expect(source).toContain('toolbarDensity="compact"')
    expect(source).toContain('<ActionSkeleton size="sm" widthClassName="w-24" />')
    expect(source).toContain('<ActionSkeleton size="sm" widthClassName="w-28" />')
    expect(source).not.toContain('Skeleton className="h-10 w-64"')
  })

  it("hard-cuts the standalone gallery skeleton file from normal loading", () => {
    expect(source).toContain("function ScreenshotsGalleryToolbarLoadingState")
    expect(source).toContain("function ScreenshotsGalleryItemLoadingState")
    expect(source).toContain("SCREENSHOT_GALLERY_LOADING_ITEM_COUNT")
    expect(source).toContain("state.pagination.pageSize")
    expect(pageSource).not.toContain("ScreenshotsGallerySkeleton")
  })

  it("lets detail shells select stable route fallback cards and target-only action geometry", () => {
    expect(source).toContain("itemCount = SCREENSHOT_GALLERY_LOADING_ITEM_COUNT")
    expect(source).toContain("showSelectionAction = false")
    expect(source).toContain("{ itemCount?: number; showSelectionAction?: boolean } = {}")
    expect(source).toContain('throw new Error("ScreenshotsGalleryRouteFallback itemCount must be a positive integer.")')
    expect(source).toContain("Array.from({ length: itemCount })")
    expect(source).toContain("<ScreenshotsGalleryToolbarLoadingState showSelectionAction={showSelectionAction} />")
    expect(source).toContain('widthClassName="w-16"')
  })

  it("does not nest the checkbox trigger inside another button", () => {
    expect(source).toContain("onCheckedChange={() => state.toggleSelect(screenshot.id)}")
    expect(source).toContain('aria-label={state.t("selectScreenshot", { index: index + 1 })}')
    expect(source).toContain('aria-label={state.t("openScreenshot", { index: index + 1 })}')
    expect(source).not.toContain("Select screenshot")
    expect(source).not.toContain("Open screenshot")
    expect(source).not.toContain('className="absolute left-2 top-2 z-10"')
  })

  it("uses the shared selected-row action bar for target screenshot deletion", () => {
    expect(source).toContain('from "@/components/shared/data-table"')
    expect(source).toContain("<SelectedRowActionBar")
    expect(source).toContain("selectedCount={state.selectedIds.size}")
    expect(source).toContain("onClearSelection={state.clearSelection}")
    expect(source).not.toContain('state.tCommon("actions.delete")} ({state.selectedIds.size})')
  })

  it("uses project-owned visually hidden markup instead of raw Radix primitives", () => {
    expect(source).toContain('className="sr-only"')
    expect(source).not.toContain("@radix-ui/react-visually-hidden")
    expect(source).not.toContain("<VisuallyHidden")
  })

  it("renders ordinary URL search without SmartFilter or FOFA-style syntax affordance", () => {
    expect(source).toContain("SimpleSearchToolbar")
    expect(source).toContain("searchURL")
    expect(source).not.toContain("SmartFilter")
    expect(source).not.toContain("FOFA")
    expect(source).not.toContain("fofa")
    expect(source).not.toContain("field expression")
  })

  it("renders only statusCode faceted filtering from backend-scoped options", () => {
    expect(source).toContain("DataTableFacetedFilterGroup")
    expect(source).toContain("DataTableFacetedFilter")
    expect(source).toContain("state.statusCodeFilter")
    expect(source).toContain("state.statusCodeOptions")
    expect(source).toContain("state.handleStatusCodeFilterChange")
    expect(source).not.toContain("contentTypeFilter")
    expect(source).not.toContain("techFilter")
    expect(source).not.toContain("webserverFilter")
    expect(source).not.toContain("vhostFilter")
  })

  it("exposes backend sorting only for statusCode and createdAt", () => {
    expect(source).toContain("state.sorting")
    expect(source).toContain("state.handleSortingChange")
    expect(source).toContain("statusCode")
    expect(source).toContain("createdAt")
    expect(source).not.toContain('orderBy: "url"')
    expect(source).not.toContain('orderBy: "image"')
    expect(source).not.toContain('orderBy: "updatedAt"')
  })

  it("owns screenshot sorting through one compact menu instead of parallel field buttons", () => {
    expect(source).toContain('from "./screenshot-sort-control"')
    expect(source).toContain("<ScreenshotSortControl")
    expect(sortControlSource).toContain("DropdownMenu")
    expect(sortControlSource).toContain("排序方式")
    expect(sortControlSource).toContain("statusCode")
    expect(sortControlSource).toContain("createdAt")
    expect(sortControlSource).not.toContain("screenshot-sort-prototype")
  })
})
