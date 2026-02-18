import * as React from "react"
import { useScanLogs } from "@/hooks/use-scan-logs"
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

  const { logs, loading: logsLoading } = useScanLogs({
    scanId: data?.id ?? 0,
    enabled: open && activeTab === "logs" && !!data?.id,
    pollingInterval: isRunning ? 3000 : 0,
  })

  return {
    activeTab,
    setActiveTab,
    logs,
    logsLoading,
  }
}
