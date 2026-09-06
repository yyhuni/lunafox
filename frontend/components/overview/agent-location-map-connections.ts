import type { AgentLocationMapNode } from "@/components/overview/agent-location-map-data"
import type { WorldMapConnection, WorldMapPoint } from "@/components/overview/world-map"

function compareNodesByStableId(left: AgentLocationMapNode, right: AgentLocationMapNode) {
  return left.id.localeCompare(right.id, undefined, { numeric: true })
}

export function buildMapConnections(serverLocation: WorldMapPoint, nodes: AgentLocationMapNode[]): WorldMapConnection[] {
  // The map represents Server dispatch reach, so every eligible node gets one path instead of a sampled subset.
  return [...nodes]
    .filter((node) => node.status !== "offline")
    .sort(compareNodesByStableId)
    .map((node, index) => ({
      id: `server-network-${node.id}`,
      start: { lat: serverLocation.lat, lng: serverLocation.lng },
      end: { lat: node.latitude, lng: node.longitude },
      curveDirection: index % 2 === 0 ? 1 : -1,
      active: node.hasTaskLoadData && node.taskSlotsUsed > 0,
    }))
}
