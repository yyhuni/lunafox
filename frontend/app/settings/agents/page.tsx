import { AgentList } from "@/components/settings/agents/agent-list"
import {
  COMPACT_CONTENT_GUTTER_CLASS,
  COMPACT_FLEX_PAGE_SHELL_CLASS,
} from "@/components/shared/layout/page-shell-density"

export default function AgentPage() {
  return (
    <div className={COMPACT_FLEX_PAGE_SHELL_CLASS}>
      <div className={COMPACT_CONTENT_GUTTER_CLASS}>
        <AgentList />
      </div>
    </div>
  )
}
