import { notFound } from "next/navigation"
import { AgentList } from "@/components/settings/agents/agent-list"
import { COMPACT_CONTENT_GUTTER_CLASS, COMPACT_FLEX_PAGE_SHELL_CLASS } from "@/components/shared/layout/page-shell-density"

export default function AgentActivationWorkspacePage() {
  if (process.env.NODE_ENV !== "development" || process.env.NEXT_PUBLIC_USE_MOCK !== "true") notFound()
  return <div className={COMPACT_FLEX_PAGE_SHELL_CLASS}><div className={COMPACT_CONTENT_GUTTER_CLASS}><AgentList activationPreview activationPlacement="on-demand" /></div></div>
}
