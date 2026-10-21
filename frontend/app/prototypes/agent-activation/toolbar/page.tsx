import { notFound, redirect } from "next/navigation"

export default function AgentActivationToolbarPage() {
  if (process.env.NODE_ENV !== "development" || process.env.NEXT_PUBLIC_USE_MOCK !== "true") notFound()
  redirect("/prototypes/agent-activation/workspace/")
}
