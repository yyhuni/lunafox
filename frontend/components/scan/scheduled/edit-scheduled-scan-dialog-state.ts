import React from "react"
import { toast } from "sonner"
import { getConfigConflictMessage, isCronExpressionValid } from "@/lib/scheduled-scan-helpers"
import { useUpdateScheduledScan } from "@/hooks/use-scheduled-scans"
import { useTargets } from "@/hooks/use-targets"
import { useEngines } from "@/hooks/use-engines"
import type { ScheduledScan, UpdateScheduledScanRequest } from "@/types/scheduled-scan.types"
import type { ScanEngine } from "@/types/engine.types"
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
  const { data: enginesData } = useEngines()

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
  const [engineIds, setEngineIds] = React.useState<number[]>([])
  const [selectedTargetId, setSelectedTargetId] = React.useState<number | null>(null)
  const [cronExpression, setCronExpression] = React.useState("")

  React.useEffect(() => {
    if (scheduledScan && open) {
      setName(scheduledScan.name)
      setEngineIds(scheduledScan.engineIds || [])
      setSelectedTargetId(scheduledScan.targetId || null)
      setCronExpression(scheduledScan.cronExpression || "0 2 * * *")
    }
  }, [scheduledScan, open])

  const handleEngineToggle = React.useCallback((engineId: number, checked: boolean) => {
    setEngineIds((prev) => (
      checked ? [...prev, engineId] : prev.filter((id) => id !== engineId)
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
    if (engineIds.length === 0) {
      toast.error(t("form.scanEngineRequired"))
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
      engineIds: engineIds,
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
  }, [cronExpression, engineIds, name, onOpenChange, onSuccess, scheduledScan, selectedTargetId, t, updateScheduledScan])

  const targets: Target[] = targetsData?.targets || []
  const engines: ScanEngine[] = enginesData || []

  return {
    cronPresets,
    name,
    setName,
    engineIds,
    selectedTargetId,
    cronExpression,
    setCronExpression,
    targets,
    engines,
    isPending,
    handleEngineToggle,
    handleTargetSelect,
    handleSubmit,
  }
}
