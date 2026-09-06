import React from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import * as z from "zod"
import { useUpdateAgentConfig } from "@/hooks/use-agents"
import type { Agent } from "@/types/agent.types"

const agentNodeConfigFormSchema = z.object({
  maxTasks: z.coerce.number().int().min(1),
  cpuThreshold: z.coerce.number().int().min(1).max(100),
  memThreshold: z.coerce.number().int().min(1).max(100),
  diskThreshold: z.coerce.number().int().min(1).max(100),
})

export type AgentConfigFormValues = z.infer<typeof agentNodeConfigFormSchema>

type UseAgentConfigDialogStateProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  agentNode?: Agent | null
}

export function useAgentConfigDialogState({
  open,
  onOpenChange,
  agentNode,
}: UseAgentConfigDialogStateProps) {
  const updateDistributedConfig = useUpdateAgentConfig()

  const form = useForm<AgentConfigFormValues>({
    resolver: zodResolver(agentNodeConfigFormSchema) as never,
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
      maxTasks: agentNode?.maxTasks ?? 5,
      cpuThreshold: agentNode?.cpuThreshold ?? 85,
      memThreshold: agentNode?.memThreshold ?? 85,
      diskThreshold: agentNode?.diskThreshold ?? 90,
    })
  }, [open, agentNode, form])

  const handleSubmit = React.useCallback(async (values: AgentConfigFormValues) => {
    if (!agentNode) return
    try {
      await updateDistributedConfig.mutateAsync({
        id: agentNode.id,
        data: values,
      })
      onOpenChange(false)
    } catch {
      // handled by hook
    }
  }, [agentNode, onOpenChange, updateDistributedConfig])

  return {
    form,
    isPending: updateDistributedConfig.isPending,
    handleSubmit,
  }
}
