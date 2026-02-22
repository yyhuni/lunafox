import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type ComponentType,
  type CSSProperties,
} from "react"
import {
  ConnectionLineType,
  MarkerType,
  Position,
  type Edge,
  type Node,
  type XYPosition,
} from "@xyflow/react"

import { IconCloud, IconCpu, IconServer } from "@/components/icons"

export type RoleKind = "server" | "agent" | "worker"

export type RoleNodeData = {
  title: string
  description: string
  kind: RoleKind
  icon: ComponentType<{ className?: string }>
  iconClassName?: string
}

export type GroupNodeData = {
  label: string
}

type RoleTemplate = {
  titleKey: string
  descKey: string
  icon: RoleNodeData["icon"]
  iconClassName?: string
}

type FlowRole = {
  id: string
  data: RoleNodeData
}

type EdgeKind = "bidirectional" | "local"

type FlowLink = {
  id: string
  source: string
  target: string
  label?: string
  sourceHandle?: string
  targetHandle?: string
  kind: EdgeKind
}

type FlowLayout = {
  positions: Record<string, XYPosition>
  groups: Record<
    string,
    {
      position: XYPosition
      width: number
      height: number
    }
  >
  height: number
}

const ROLE_START = { x: 40, y: 40 }
const NODE_WIDTH = 300
const NODE_HEIGHT = 72
const DEFAULT_NODE_GAP = 60
const MIN_HORIZONTAL_GAP = 12
const WORKERS_PER_AGENT = 2
const WORKER_STACK_GAP = 140
const MAX_HORIZONTAL_GAP = 160
const CANVAS_BOTTOM_PADDING = 160
const GROUP_ROW_GAP = 240
const GROUP_PADDING_X = 24
const GROUP_PADDING_TOP = 32
const GROUP_PADDING_BOTTOM = 24
const SOURCE_HANDLE_OFFSET = 6

export const FLOW_HANDLE_CLASS = "!h-2 !w-2 !border-0 !bg-transparent"

export function getSourceHandleStyle(position: Position): CSSProperties {
  switch (position) {
    case Position.Left:
      return { transform: `translateX(-${SOURCE_HANDLE_OFFSET}px)` }
    case Position.Right:
      return { transform: `translateX(${SOURCE_HANDLE_OFFSET}px)` }
    case Position.Top:
      return { transform: `translateY(-${SOURCE_HANDLE_OFFSET}px)` }
    case Position.Bottom:
      return { transform: `translateY(${SOURCE_HANDLE_OFFSET}px)` }
    default:
      return {}
  }
}

const LABEL_STYLE: CSSProperties = {
  fontSize: 11,
  fontWeight: 500,
  fill: "var(--muted-foreground)",
}

const LABEL_BG_STYLE: CSSProperties = {
  fill: "var(--background)",
  fillOpacity: 0.9,
}

const ARROW_MARKER = {
  type: MarkerType.ArrowClosed,
  width: 16,
  height: 16,
  color: "var(--primary)",
  markerUnits: "userSpaceOnUse",
  strokeWidth: 1.4,
}

const EDGE_PRESETS: Record<
  EdgeKind,
  {
    animated: boolean
    style: CSSProperties
    markerStart?: Edge["markerStart"]
    markerEnd?: Edge["markerEnd"]
  }
> = {
  bidirectional: {
    animated: false,
    style: {
      stroke: "var(--primary)",
      strokeWidth: 2.4,
      strokeOpacity: 0.85,
    },
    markerStart: ARROW_MARKER,
    markerEnd: ARROW_MARKER,
  },
  local: {
    animated: false,
    style: {
      stroke: "var(--primary)",
      strokeWidth: 2.1,
      strokeOpacity: 0.7,
      strokeDasharray: "7 5",
    },
    markerEnd: ARROW_MARKER,
  },
}

