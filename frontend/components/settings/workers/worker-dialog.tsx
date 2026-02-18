"use client"

import { useTranslations } from "next-intl"
import {
  Dialog,
  DialogContent,
} from "@/components/ui/dialog"
import { Form } from "@/components/ui/form"
import { useAgentConfigDialogState } from "@/components/settings/workers/worker-dialog-state"
import {
  AgentConfigDialogHeader,
  AgentConfigFormFields,
  AgentConfigDialogFooter,
} from "@/components/settings/workers/worker-dialog-sections"
import type { Agent } from "@/types/agent.types"

interface AgentConfigDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  agent?: Agent | null
}

export function AgentConfigDialog({ open, onOpenChange, agent }: AgentConfigDialogProps) {
  const t = useTranslations("settings.workers")
  const tCommon = useTranslations("common.actions")

  const {
    form,
    isPending,
    handleSubmit,
  } = useAgentConfigDialogState({
    open,
    onOpenChange,
    agent,
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
              tCommon={tCommon}
              isPending={isPending}
              onCancel={() => onOpenChange(false)}
            />
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
