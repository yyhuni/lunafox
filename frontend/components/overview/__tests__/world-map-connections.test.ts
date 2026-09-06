import { describe, expect, it } from "vitest"

import { createBoundedConnectionPath } from "@/components/overview/world-map"

function parsePathCoordinates(path: string) {
  return [...path.matchAll(/-?\d+(?:\.\d+)?/g)].map(([value]) => Number(value))
}

describe("world-map connection paths", () => {
  it("connects far-apart points directly without wrapping through the opposite map edge", () => {
    const coordinates = parsePathCoordinates(createBoundedConnectionPath(
      { x: 214, y: 44 },
      { x: 32, y: 48 },
      248,
      100,
      1,
    ))

    expect(coordinates).toHaveLength(6)
    expect(coordinates.slice(0, 2)).toEqual([214, 44])
    expect(coordinates.slice(4, 6)).toEqual([32, 48])
    expect(coordinates[2]).toBeGreaterThan(2)
    expect(coordinates[2]).toBeLessThan(246)
    expect(coordinates[3]).toBeGreaterThan(2)
    expect(coordinates[3]).toBeLessThan(98)
  })

  it("keeps every curve coordinate within the map viewport", () => {
    const coordinates = parsePathCoordinates(createBoundedConnectionPath(
      { x: -30, y: -20 },
      { x: 320, y: 130 },
      248,
      100,
      -1,
    ))

    expect(coordinates).toHaveLength(6)
    expect(coordinates.filter((_, index) => index % 2 === 0).every((value) => value >= 2 && value <= 246)).toBe(true)
    expect(coordinates.filter((_, index) => index % 2 === 1).every((value) => value >= 2 && value <= 98)).toBe(true)
  })
})
