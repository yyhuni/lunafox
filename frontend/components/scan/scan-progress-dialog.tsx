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
import { useLocalizedEngineNames } from "@/hooks/use-localized-engine-names"

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
  const engineIds = data
    ? [...data.plannedEngineIds, ...data.stages.map((stage) => stage.engineId)]
    : []
  const executedEngines = useLocalizedEngineNames(engineIds)
  
  if (!data) return null

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className={`${DIALOG_WIDTH} transition-[opacity,transform] duration-200`}>
        <DialogHeader>
          <DialogTitle className="flex gap-2 items-center">
            <ScanStatusIcon status={data.status} />
            {t("title")}
          </DialogTitle>
        </DialogHeader>

        {executedEngines.error ? (
          <p role="alert" className="text-sm text-destructive">{t("engineCatalogUnavailable")}</p>
        ) : executedEngines.isLoading ? (
          <p aria-live="polite" className="text-sm text-muted-foreground">{t("engineCatalogLoading")}</p>
        ) : (
          <>
            <ScanProgressSummary
              data={data}
              engineNames={data.plannedEngineIds.map((engineId) => executedEngines.engineNamesById.get(engineId)!)}
              locale={locale}
              t={t}
            />

            <Separator />

            <ScanProgressTabs activeTab={activeTab} onChange={setActiveTab} t={t} />

            {activeTab === "stages" ? (
              <ScanProgressStageList
                stages={data.stages}
                engineNamesById={executedEngines.engineNamesById}
                t={t}
              />
            ) : (
              <ScanProgressLogsPanel logs={logs} loading={logsLoading} />
            )}
          </>
        )}
      </DialogContent>
    </Dialog>
  )
}
export type { ScanProgressData } from "@/components/scan/scan-progress-dialog-types"
export { buildScanProgressData } from "@/components/scan/scan-progress-dialog-utils"
