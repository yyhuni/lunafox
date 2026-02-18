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
      className="w-full rounded-lg border bg-gradient-to-br from-muted/30 to-muted/10 overflow-hidden"
      style={{ height: state.layoutHeight }}
    >
      <ReactFlow
        nodes={state.nodes}
        edges={state.edges}
        nodeTypes={architectureFlowNodeTypes}
        fitView
        fitViewOptions={{
          padding: 0.12,
          minZoom: 0.6,
          maxZoom: 1,
        }}
        minZoom={0.6}
        maxZoom={1.5}
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
          showZoom={false}
          showFitView={false}
          position="top-right"
        />
      </ReactFlow>
    </div>
  )
}
