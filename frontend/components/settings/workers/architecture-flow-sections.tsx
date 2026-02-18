"use client"

import { memo } from "react"
import { Handle, Position, type Node, type NodeProps } from "@xyflow/react"

import { BaseNode, BaseNodeHeader, BaseNodeHeaderTitle } from "@/components/base-node"
import { cn } from "@/lib/utils"

import {
  FLOW_HANDLE_CLASS,
  getSourceHandleStyle,
  type GroupNodeData,
  type RoleNodeData,
} from "./architecture-flow-state"

const RoleNode = memo(function RoleNode({ data }: NodeProps<Node<RoleNodeData>>) {
  const Icon = data.icon

  return (
    <BaseNode className="w-[300px] shadow-sm">
      <Handle
        type="source"
        position={Position.Left}
        id="left-source"
        className={FLOW_HANDLE_CLASS}
        style={getSourceHandleStyle(Position.Left)}
      />
      <Handle type="target" position={Position.Left} id="left-target" className={FLOW_HANDLE_CLASS} />
      <Handle type="target" position={Position.Right} id="right-target" className={FLOW_HANDLE_CLASS} />
      <Handle
        type="source"
        position={Position.Right}
        id="right-source"
        className={FLOW_HANDLE_CLASS}
        style={getSourceHandleStyle(Position.Right)}
      />
      <Handle
        type="source"
        position={Position.Top}
        id="top-source"
        className={FLOW_HANDLE_CLASS}
        style={getSourceHandleStyle(Position.Top)}
      />
      <Handle type="target" position={Position.Top} id="top-target" className={FLOW_HANDLE_CLASS} />
      <Handle type="target" position={Position.Bottom} id="bottom-target" className={FLOW_HANDLE_CLASS} />
      <Handle
        type="source"
        position={Position.Bottom}
        id="bottom-source"
        className={FLOW_HANDLE_CLASS}
        style={getSourceHandleStyle(Position.Bottom)}
      />
      <BaseNodeHeader className="bg-muted/30 items-start mb-0">
        <div className="flex items-start gap-2 flex-1">
          <div className="rounded-lg border bg-background p-1.5">
            <Icon className={cn("h-4 w-4", data.iconClassName)} />
          </div>
          <div className="flex flex-col gap-0.5">
            <BaseNodeHeaderTitle className="text-sm leading-none flex-none">
              {data.title}
            </BaseNodeHeaderTitle>
            <p className="text-[11px] text-muted-foreground truncate">{data.description}</p>
          </div>
        </div>
      </BaseNodeHeader>
    </BaseNode>
  )
})

const GroupNode = memo(function GroupNode({ data }: NodeProps<Node<GroupNodeData>>) {
  return (
    <div className="h-full w-full rounded-xl border border-dashed border-muted-foreground/30 bg-muted/10">
      <div className="px-3 pt-2 text-[11px] font-medium text-muted-foreground">{data.label}</div>
    </div>
  )
})

export const architectureFlowNodeTypes = {
  role: RoleNode,
  group: GroupNode,
}
