import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(
  path.resolve(process.cwd(), "components/tools/engines/engine-installation-page.tsx"),
  "utf8"
)
const cardSource = readFileSync(
  path.resolve(process.cwd(), "components/tools/engines/engine-catalog-card.tsx"),
  "utf8"
)
const detailSource = readFileSync(
  path.resolve(process.cwd(), "components/tools/engines/engine-catalog-detail-drawer.tsx"),
  "utf8"
)
describe("engine-installation-page loading contract", () => {
  it("keeps catalog loading on the resolved page and card geometry", () => {
    expect(source).toContain("function EngineCatalogGridLoadingState")
    expect(source).toContain("function EngineCatalogLoadingState")
    expect(source).toContain("function EngineCatalogControls")
    expect(source).toContain("function EngineCatalogResults")
    expect(source).toContain("<PageHeader")
    expect(source).toContain("<SearchInput")
    expect(cardSource).toContain('data-slot="engine-card-loading-state"')
    expect(cardSource).toContain("ENGINE_CARD_HEADER_CLASS")
    expect(cardSource).toContain("ENGINE_CARD_CAPABILITIES_CLASS")
    expect(cardSource).toContain("ENGINE_CARD_FOOTER_CLASS")
    expect(cardSource).toContain("function EngineTextPlaceholder")
    expect(cardSource).toContain("roleClassName={textRole.panelTitle}")
    expect(cardSource).toContain('lineWidths={["w-full", "w-3/4"]}')
    expect(cardSource).toContain("roleClassName={textRole.caption}")
    expect(source).toContain("ContentHandoff")
    expect(source).toContain('owner="engine-catalog-content"')
    expect(source).toContain("isLoading={catalog.isPending}")
    expect(source).toContain('skeleton={<EngineCatalogLoadingState searchPlaceholder={t("searchPlaceholder")} installLabel={t("install")} />}')
    expect(source).not.toContain("catalogLoading")
  })

  it("keeps loading ownership inside the catalog ContentHandoff", () => {
    expect(source).not.toContain("getLoadingOwnerAttributes")
    expect(source).not.toContain("EngineInstallationPageLoadingState")
  })

  it("allows the scan configuration workspace to reuse the catalog without a second page header", () => {
    expect(source).toContain("export function EngineInstallationPage({ embedded = false }")
    expect(source).toContain("if (embedded) {")
    expect(source).toContain("return content")
    expect(source).toContain('code="TLS-04"')
  })

  it("pairs controls and results within the catalog owner", () => {
    for (const slot of [
      "engine-catalog-controls",
      "engine-catalog-grid",
      "engine-catalog-card-rhythm",
    ]) {
      expect(source).toContain(`getLoadingStructureSlotAttributes("${slot}")`)
    }

    expect(source).toContain("skeletonClassName=\"space-y-6\"")
    expect(source).toContain("contentClassName=\"space-y-6\"")
    expect(source).toContain("disabled={disabled}")
  })

  it("uses the real built-in catalog window for the initial card grid", () => {
    expect(source).toContain("const ENGINE_CATALOG_FIRST_FRAME_CARD_COUNT = 8")
    expect(source).toContain("Array.from({ length: ENGINE_CATALOG_FIRST_FRAME_CARD_COUNT }")
  })

  it("uses the shared install drawer and keeps replacement confirmation server-driven", () => {
    expect(source).toContain('from "@/components/shared/form-drawer"')
    expect(source).toContain("<FormDrawer")
    expect(source).toContain("getEngineReplacementConflict")
    expect(source).toContain("submitInstall(false)")
    expect(source).toContain("submitInstall(true)")
  })

  it("keeps engine detail hook-owned and inside the shared interaction drawer", () => {
    expect(source).toContain("<EngineCatalogDetailDrawer")
    expect(cardSource).toContain('aria-label={`${t("viewDetails")}: ${display.displayName}`}')
    expect(detailSource).toContain("useEngineCatalogDetail")
    expect(detailSource).toContain("<DetailDrawer")
    expect(detailSource).toContain('owner: "engine-catalog-detail"')
    expect(detailSource).toContain('layer: "interaction"')
    expect(detailSource).toContain("localizeEngineCatalogDetail")
    expect(detailSource).toContain("CopyButton")
  })
})