const ROLE_TEMPLATES: Record<RoleKind, RoleTemplate> = {
  server: {
    titleKey: "flowServerTitle",
    descKey: "flowServerDesc",
    icon: IconServer,
    iconClassName: "text-primary",
  },
  agent: {
    titleKey: "flowAgentTitle",
    descKey: "flowAgentDesc",
    icon: IconCloud,
    iconClassName: "text-[color:var(--color-chart-2)]",
  },
  worker: {
    titleKey: "flowWorkerTitle",
    descKey: "flowWorkerDesc",
    icon: IconCpu,
    iconClassName: "text-[color:var(--color-chart-1)]",
  },
}

const SERVER_ID = "server"
const AGENT_COUNT = 3

function buildIndexedIds(prefix: string, count: number) {
  return Array.from({ length: count }, (_, index) => `${prefix}-${index + 1}`)
}

const AGENT_IDS = buildIndexedIds("agent", AGENT_COUNT)
const WORKER_GROUPS = AGENT_IDS.map((agentId, agentIndex) => ({
  agentId,
  workerIds: buildIndexedIds(`worker-${agentIndex + 1}`, WORKERS_PER_AGENT),
}))
const VPS_GROUPS = WORKER_GROUPS.map((group, index) => ({
  ...group,
  groupId: `vps-${index + 1}`,
}))
const GROUP_BY_NODE_ID = VPS_GROUPS.reduce<Record<string, string>>((acc, group) => {
  acc[group.agentId] = group.groupId
  group.workerIds.forEach((workerId) => {
    acc[workerId] = group.groupId
  })
  return acc
}, {})

const LAYOUT_NODES = {
  serverId: SERVER_ID,
  agentIds: AGENT_IDS,
  workerGroups: VPS_GROUPS,
}

function clamp(value: number, min: number, max: number) {
  return Math.min(max, Math.max(min, value))
}

function resolveEdgeHandles(layout: FlowLayout, sourceId: string, targetId: string) {
  const source = layout.positions[sourceId]
  const target = layout.positions[targetId]
  if (!source || !target) return {}

  const dx = target.x - source.x
  const dy = target.y - source.y
  if (Math.abs(dx) >= Math.abs(dy)) {
    return dx >= 0
      ? { sourceHandle: "right-source", targetHandle: "left-target" }
      : { sourceHandle: "left-source", targetHandle: "right-target" }
  }

  return dy >= 0
    ? { sourceHandle: "bottom-source", targetHandle: "top-target" }
    : { sourceHandle: "top-source", targetHandle: "bottom-target" }
}

function useElementSize<T extends HTMLElement>() {
  const ref = useRef<T>(null)
  const [size, setSize] = useState({ width: 0, height: 0 })

  useEffect(() => {
    const element = ref.current
    if (!element) return

    const updateSize = () => {
      const rect = element.getBoundingClientRect()
      setSize({ width: rect.width, height: rect.height })
    }

    updateSize()

    if (typeof ResizeObserver !== "undefined") {
      const observer = new ResizeObserver(() => updateSize())
      observer.observe(element)
      return () => observer.disconnect()
    }

    window.addEventListener("resize", updateSize)
    return () => window.removeEventListener("resize", updateSize)
  }, [])

  return { ref, size }
}

