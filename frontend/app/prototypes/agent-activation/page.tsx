import { notFound } from "next/navigation"

import { AgentActivationPrototype } from "@/components/settings/agents/agent-activation-prototype"

export default function AgentActivationPrototypePage() {
  if (process.env.NODE_ENV === "production" || process.env.NEXT_PUBLIC_USE_MOCK !== "true") {
    notFound()
  }

  return <AgentActivationPrototype />
}
