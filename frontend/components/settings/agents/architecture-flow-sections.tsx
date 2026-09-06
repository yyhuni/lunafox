"use client"

import { memo, type ReactNode } from "react"

import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

import type {
  AgentGroupData,
  ArchitectureFlowData,
  RoleKind,
  RoleNodeData,
  EngineExecutionNodeData,
} from "./architecture-flow-state"
import {
  agentFrames,
  ArchitectureFlowViewport,
  CANVAS_HEIGHT,
  CANVAS_WIDTH,
  type FlowFrame,
  type FlowPoint,
  groupFrames,
  leftAnchor,
  leftCenter,
  rightAnchor,
  rightCenter,
  serverFrame,
  engineFrames,
} from "./architecture-flow-layout"

function createConnectorPath(from: FlowPoint, to: FlowPoint, viaX: number) {
  return `M${from.x} ${from.y} H${viaX} V${to.y} H${to.x}`
}

const serverOutput = rightCenter(serverFrame)
const controlTrunkX = serverOutput.x + 40
const engineTrunkX = engineFrames[0][0].x - 50
const resultEngineTrunkX = engineFrames[0][0].x - 24
const resultAgentTrunkX = serverOutput.x + 78
const resultStrokeOpacity = 0.64
const resultEngineAnchor = (engineFrame: FlowFrame) => leftAnchor(engineFrame, 16)
const resultAgentOutputBottomInset = 16
const resultAgentInputBottomInset = 12
const resultAgentOutputAnchor = (agentFrame: FlowFrame) => rightAnchor(
  agentFrame,
  agentFrame.height / 2 - resultAgentOutputBottomInset
)
const resultAgentInputAnchor = (agentFrame: FlowFrame) => leftAnchor(
  agentFrame,
  agentFrame.height / 2 - resultAgentInputBottomInset
)
const resultServerAnchor = rightAnchor(serverFrame, 32)
// Align the result arrow tip with the Server edge so the marker never covers the card.
const resultArrowTipRefX = 8

const controlConnectorPaths = agentFrames.map((agentFrame) =>
  createConnectorPath(serverOutput, leftCenter(agentFrame), controlTrunkX)
)
const engineConnectorPaths = agentFrames.flatMap((agentFrame, groupIndex) => {
  const from = rightCenter(agentFrame)

  return (engineFrames[groupIndex] ?? []).map((engineFrame) =>
    createConnectorPath(from, leftCenter(engineFrame), engineTrunkX)
  )
})
const resultEngineTapPaths = engineFrames.flatMap((frames) =>
  frames.map((engineFrame) => {
    const from = resultEngineAnchor(engineFrame)

    return `M${from.x} ${from.y} H${resultEngineTrunkX}`
  })
)
const resultEngineSpinePaths = agentFrames.map((agentFrame, groupIndex) => {
  const engineYList = (engineFrames[groupIndex] ?? []).map((engineFrame) =>
    resultEngineAnchor(engineFrame).y
  )
  const agentY = resultAgentOutputAnchor(agentFrame).y
  const fromY = Math.min(agentY, ...engineYList)
  const toY = Math.max(agentY, ...engineYList)

  return `M${resultEngineTrunkX} ${fromY} V${toY}`
})
const resultEngineToAgentPaths = agentFrames.map((agentFrame) => {
  const to = resultAgentOutputAnchor(agentFrame)

  return `M${resultEngineTrunkX} ${to.y} H${to.x}`
})
const resultAgentTapPaths = agentFrames.map((agentFrame) => {
  const from = resultAgentInputAnchor(agentFrame)

  return `M${from.x} ${from.y} H${resultAgentTrunkX}`
})
const resultAgentYList = agentFrames.map((agentFrame) => resultAgentInputAnchor(agentFrame).y)
const resultAgentSpinePath = `M${resultAgentTrunkX} ${Math.min(resultServerAnchor.y, ...resultAgentYList)} V${Math.max(resultServerAnchor.y, ...resultAgentYList)}`
const resultAgentToServerPaths = [`M${resultAgentTrunkX} ${resultServerAnchor.y} H${resultServerAnchor.x}`]

function roleIconShellClassName(kind: RoleKind) {
  if (kind === "agent") return "bg-success/15 text-success"
  if (kind === "engine") return "bg-info/15 text-info"
  return "bg-background text-primary"
}

