
"use client"

import React, { memo, useMemo, useState } from "react"
import { MOCK_WORKFLOWS, FEATURE_LIST } from "../data"
import { ReactFlow, Background, Controls, Handle, Position, NodeProps, Edge, Node } from "@xyflow/react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Check, Play, AlertCircle } from "@/components/icons"
import Link from "next/link"

// Custom Node Component
const CustomFeatureNode = memo(function CustomFeatureNode({ data }: NodeProps) {
  return (
    <div className={`px-4 py-3 shadow-lg rounded-xl bg-card border-2 min-w-[180px] ${data.enabled ? 'border-primary' : 'border-muted opacity-50'}`}>
      <Handle type="target" position={Position.Top} className="!bg-muted-foreground" />
      <div className="flex items-center gap-2 mb-1">
        <span className="text-xl">{data.icon as string}</span>
        <div className="font-semibold text-sm">{data.label as string}</div>
      </div>
      <div className="text-[10px] text-muted-foreground">
        {data.enabled ? "Active" : "Skipped"}
      </div>
      <Handle type="source" position={Position.Bottom} className="!bg-muted-foreground" />
    </div>
  )
})

const nodeTypes = {
  feature: CustomFeatureNode,
}

export default function FeatureFlowDemo() {
  const [selectedWorkflowId, setSelectedWorkflowId] = useState(MOCK_WORKFLOWS[0].id)
  
  const selectedWorkflow = MOCK_WORKFLOWS.find(e => e.id === selectedWorkflowId) || MOCK_WORKFLOWS[0]

  const { nodes, edges } = useMemo(() => {
    const newNodes: Node[] = []
    const newEdges: Edge[] = []
    
    // Start Node
    newNodes.push({
      id: 'start',
      type: 'input',
      data: { label: 'Start Scan' },
      position: { x: 250, y: 0 },
      style: { background: '#10b981', color: 'white', border: 'none', borderRadius: '20px', width: '100px', textAlign: 'center' }
    })

    // Layout calculation vars
    let yPos = 100
    let lastNodeId = 'start'

    // Define a logical flow order
    const flowOrder = [
      'subdomain_discovery',
      'port_scan',
      ['site_scan', 'fingerprint_detect'], // Parallel group
      'directory_scan',
      ['screenshot', 'url_fetch'], // Parallel group
      'vuln_scan'
    ]

    flowOrder.forEach((step) => {
      if (Array.isArray(step)) {
        // Parallel nodes
        const enabledInGroup = step.filter(key => selectedWorkflow.features.includes(key))
        if (enabledInGroup.length > 0) {
          const totalWidth = enabledInGroup.length * 200
          const startX = 250 - (totalWidth / 2) + 100 // center align
          
          enabledInGroup.forEach((key, i) => {
            const feature = FEATURE_LIST.find(f => f.key === key)
            const nodeId = key
            
            newNodes.push({
              id: nodeId,
              type: 'feature',
              position: { x: startX + (i * 220) - 100, y: yPos },
              data: { 
                label: feature?.label, 
                icon: feature?.icon,
                enabled: true 
              }
            })
            
            newEdges.push({
              id: `e-${lastNodeId}-${nodeId}`,
              source: lastNodeId,
              target: nodeId,
              animated: true,
              style: { stroke: '#2563eb' }
            })
          })
          
          // Re-converge point (virtual or next node)
          // For simplicity in this demo, all parallel nodes connect to the next step's first node
          // Or we update lastNodeId to be an array? simplified: just take the first one or create a merge node
          lastNodeId = enabledInGroup[0] // Simplified linking
          yPos += 120
        }
      } else {
        // Single node step
        if (selectedWorkflow.features.includes(step)) {
          const feature = FEATURE_LIST.find(f => f.key === step)
          const nodeId = step
          
          newNodes.push({
            id: nodeId,
            type: 'feature',
            position: { x: 200, y: yPos }, // centered roughly
            data: { 
              label: feature?.label, 
              icon: feature?.icon,
              enabled: true 
            }
          })

          newEdges.push({
            id: `e-${lastNodeId}-${nodeId}`,
            source: lastNodeId,
            target: nodeId,
            animated: true,
            style: { stroke: '#2563eb' }
          })

          lastNodeId = nodeId
          yPos += 120
        }
      }
    })

    // End Node
    newNodes.push({
      id: 'end',
      type: 'output',
      data: { label: 'Report Generated' },
      position: { x: 250, y: yPos },
      style: { background: '#6366f1', color: 'white', border: 'none', borderRadius: '20px', width: '120px', textAlign: 'center' }
    })
    
    // Connect all leaf nodes to end if not already connected
    // Simplified: just connect last added node
    newEdges.push({
      id: `e-${lastNodeId}-end`,
      source: lastNodeId,
      target: 'end',
      animated: true,
      style: { stroke: '#2563eb' }
    })

    return { nodes: newNodes, edges: newEdges }
  }, [selectedWorkflow])

  return (
    <div className="flex h-full bg-background">
      {/* Sidebar List */}
      <div className="w-80 border-r flex flex-col bg-muted/10">
        <div className="p-4 border-b">
            <Link href="../demo" className="text-sm text-muted-foreground hover:text-foreground mb-4 block">← Back to Demos</Link>
          <h2 className="font-semibold text-lg">Scan Pipelines</h2>
          <p className="text-xs text-muted-foreground">Select an workflow to visualize its workflow</p>
        </div>
        <ScrollArea className="flex-1">
          <div className="p-3 space-y-2">
            {MOCK_WORKFLOWS.map(workflow => (
              <button type="button"
                key={workflow.id}
                onClick={() => setSelectedWorkflowId(workflow.id)}
                className={`w-full text-left p-3 rounded-lg border transition-[background-color,border-color,box-shadow] ${
                  selectedWorkflowId === workflow.id 
                    ? 'bg-background border-primary shadow-sm ring-1 ring-primary/20' 
                    : 'bg-transparent border-transparent hover:bg-muted'
                }`}
              >
                <div className="flex justify-between items-start mb-1">
                  <span className="font-medium text-sm">{workflow.name}</span>
                  {workflow.type === 'preset' && <Badge variant="secondary" className="text-[10px] px-1 h-4">Preset</Badge>}
                </div>
                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                  <span>{workflow.features.length} steps</span>
                  <span>•</span>
                  <span>{workflow.stats.avgTime}</span>
                </div>
              </button>
            ))}
          </div>
        </ScrollArea>
      </div>

      {/* Main Flow Area */}
      <div className="flex-1 relative">
        <div className="absolute top-4 left-4 z-10 bg-background/80 backdrop-blur p-2 rounded-lg border shadow-sm">
          <h3 className="font-semibold flex items-center gap-2">
            {selectedWorkflow.name}
            {selectedWorkflow.isValid ? (
                <Check className="h-4 w-4 text-green-500" />
            ) : (
                <AlertCircle className="h-4 w-4 text-amber-500" />
            )}
          </h3>
          <p className="text-xs text-muted-foreground max-w-md">{selectedWorkflow.description}</p>
        </div>

        <div className="absolute top-4 right-4 z-10">
            <Button size="sm" className="shadow-lg">
                <Play className="h-4 w-4 mr-2" />
                Run Pipeline
            </Button>
        </div>

        <ReactFlow 
          nodes={nodes}
          edges={edges}
          nodeTypes={nodeTypes}
          fitView
          attributionPosition="bottom-right"
        >
          <Background color="#999" gap={16} size={1} />
          <Controls />
        </ReactFlow>
      </div>
    </div>
  )
}
