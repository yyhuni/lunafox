import { describe, expect, it } from "vitest"

import { allocateSemanticColumnWidths } from "../semantic-column-widths"

describe("allocateSemanticColumnWidths", () => {
  it("starts flex columns from their declared minimum widths and distributes surplus by weight", () => {
    const allocation = allocateSemanticColumnWidths([
      { id: "support", size: 80, minSize: 80, maxSize: 80, widthPolicy: { mode: "fixed" } },
      { id: "url", size: 200, minSize: 100, maxSize: 600, widthPolicy: { mode: "flex", flex: 2, fill: true } },
      { id: "title", size: 160, minSize: 100, maxSize: 600, widthPolicy: { mode: "flex", flex: 1 } },
    ], 580)

    expect(allocation.minimumWidth).toBe(280)
    expect(allocation.widthsById.get("support")).toBe(80)
    expect(allocation.widthsById.get("url")).toBeCloseTo(300)
    expect(allocation.widthsById.get("title")).toBeCloseTo(200)
  })

  it("redistributes width after a flex column reaches its maximum", () => {
    const allocation = allocateSemanticColumnWidths([
      { id: "url", size: 200, minSize: 100, maxSize: 300, widthPolicy: { mode: "flex", flex: 1, fill: true } },
      { id: "title", size: 200, minSize: 100, maxSize: 700, widthPolicy: { mode: "flex", flex: 1 } },
    ], 700)

    expect(allocation.widthsById.get("url")).toBe(300)
    expect(allocation.widthsById.get("title")).toBe(400)
  })

  it("lets the fill column receive remaining desktop width after all columns cap", () => {
    const allocation = allocateSemanticColumnWidths([
      { id: "url", size: 200, minSize: 100, maxSize: 250, widthPolicy: { mode: "flex", flex: 2, fill: true } },
      { id: "title", size: 160, minSize: 100, maxSize: 200, widthPolicy: { mode: "flex", flex: 1 } },
    ], 800)

    expect(allocation.widthsById.get("title")).toBe(200)
    expect(allocation.widthsById.get("url")).toBe(600)
  })

  it("keeps the minimum allocation when the container is too narrow", () => {
    const allocation = allocateSemanticColumnWidths([
      { id: "url", size: 200, minSize: 180, maxSize: 500, widthPolicy: { mode: "flex", flex: 1, fill: true } },
      { id: "createdAt", size: 160, minSize: 160, maxSize: 160, widthPolicy: { mode: "fixed" } },
    ], 200)

    expect(allocation.minimumWidth).toBe(340)
    expect(allocation.widthsById.get("url")).toBe(180)
    expect(allocation.widthsById.get("createdAt")).toBe(160)
  })

  it("excludes hidden columns from both the minimum and the allocation", () => {
    const allocation = allocateSemanticColumnWidths([
      { id: "url", size: 200, minSize: 160, maxSize: 500, widthPolicy: { mode: "flex", flex: 1, fill: true } },
      { id: "location", size: 180, minSize: 180, maxSize: 180, visible: false, widthPolicy: { mode: "fixed" } },
    ], 400)

    expect(allocation.minimumWidth).toBe(160)
    expect(allocation.widthsById.get("url")).toBe(400)
    expect(allocation.widthsById.has("location")).toBe(false)
  })
})