function nodeStyle(frame: { x: number; y: number; width: number; height?: number }) {
  return {
    left: frame.x,
    top: frame.y,
    width: frame.width,
    ...(frame.height ? { minHeight: frame.height } : null),
  }
}

function agentNodeStyle(frame: FlowFrame) {
  return {
    left: frame.x,
    top: frame.y,
    width: frame.width,
  }
}

function FlowArrowMarkers() {
  return (
    <defs>
      <marker id="flow-arrow-agent" markerHeight="8" markerWidth="8" orient="auto" refX="7" refY="4">
        <path d="M0,0 L8,4 L0,8 Z" fill="var(--chart-2)" />
      </marker>
      <marker id="flow-arrow-agent-start" markerHeight="8" markerWidth="8" orient="auto-start-reverse" refX="1" refY="4">
        <path d="M0,0 L8,4 L0,8 Z" fill="var(--chart-2)" />
      </marker>
      <marker id="flow-arrow-engine" markerHeight="8" markerWidth="8" orient="auto" refX="7" refY="4">
        <path d="M0,0 L8,4 L0,8 Z" fill="var(--chart-1)" />
      </marker>
      <marker id="flow-arrow-engine-start" markerHeight="8" markerWidth="8" orient="auto-start-reverse" refX="1" refY="4">
        <path d="M0,0 L8,4 L0,8 Z" fill="var(--chart-1)" />
      </marker>
      <marker id="flow-arrow-result" markerHeight="8" markerWidth="8" orient="auto" refX={resultArrowTipRefX} refY="4">
        <path d="M0,0 L8,4 L0,8 Z" fill="var(--muted-foreground)" />
      </marker>
    </defs>
  )
}

function FlowConnectors() {
  return (
    <svg
      aria-hidden="true"
      className="pointer-events-none absolute inset-0"
      height={CANVAS_HEIGHT}
      viewBox={`0 0 ${CANVAS_WIDTH} ${CANVAS_HEIGHT}`}
      width={CANVAS_WIDTH}
    >
      <FlowArrowMarkers />
      {[...resultEngineTapPaths, ...resultEngineSpinePaths, ...resultAgentTapPaths, resultAgentSpinePath].map((path) => (
        <path
          key={`result-guide-${path}`}
          d={path}
          fill="none"
          stroke="var(--muted-foreground)"
          strokeLinecap="round"
          strokeOpacity={resultStrokeOpacity}
          strokeWidth="2"
        />
      ))}
      {resultEngineToAgentPaths.map((path) => (
        <path
          key={`engine-result-${path}`}
          d={path}
          fill="none"
          markerEnd="url(#flow-arrow-result)"
          stroke="var(--muted-foreground)"
          strokeLinecap="round"
          strokeOpacity={resultStrokeOpacity}
          strokeWidth="2"
        />
      ))}
      {resultEngineToAgentPaths.map((path) => (
        <path
          key={`engine-result-pulse-${path}`}
          className="architecture-flow-connector-result-pulse"
          d={path}
          fill="none"
          pathLength={1}
          stroke="var(--muted-foreground)"
          strokeLinecap="round"
          strokeWidth="2.5"
        />
      ))}
      {resultAgentToServerPaths.map((path) => (
        <path
          key={`agent-result-${path}`}
          d={path}
          fill="none"
          markerEnd="url(#flow-arrow-result)"
          stroke="var(--muted-foreground)"
          strokeLinecap="round"
          strokeOpacity={resultStrokeOpacity}
          strokeWidth="2"
        />
      ))}
      {resultAgentToServerPaths.map((path) => (
        <path
          key={`agent-result-pulse-${path}`}
          className="architecture-flow-connector-result-pulse"
          d={path}
          fill="none"
          pathLength={1}
          stroke="var(--muted-foreground)"
          strokeLinecap="round"
          strokeWidth="2.5"
        />
      ))}
      {controlConnectorPaths.map((path, index) => (
        <path
          key={`control-${path}`}
          className="architecture-flow-connector-control"
          d={path}
          fill="none"
          markerEnd="url(#flow-arrow-agent)"
          markerStart={index === 1 ? "url(#flow-arrow-agent-start)" : undefined}
          stroke="var(--chart-2)"
          strokeDasharray="6 6"
          strokeLinecap="round"
          strokeWidth="2"
        />
      ))}
      {engineConnectorPaths.map((path) => (
        <path
          key={`engine-${path}`}
          className="architecture-flow-connector-engine"
          d={path}
          fill="none"
          markerEnd="url(#flow-arrow-engine)"
          stroke="var(--chart-1)"
          strokeDasharray="6 6"
          strokeLinecap="round"
          strokeWidth="2"
        />
      ))}
    </svg>
  )
}

