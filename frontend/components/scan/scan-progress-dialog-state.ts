import * as React from "react"
import { useTaskProgressLogs } from "@/hooks/use-task-progress-logs"
import type { ScanProgressData } from "@/components/scan/scan-progress-dialog-types"

type UseScanProgressDialogStateProps = {
  open: boolean
  data: ScanProgressData | null
}

export function useScanProgressDialogState({ open, data }: UseScanProgressDialogStateProps) {
  const [activeTab, setActiveTab] = React.useState<"stages" | "logs">("stages")

  const isRunning = React.useMemo(
    () => data?.status === "running" || data?.status === "initiated",
    [data?.status]
  )

  const { logs, loading: logsLoading } = useTaskProgressLogs({
    scanId: data?.id ?? 0,
    enabled: open && activeTab === "logs" && !!data?.id && Boolean(data.stages.some((stage) => stage.status !== "skipped")),
    pollingInterval: isRunning ? 3000 : 0,
  })

  return {
    activeTab,
    setActiveTab,
    logs,
    logsLoading,
  }
}
