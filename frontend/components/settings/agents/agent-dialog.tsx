"use client"

import { useTranslations } from "next-intl"
import {
  Dialog,
  DialogContent,
} from "@/components/ui/dialog"
import { Form } from "@/components/ui/form"
import { useAgentConfigDialogState } from "@/components/settings/agents/agent-dialog-state"
import {
  AgentConfigDialogHeader,
  AgentConfigFormFields,
  AgentConfigDialogFooter,
} from "@/components/settings/agents/agent-dialog-sections"
import type { Agent } from "@/types/agent.types"

interface AgentConfigDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  agentNode?: Agent | null
}

export function AgentConfigDialog({ open, onOpenChange, agentNode }: AgentConfigDialogProps) {
  const t = useTranslations("settings.agents")

  const {
    form,
    isPending,
    handleSubmit,
  } = useAgentConfigDialogState({
    open,
    onOpenChange,
    agentNode,
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[420px]">
        <AgentConfigDialogHeader t={t} />
        <Form {...form}>
          <form onSubmit={form.handleSubmit(handleSubmit)} className="space-y-4">
            <AgentConfigFormFields
              t={t}
              control={form.control}
            />
            <AgentConfigDialogFooter
              t={t}
              isPending={isPending}
            />
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