function resolveLayout(
  width: number,
  layoutNodes: {
    serverId: string
    agentIds: string[]
    workerGroups: {
      agentId: string
      workerIds: string[]
      groupId: string
    }[]
  }
): FlowLayout {
  const { serverId, agentIds, workerGroups } = layoutNodes
  const maxWorkersPerAgent = Math.max(1, ...workerGroups.map((group) => group.workerIds.length))
  const horizontalPadding = ROLE_START.x * 2
  const minContentWidth = NODE_WIDTH * 3 + MIN_HORIZONTAL_GAP * 2
  const fallbackWidth = minContentWidth + horizontalPadding
  const safeWidth = width > 0 ? width : fallbackWidth
  const availableForGap = safeWidth - horizontalPadding - NODE_WIDTH * 3
  const responsiveGap = Number.isFinite(availableForGap) ? availableForGap / 2 : DEFAULT_NODE_GAP
  const gap = clamp(responsiveGap, MIN_HORIZONTAL_GAP, MAX_HORIZONTAL_GAP)
  const totalGraphWidth = NODE_WIDTH * 3 + gap * 2
  const startX = Math.max(ROLE_START.x, (safeWidth - totalGraphWidth) / 2)

  const positions: Record<string, XYPosition> = {}
  const stepX = NODE_WIDTH + gap
  const groupStep = Math.max(GROUP_ROW_GAP, maxWorkersPerAgent * WORKER_STACK_GAP)
  const totalHeight =
    (agentIds.length - 1) * groupStep + (maxWorkersPerAgent - 1) * WORKER_STACK_GAP
  const serverY = ROLE_START.y + Math.max(0, totalHeight) / 2

  positions[serverId] = { x: startX, y: serverY }

  workerGroups.forEach((group, index) => {
    const groupTop = ROLE_START.y + groupStep * index
    const workerCount = Math.max(1, group.workerIds.length)
    const agentY = groupTop + ((workerCount - 1) * WORKER_STACK_GAP) / 2

    positions[group.agentId] = {
      x: startX + stepX,
      y: agentY,
    }

    group.workerIds.forEach((workerId, workerIndex) => {
      positions[workerId] = {
        x: startX + stepX * 2,
        y: groupTop + WORKER_STACK_GAP * workerIndex,
      }
    })
  })

  const groups = workerGroups.reduce<FlowLayout["groups"]>((acc, group) => {
    const agentPosition = positions[group.agentId]
    if (!agentPosition) return acc

    const workerPositions = group.workerIds.map((workerId) => positions[workerId]).filter(Boolean)
    const minX = Math.min(agentPosition.x, ...workerPositions.map((position) => position.x))
    const maxX = Math.max(agentPosition.x, ...workerPositions.map((position) => position.x))
    const minY = Math.min(agentPosition.y, ...workerPositions.map((position) => position.y))
    const maxY = Math.max(agentPosition.y, ...workerPositions.map((position) => position.y))
    const position = {
      x: minX - GROUP_PADDING_X,
      y: minY - GROUP_PADDING_TOP,
    }
    const width = maxX - minX + NODE_WIDTH + GROUP_PADDING_X * 2
    const height = maxY - minY + NODE_HEIGHT + GROUP_PADDING_TOP + GROUP_PADDING_BOTTOM

    acc[group.groupId] = { position, width, height }
    return acc
  }, {})

  const bottomY = Math.max(
    ROLE_START.y + totalHeight,
    ...Object.values(groups).map((group) => group.position.y + group.height)
  )
  const height = Math.max(CANVAS_BOTTOM_PADDING, bottomY + CANVAS_BOTTOM_PADDING)

  return { positions, groups, height }
}

function buildRoles(t: (key: string) => string): FlowRole[] {
  const buildRole = (
    id: string,
    template: RoleTemplate,
    kind: RoleKind,
    suffix?: string | number
  ): FlowRole => ({
    id,
    data: {
      title: typeof suffix !== "undefined" ? `${t(template.titleKey)} ${suffix}` : t(template.titleKey),
      description: t(template.descKey),
      kind,
      icon: template.icon,
      iconClassName: template.iconClassName,
    },
  })

  return [
    buildRole(SERVER_ID, ROLE_TEMPLATES.server, "server"),
    ...AGENT_IDS.map((id, index) => buildRole(id, ROLE_TEMPLATES.agent, "agent", index + 1)),
    ...WORKER_GROUPS.flatMap((group, groupIndex) =>
      group.workerIds.map((workerId, workerIndex) =>
        buildRole(workerId, ROLE_TEMPLATES.worker, "worker", `${groupIndex + 1}-${workerIndex + 1}`)
      )
    ),
  ]
}

