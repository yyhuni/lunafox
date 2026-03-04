import React from "react"
import { toast } from "sonner"
import { getConfigConflictMessage, isCronExpressionValid } from "@/lib/scheduled-scan-helpers"
import { useUpdateScheduledScan } from "@/hooks/use-scheduled-scans"
import { useTargets } from "@/hooks/use-targets"
import { useWorkflows } from "@/hooks/use-workflows"
import type { ScheduledScan, UpdateScheduledScanRequest } from "@/types/scheduled-scan.types"
import type { ScanWorkflow } from "@/types/workflow.types"
import type { Target } from "@/types/target.types"

interface UseEditScheduledScanDialogStateProps {
  open: boolean
  scheduledScan: ScheduledScan | null
  onOpenChange: (open: boolean) => void
  onSuccess?: () => void
  t: (key: string, params?: Record<string, string | number | Date>) => string
}

export function useEditScheduledScanDialogState({
  open,
  scheduledScan,
  onOpenChange,
  onSuccess,
  t,
}: UseEditScheduledScanDialogStateProps) {
  const { mutate: updateScheduledScan, isPending } = useUpdateScheduledScan()
  const { data: targetsData } = useTargets()
  const { data: workflowsData } = useWorkflows()

  const cronPresets = React.useMemo(() => [
    { label: t("presets.everyMinute"), value: "* * * * *" },
    { label: t("presets.every5Minutes"), value: "*/5 * * * *" },
    { label: t("presets.everyHour"), value: "0 * * * *" },
    { label: t("presets.daily2am"), value: "0 2 * * *" },
    { label: t("presets.daily4am"), value: "0 4 * * *" },
    { label: t("presets.weekly"), value: "0 2 * * 1" },
    { label: t("presets.monthly"), value: "0 2 1 * *" },
  ], [t])

  const [name, setName] = React.useState("")
  const [workflowIds, setWorkflowIds] = React.useState<number[]>([])
  const [selectedTargetId, setSelectedTargetId] = React.useState<number | null>(null)
  const [cronExpression, setCronExpression] = React.useState("")

  React.useEffect(() => {
    if (scheduledScan && open) {
      setName(scheduledScan.name)
      setWorkflowIds(scheduledScan.workflowIds || [])
      setSelectedTargetId(scheduledScan.targetId || null)
      setCronExpression(scheduledScan.cronExpression || "0 2 * * *")
    }
  }, [scheduledScan, open])

  const handleWorkflowToggle = React.useCallback((workflowID: number, checked: boolean) => {
    setWorkflowIds((prev) => (
      checked ? [...prev, workflowID] : prev.filter((id) => id !== workflowID)
    ))
  }, [])

  const handleTargetSelect = React.useCallback((targetId: number) => {
    setSelectedTargetId((prev) => (prev === targetId ? null : targetId))
  }, [])

  const handleSubmit = React.useCallback(() => {
    if (!scheduledScan) return

    if (!name.trim()) {
      toast.error(t("form.taskNameRequired"))
      return
    }
    if (workflowIds.length === 0) {
      toast.error(t("form.scanWorkflowRequired"))
      return
    }
    if (scheduledScan.scanMode === "target" && !selectedTargetId) {
      toast.error(t("toast.selectTarget"))
      return
    }
    if (!isCronExpressionValid(cronExpression)) {
      toast.error(t("form.cronRequired"))
      return
    }

    const request: UpdateScheduledScanRequest = {
      name: name.trim(),
      workflowIds: workflowIds,
      cronExpression: cronExpression.trim(),
    }

    if (scheduledScan.scanMode === "target" && selectedTargetId) {
      request.targetId = selectedTargetId
    }

    updateScheduledScan(
      { id: scheduledScan.id, data: request },
      {
        onSuccess: () => {
          onOpenChange(false)
          onSuccess?.()
        },
        onError: (err: unknown) => {
          const conflictMessage = getConfigConflictMessage(err)
          if (conflictMessage !== null) {
            toast.error(t("toast.configConflict"), {
              description: conflictMessage,
            })
          }
        },
      }
    )
  }, [cronExpression, workflowIds, name, onOpenChange, onSuccess, scheduledScan, selectedTargetId, t, updateScheduledScan])

  const targets: Target[] = targetsData?.targets || []
  const workflows: ScanWorkflow[] = workflowsData || []

  return {
    cronPresets,
    name,
    setName,
    workflowIds,
    selectedTargetId,
    cronExpression,
    setCronExpression,
    targets,
    workflows,
    isPending,
    handleWorkflowToggle,
    handleTargetSelect,
    handleSubmit,
  }
}
