import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"
import { collectRouteInventory } from "../route-inventory.mjs"

const source = readFileSync(path.resolve(process.cwd(), "scripts/build-route-todo.mjs"), "utf8")
const inventorySource = readFileSync(path.resolve(process.cwd(), "scripts/route-inventory.mjs"), "utf8")
const routeInventory = collectRouteInventory(path.resolve(process.cwd(), "app"))

describe("build-route-todo contract", () => {
  it("delegates public route discovery to the shared source inventory", () => {
    expect(source).toContain("from \"node:fs/promises\"")
    expect(source).toContain('from "./route-inventory.mjs"')
    expect(source).toContain("collectRouteInventory(appDir)")
    expect(source).toContain('include: "app/**/page.tsx"')
    expect(source).toContain("skippedRoutes")
    expect(inventorySource).toContain('const excludedPrefixes = Object.freeze(["/prototypes"])')
    expect(inventorySource).toContain("const serverRedirectAliases = new Map([")
  })

  it("uses canonical plural route prefixes when classifying priorities", () => {
    expect(inventorySource).toContain('routePattern.startsWith("/targets/")')
    expect(inventorySource).toContain('routePattern.startsWith("/organizations/")')
    expect(inventorySource).not.toContain('routePattern.startsWith("/target/")')
    expect(inventorySource).not.toContain('routePattern.startsWith("/organization/")')
  })

  it("accounts for every production page as either a smoke route or an explicit skip", () => {
    const smokeRoutePatterns = routeInventory.routes.map((route) => route.routePattern)
    const skippedRoutePatterns = routeInventory.skippedRoutes.map((route) => route.routePattern)
    const coveredRoutePatterns = [...smokeRoutePatterns, ...skippedRoutePatterns].sort((left, right) => left.localeCompare(right))

    expect(routeInventory.schemaVersion).toBe(2)
    expect(new Set(coveredRoutePatterns).size).toBe(coveredRoutePatterns.length)
    expect(coveredRoutePatterns).toHaveLength(routeInventory.summary.discovered)
    expect(routeInventory.summary).toMatchObject({
      discovered: coveredRoutePatterns.length,
      planned: smokeRoutePatterns.length,
      skipped: skippedRoutePatterns.length,
    })
    expect(routeInventory.routes.every((route) => route.status !== "skipped")).toBe(true)
    expect(routeInventory.routes.every((route) => route.sourcePath.endsWith("/page.tsx"))).toBe(true)
  })

  it("tracks legacy aliases as skipped server redirects while retaining their canonical smoke routes", () => {
    const redirectAliases = routeInventory.skippedRoutes.filter((route) => route.skipReason === "server-redirect-alias")

    expect(redirectAliases).toEqual(expect.arrayContaining([
      expect.objectContaining({
        id: "route_targets_id_relations",
        routePattern: "/targets/[id]/relations",
        status: "skipped",
        skipReason: "server-redirect-alias",
        redirectTarget: "/targets/[id]/websites/",
      }),
      expect.objectContaining({
        id: "route_targets_id_relations_websiteid_section",
        routePattern: "/targets/[id]/relations/[websiteId]/[[...section]]",
        status: "skipped",
        skipReason: "server-redirect-alias",
        redirectTarget: "/targets/[id]/websites/[websiteId]/[[...section]]/",
      }),
      expect.objectContaining({
        id: "route_scan_workflow",
        routePattern: "/scan/workflow",
        status: "skipped",
        skipReason: "server-redirect-alias",
        redirectTarget: "/scan/config/workflows/",
      }),
      expect.objectContaining({
        id: "route_tools_engines",
        routePattern: "/tools/engines",
        status: "skipped",
        skipReason: "server-redirect-alias",
        redirectTarget: "/scan/config/engines/",
      }),
    ]))
    expect(redirectAliases).toHaveLength(4)

    expect(routeInventory.routes.map((route) => route.routePattern)).toEqual(expect.arrayContaining([
      "/targets/[id]/websites/[websiteId]/[[...section]]",
      "/scan/config/workflows",
      "/scan/config/engines",
    ]))
  })
})
