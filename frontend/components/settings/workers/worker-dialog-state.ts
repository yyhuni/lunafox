import React from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import * as z from "zod"
import { useUpdateAgentConfig } from "@/hooks/use-agents"
import type { Agent } from "@/types/agent.types"

const agentConfigFormSchema = z.object({
  maxTasks: z.coerce.number().int().min(1).max(20),
  cpuThreshold: z.coerce.number().int().min(1).max(100),
  memThreshold: z.coerce.number().int().min(1).max(100),
  diskThreshold: z.coerce.number().int().min(1).max(100),
})

export type AgentConfigFormValues = z.infer<typeof agentConfigFormSchema>

type UseAgentConfigDialogStateProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  agent?: Agent | null
}

export function useAgentConfigDialogState({
  open,
  onOpenChange,
  agent,
}: UseAgentConfigDialogStateProps) {
  const updateAgentConfig = useUpdateAgentConfig()

  const form = useForm<AgentConfigFormValues>({
    resolver: zodResolver(agentConfigFormSchema) as never,
    defaultValues: {
      maxTasks: 5,
      cpuThreshold: 85,
      memThreshold: 85,
      diskThreshold: 90,
    },
  })

  React.useEffect(() => {
    if (!open) return
    form.reset({
      maxTasks: agent?.maxTasks ?? 5,
      cpuThreshold: agent?.cpuThreshold ?? 85,
      memThreshold: agent?.memThreshold ?? 85,
      diskThreshold: agent?.diskThreshold ?? 90,
    })
  }, [open, agent, form])

  const handleSubmit = React.useCallback(async (values: AgentConfigFormValues) => {
    if (!agent) return
    try {
      await updateAgentConfig.mutateAsync({
        id: agent.id,
        data: values,
      })
      onOpenChange(false)
    } catch {
      // handled by hook
    }
  }, [agent, onOpenChange, updateAgentConfig])

  return {
    form,
    isPending: updateAgentConfig.isPending,
    handleSubmit,
  }
}
