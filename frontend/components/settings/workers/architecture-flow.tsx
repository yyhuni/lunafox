"use client"

import { Background, Controls, ReactFlow } from "@xyflow/react"
import { useTranslations } from "next-intl"

import { architectureFlowNodeTypes } from "./architecture-flow-sections"
import { useArchitectureFlowState } from "./architecture-flow-state"

export function ArchitectureFlow() {
  const t = useTranslations("pages.workers")
  const state = useArchitectureFlowState(t)

  return (
    <div
      ref={state.ref}
      className="w-full rounded-lg border bg-gradient-to-br from-muted/20 to-background overflow-hidden"
      style={{ height: Math.max(state.layoutHeight, 560) }}
    >
      <ReactFlow
        nodes={state.nodes}
        edges={state.edges}
        nodeTypes={architectureFlowNodeTypes}
        fitView
        fitViewOptions={{
          padding: 0.24,
          minZoom: 0.45,
          maxZoom: 1.3,
        }}
        minZoom={0.4}
        maxZoom={1.8}
        nodesDraggable={false}
        nodesConnectable={false}
        elementsSelectable={false}
        panOnDrag
        zoomOnScroll
        zoomOnPinch
        preventScrolling
        proOptions={{ hideAttribution: true }}
      >
        <Background gap={16} size={1} color="var(--border)" style={{ opacity: 0.3 }} />
        <Controls
          showInteractive={false}
          showZoom
          showFitView
          position="top-right"
        />
      </ReactFlow>
    </div>
  )
}
