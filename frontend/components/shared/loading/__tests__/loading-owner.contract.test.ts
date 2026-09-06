import { describe, expect, it } from "vitest"

import {
  getLoadingOwnerAttributes,
  getLoadingStructureSlotAttributes,
  isLoadingLayer,
  LOADING_INTENTS,
  LOADING_LAYERS,
} from "@/components/shared/loading/loading-owner"

describe("loading owner metadata contract", () => {
  it("defines the approved loading layer vocabulary", () => {
    expect(LOADING_LAYERS).toEqual([
      "boot",
      "auth-shell",
      "app-shell",
      "route",
      "workspace",
      "section",
      "interaction",
      "compact-pending",
      "domain-status",
    ])
  })

  it("defines the approved loading intent vocabulary", () => {
    expect(LOADING_INTENTS).toEqual([
      "boot",
      "auth",
      "app-shell",
      "route",
      "data",
      "navigation",
      "interaction",
      "pending",
      "status",
    ])
  })

  it("recognizes valid layers without accepting arbitrary strings", () => {
    expect(isLoadingLayer("workspace")).toBe(true)
    expect(isLoadingLayer("not-a-layer")).toBe(false)
    expect(isLoadingLayer(null)).toBe(false)
  })

  it("returns stable data attributes for loading owners", () => {
    expect(
      getLoadingOwnerAttributes({
        owner: "organization-list-content",
        layer: "workspace",
        intent: "data",
      })
    ).toEqual({
      "data-loading-owner": "organization-list-content",
      "data-loading-layer": "workspace",
      "data-loading-intent": "data",
    })
  })

  it("fails fast when owner metadata is missing a real owner", () => {
    expect(() =>
      getLoadingOwnerAttributes({
        owner: " ",
        layer: "workspace",
        intent: "data",
      })
    ).toThrow("Loading owner requires a non-empty owner.")
  })

  it("requires named structure slots for loading geometry pairing", () => {
    expect(getLoadingStructureSlotAttributes("primary-content")).toEqual({
      "data-loading-slot": "primary-content",
    })
    expect(() => getLoadingStructureSlotAttributes("   ")).toThrow(
      "Loading structure slot requires a non-empty slot."
    )
  })
})
