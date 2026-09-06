import { AgentList } from "@/components/settings/agents/agent-list"

export default function AgentPage() {
  return (
    <div className="flex flex-1 flex-col gap-4 py-4 md:gap-6 md:py-6">
      <div className="px-4 lg:px-6">
        <AgentList />
      </div>
    </div>
  )
}
