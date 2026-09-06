import { describe, expect, it } from "vitest"

import { buildMapConnections } from "@/components/overview/agent-location-map-connections"
import type { AgentLocationMapNode } from "@/components/overview/agent-location-map-data"

function makeNode(id: number, taskSlotsUsed = 0, hasTaskLoadData = true): AgentLocationMapNode {
  return {
    id: `agent-location-${id}`,
    resourceName: `agents/${id}`,
    name: `agent-${id}`,
    latitude: id,
    longitude: id * 10,
    status: "healthy",
    tone: "success",
    color: "var(--severity-low)",
    locationState: "current",
    sourceObservedIp: "1.1.1.1",
    taskSlotsUsed,
    hasTaskLoadData,
  }
}

describe("agent-location-map-connections", () => {
  it("keeps the Server radial network stable when the API result order changes", () => {
    const nodes = [makeNode(10), makeNode(2), makeNode(7), makeNode(1), makeNode(4), makeNode(9)]
    const first = buildMapConnections({ lat: 31.2304, lng: 121.4737 }, nodes)
    const second = buildMapConnections({ lat: 31.2304, lng: 121.4737 }, [...nodes].reverse())

    expect(first.map((connection) => connection.id)).toEqual(second.map((connection) => connection.id))
    expect(first).toHaveLength(nodes.length)
    expect(first.every((connection) => connection.start.lat === 31.2304 && connection.start.lng === 121.4737)).toBe(true)
    expect(first.map((connection) => connection.curveDirection)).toEqual([1, -1, 1, -1, 1, -1])
  })

  it("only marks links as active when an endpoint reports task load", () => {
    const connections = buildMapConnections({ lat: 31.2304, lng: 121.4737 }, [
      makeNode(1, 0, false),
      makeNode(2, 2),
      { ...makeNode(3, 0, false), status: "offline" },
    ])

    expect(connections).toHaveLength(2)
    expect(connections.some((connection) => connection.end.lat === 3)).toBe(false)
    expect(connections.some((connection) => connection.active)).toBe(true)
    expect(buildMapConnections({ lat: 31.2304, lng: 121.4737 }, [makeNode(1, 0, false), makeNode(2, 0, false)])
      .every((connection) => !connection.active)).toBe(true)
  })

  it("does not cap the number of eligible Server connections", () => {
    const nodes = Array.from({ length: 24 }, (_, index) => makeNode(index + 1))
    nodes[11] = { ...nodes[11], status: "offline" }

    const connections = buildMapConnections({ lat: 31.2304, lng: 121.4737 }, nodes)

    expect(connections).toHaveLength(23)
    expect(connections.every((connection) => connection.start.lat === 31.2304 && connection.start.lng === 121.4737)).toBe(true)
    expect(connections.map((connection) => connection.id)).not.toContain("server-network-agent-location-12")
    expect(new Set(connections.map((connection) => connection.id)).size).toBe(23)
  })
})