function RoleIcon({ node }: { node: RoleNodeData }) {
  const Icon = node.icon

  return (
    <span className={cn("flex size-9 shrink-0 items-center justify-center rounded-md", roleIconShellClassName(node.kind))}>
      <Icon className={cn("size-5", node.iconClassName)} />
    </span>
  )
}

function ServerNode({ node }: { node: RoleNodeData }) {
  return (
    <div
      data-architecture-flow-node="server"
      className="absolute rounded-md border bg-card/70 p-4 shadow-sm"
      style={nodeStyle(serverFrame)}
    >
      <RoleIcon node={node} />
      <div className="mt-3 space-y-1">
        <p className={textRole.bodyStrong}>{node.title}</p>
        <p className={textRole.helperText}>{node.description}</p>
      </div>
    </div>
  )
}

function AgentNode({ node, frame }: { node: RoleNodeData; frame: typeof agentFrames[number] }) {
  return (
    <div
      data-architecture-flow-node="agent"
      className="absolute rounded-md border bg-card/65 p-4 shadow-sm"
      style={agentNodeStyle(frame)}
    >
      <div className="flex items-start gap-3">
        <RoleIcon node={node} />
        <div className="min-w-0 space-y-1">
          <p className={textRole.bodyStrong}>{node.title}</p>
          <p className={textRole.helperText}>{node.description}</p>
        </div>
      </div>
    </div>
  )
}

function EngineExecutionNode({
  node,
  frame,
}: {
  node: EngineExecutionNodeData
  frame: FlowFrame
}) {
  return (
    <div
      data-architecture-flow-node="engine"
      className="absolute flex h-12 items-center gap-3 rounded-md border bg-card/65 px-3 shadow-sm"
      style={{ left: frame.x, top: frame.y, width: frame.width, height: frame.height }}
    >
      <RoleIcon node={node} />
      <div className="min-w-0 flex-1">
        <p className={cn(textRole.bodyStrong, "truncate")}>{node.title}</p>
        <p className={textRole.helperText}>{node.description}</p>
      </div>
    </div>
  )
}

function AgentGroup({
  group,
  index,
}: {
  group: AgentGroupData
  index: number
}) {
  const frame = groupFrames[index] ?? groupFrames[0]

  return (
    <div
      data-architecture-flow-node="vps-group"
      className="absolute rounded-lg border border-dashed bg-muted/10"
      style={{ left: frame.x, top: frame.y, width: frame.width, height: frame.height }}
    >
      <div className={cn("flex items-center gap-2 px-4 py-3", textRole.helperText)}>
        <span className={textRole.bodyStrong}>{group.label}</span>
        <span className="text-muted-foreground/60">|</span>
        <span>{group.ip}</span>
      </div>
    </div>
  )
}

export const ArchitectureFlowCanvas = memo(function ArchitectureFlowCanvas({
  fillAvailableSpace = false,
  flow,
  overlay,
}: {
  fillAvailableSpace?: boolean
  flow: ArchitectureFlowData
  overlay?: ReactNode
}) {
  return (
    <ArchitectureFlowViewport fillAvailableSpace={fillAvailableSpace} overlay={overlay}>
      {flow.groups.map((group, index) => (
        <AgentGroup key={group.id} group={group} index={index} />
      ))}
      <FlowConnectors />
      <ServerNode node={flow.server} />
      {flow.groups.map((group, index) => (
        <AgentNode
          key={group.agent.title}
          frame={agentFrames[index] ?? agentFrames[0]}
          node={group.agent}
        />
      ))}
      {flow.groups.flatMap((group, groupIndex) =>
        group.engineExecutions.map((engineExecution, engineIndex) => (
          <EngineExecutionNode
            key={engineExecution.id}
            frame={engineFrames[groupIndex]?.[engineIndex] ?? engineFrames[0][0]}
            node={engineExecution}
          />
        ))
      )}
    </ArchitectureFlowViewport>
  )
})