function buildLinks(t: (key: string) => string): FlowLink[] {
  const serverLinks = AGENT_IDS.map((agentId, index) => ({
    id: `${SERVER_ID}-${agentId}`,
    source: SERVER_ID,
    target: agentId,
    label: index === 0 ? t("flowLink") : undefined,
    kind: "bidirectional" as const,
  }))
  let workerLabelUsed = false
  const agentWorkerLinks: FlowLink[] = []

  WORKER_GROUPS.forEach((group) => {
    group.workerIds.forEach((workerId) => {
      agentWorkerLinks.push({
        id: `${group.agentId}-${workerId}`,
        source: group.agentId,
        target: workerId,
        label: workerLabelUsed ? undefined : t("flowWorkerLink"),
        kind: "local",
      })
      workerLabelUsed = true
    })
  })

  return [...serverLinks, ...agentWorkerLinks]
}

export function useArchitectureFlowState(t: (key: string) => string) {
  const { ref, size } = useElementSize<HTMLDivElement>()

  const roles = useMemo(() => buildRoles(t), [t])
  const links = useMemo(() => buildLinks(t), [t])
  const layout = useMemo(() => resolveLayout(size.width, LAYOUT_NODES), [size.width])

  const groupNodes: Node<GroupNodeData>[] = useMemo(
    () =>
      VPS_GROUPS.map((group, index) => {
        const bounds = layout.groups[group.groupId]
        return {
          id: group.groupId,
          type: "group",
          position: bounds?.position ?? { x: 0, y: 0 },
          data: { label: `${t("flowVpsTitle")} ${index + 1}` },
          style: {
            width: bounds?.width ?? NODE_WIDTH,
            height: bounds?.height ?? NODE_HEIGHT,
            border: "none",
            background: "transparent",
            boxShadow: "none",
          },
          draggable: false,
          selectable: false,
          zIndex: 0,
        }
      }),
    [layout.groups, t]
  )

  const roleNodes: Node<RoleNodeData>[] = useMemo(
    () =>
      roles.map((role) => {
        const parentId = GROUP_BY_NODE_ID[role.id]
        const basePosition = layout.positions[role.id] ?? { x: 0, y: 0 }
        const parent = parentId ? layout.groups[parentId] : undefined
        const position = parent
          ? {
              x: basePosition.x - parent.position.x,
              y: basePosition.y - parent.position.y,
            }
          : basePosition

        return {
          id: role.id,
          type: "role",
          position,
          data: role.data,
          parentId,
          extent: parentId ? "parent" : undefined,
          expandParent: Boolean(parentId),
          zIndex: parentId ? 1 : 2,
        }
      }),
    [roles, layout.positions, layout.groups]
  )

  const nodes = useMemo(() => [...groupNodes, ...roleNodes], [groupNodes, roleNodes])

  const edges: Edge[] = useMemo(() => {
    const buildEdge = (
      link: FlowLink,
      source: string,
      target: string,
      id: string,
      label?: string
    ): Edge => {
      const preset = EDGE_PRESETS[link.kind]
      const handleIds = resolveEdgeHandles(layout, source, target)
      const sourceHandle = link.sourceHandle ?? handleIds.sourceHandle
      const targetHandle = link.targetHandle ?? handleIds.targetHandle

      return {
        id,
        source,
        target,
        type: ConnectionLineType.SmoothStep,
        sourceHandle,
        targetHandle,
        animated: preset.animated,
        label,
        labelStyle: LABEL_STYLE,
        labelBgStyle: LABEL_BG_STYLE,
        labelBgPadding: [4, 8] as [number, number],
        labelBgBorderRadius: 4,
        style: preset.style,
        markerStart: preset.markerStart,
        markerEnd: preset.markerEnd,
      }
    }

    const result: Edge[] = []
    links.forEach((link) => {
      result.push(buildEdge(link, link.source, link.target, link.id, link.label))
    })
    return result
  }, [links, layout])

  return {
    ref,
    layoutHeight: layout.height,
    nodes,
    edges,
  }
}
