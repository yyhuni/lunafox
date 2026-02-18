"use client"

import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Separator } from "@/components/ui/separator"
import { useLocale, useTranslations } from "next-intl"
import type { ScanProgressData } from "@/components/scan/scan-progress-dialog-types"
import {
  ScanProgressLogsPanel,
  ScanProgressStageList,
  ScanProgressSummary,
  ScanProgressTabs,
  ScanStatusIcon,
} from "@/components/scan/scan-progress-dialog-sections"
import { useScanProgressDialogState } from "@/components/scan/scan-progress-dialog-state"

interface ScanProgressDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  data: ScanProgressData | null
}

/** Dialog width constant */
const DIALOG_WIDTH = 'sm:max-w-[600px] sm:min-w-[550px]'

/**
 * Scan progress dialog
 */
export function ScanProgressDialog({
  open,
  onOpenChange,
  data,
}: ScanProgressDialogProps) {
  const t = useTranslations("scan.progress")
  const locale = useLocale()
  const { activeTab, setActiveTab, logs, logsLoading } = useScanProgressDialogState({ open, data })
  
  if (!data) return null

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className={`${DIALOG_WIDTH} transition-[opacity,transform] duration-200`}>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <ScanStatusIcon status={data.status} />
            {t("title")}
          </DialogTitle>
        </DialogHeader>

        <ScanProgressSummary data={data} locale={locale} t={t} />

        <Separator />

        <ScanProgressTabs activeTab={activeTab} onChange={setActiveTab} t={t} />

        {activeTab === "stages" ? (
          <ScanProgressStageList stages={data.stages} t={t} />
        ) : (
          <ScanProgressLogsPanel logs={logs} loading={logsLoading} />
        )}
      </DialogContent>
    </Dialog>
  )
}
export type { ScanProgressData } from "@/components/scan/scan-progress-dialog-types"
export { buildScanProgressData } from "@/components/scan/scan-progress-dialog-utils"
