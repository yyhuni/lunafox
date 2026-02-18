import { AgentNodeDesigns } from "@/components/prototypes/agent-node-designs"
import { BauhausPageHeader } from "@/components/common/bauhaus-page-header"

export default function AgentNodeDesignsPage() {
  return (
    <div className="flex h-full flex-col">
      <BauhausPageHeader 
        code="PRT-AGENT"
        subtitle="Prototypes"
        title="Agent Node & Metrics" 
        description="Exploration of UI components for Agent/Worker nodes and system metrics."
        showDescription
      />
      
      <div className="flex-1 overflow-auto p-6">
        <AgentNodeDesigns />
      </div>
    </div>
  )
}
