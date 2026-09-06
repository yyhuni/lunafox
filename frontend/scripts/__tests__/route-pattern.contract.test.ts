import { describe, expect, it } from "vitest"
import path from "node:path"

import { instantiateRoutePattern } from "../route-pattern.mjs"
import { collectRouteInventory } from "../route-inventory.mjs"

const routeInventory = collectRouteInventory(path.resolve(process.cwd(), "app"))

describe("route pattern instantiation", () => {
  it("materializes the canonical website detail smoke target without literal dynamic syntax", () => {
    const route = instantiateRoutePattern("/targets/[id]/websites/[websiteId]/[[...section]]")

    expect(route).toBe("/targets/1/websites/1")
    expect(route).not.toMatch(/[\[\]]/)
  })

  it("materializes every current smoke inventory route without literal dynamic syntax", () => {
    for (const { routePattern } of routeInventory.routes) {
      expect(instantiateRoutePattern(routePattern)).not.toMatch(/[\[\]]/)
    }
  })

  it("uses deterministic fixtures for ordinary dynamic parameters", () => {
    expect(instantiateRoutePattern("/resources/[param]")).toBe("/resources/1")
  })

  it("materializes required catch-all segments and omits optional catch-all segments by default", () => {
    expect(instantiateRoutePattern("/docs/[...catchAll]")).toBe("/docs/1")
    expect(instantiateRoutePattern("/docs/[[...optionalCatchAll]]")).toBe("/docs")
  })

  it("supports explicit parameter values and encodes each catch-all path segment", () => {
    expect(
      instantiateRoutePattern("/targets/[id]/websites/[websiteId]/[[...section]]", {
        id: 7,
        websiteId: 12,
        section: ["request details", "headers"],
      })
    ).toBe("/targets/7/websites/12/request%20details/headers")
  })

  it("fails fast for malformed route segments and invalid required parameter values", () => {
    expect(() => instantiateRoutePattern("/targets/[id")).toThrow("Invalid route pattern segment")
    expect(() => instantiateRoutePattern("/targets/[id]", { id: "" })).toThrow("must not be empty")
    expect(() => instantiateRoutePattern("/docs/[...parts]", { parts: [] })).toThrow("must not be empty")
    expect(() => instantiateRoutePattern("/docs/[...parts]", { parts: {} })).toThrow("must be a string, number, or array")
  })
})
