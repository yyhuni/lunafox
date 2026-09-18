"use client"

import * as React from "react"
import { usePathname } from "next/navigation"
import { useTranslations } from "next-intl"

import {
  clearPendingSuccessOperationId,
  hasShownUpgradeCompletion,
  markUpgradeCompletionShown,
  readPendingSuccessOperationId,
  useUpgradeOperation,
} from "@/hooks/use-version"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { semanticIcons } from "@/components/icons"
import { textRole } from "@/lib/typography"
import { scrollableFormDialogContentClassName } from "@/lib/ui/overlay-styles"
import { isSystemUpgradePathname } from "@/components/auth/upgrade-route-boundary"

function formatDuration(
  createdAt: string,
  completedAt: string | null | undefined,
  t: (key: string, params?: Record<string, number>) => string,
): string {
  const start = Date.parse(createdAt)
  const end = Date.parse(completedAt ?? "")
  if (!Number.isFinite(start) || !Number.isFinite(end) || end < start) return t("completion.durationUnknown")
  const seconds = Math.max(1, Math.round((end - start) / 1000))
  return seconds < 60
    ? t("completion.durationSeconds", { seconds })
    : t("completion.durationMinutes", { minutes: Math.max(1, Math.round(seconds / 60)) })
}

export function SystemUpgradeCompletionDialog() {
  const pathname = usePathname()
  const t = useTranslations("systemUpgrade")
  const [hydrated, setHydrated] = React.useState(false)
  const [pendingOperationId, setPendingOperationId] = React.useState<string | null>(null)
  const [open, setOpen] = React.useState(false)
  const acknowledgedRef = React.useRef<string | null>(null)
  const shouldObserve = hydrated && !isSystemUpgradePathname(pathname) && Boolean(pendingOperationId)
  const operation = useUpgradeOperation(pendingOperationId, { enabled: shouldObserve })

  React.useEffect(() => {
    setHydrated(true)
    setPendingOperationId(readPendingSuccessOperationId())
  }, [])

  React.useEffect(() => {
    const operationId = pendingOperationId
    const data = operation.data
    if (!shouldObserve || !operationId || data?.status !== "succeeded") return
    if (acknowledgedRef.current === operationId || hasShownUpgradeCompletion(operationId)) return
    acknowledgedRef.current = operationId
    markUpgradeCompletionShown(operationId)
    clearPendingSuccessOperationId(operationId)
    setOpen(true)
  }, [operation.data, pendingOperationId, shouldObserve])

  if (!shouldObserve || !operation.data || operation.data.status !== "succeeded") return null

  const completedOperation = operation.data
  const displayedVersion = completedOperation.currentVersion
  const migration = completedOperation.migrationStatus === "not_started"
    ? t("completion.migrationNotRun")
    : completedOperation.migrationStatus

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className={`${scrollableFormDialogContentClassName} sm:max-w-lg`}>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <semanticIcons.status.success className="size-5 text-success" aria-hidden="true" />
            {t("completion.title")}
          </DialogTitle>
          <DialogDescription>{t("completion.description")}</DialogDescription>
        </DialogHeader>
        <Alert>
          <semanticIcons.status.success aria-hidden="true" />
          <AlertDescription>{t("completion.versionVerified", { version: displayedVersion })}</AlertDescription>
        </Alert>
        <dl className="grid gap-3 sm:grid-cols-2">
          <div className="border-t border-border/70 pt-2">
            <dt className={textRole.metadataLabel}>{t("completion.currentVersion")}</dt>
            <dd className={textRole.metadataValueStrong}>{displayedVersion}</dd>
          </div>
          <div className="border-t border-border/70 pt-2">
            <dt className={textRole.metadataLabel}>{t("completion.targetVersion")}</dt>
            <dd className={textRole.metadataValueStrong}>{completedOperation.releaseVersion}</dd>
          </div>
          <div className="border-t border-border/70 pt-2">
            <dt className={textRole.metadataLabel}>{t("completion.duration")}</dt>
            <dd className={textRole.metadataValueStrong}>{formatDuration(completedOperation.createdAt, completedOperation.completedAt, t)}</dd>
          </div>
          <div className="border-t border-border/70 pt-2">
            <dt className={textRole.metadataLabel}>{t("completion.migration")}</dt>
            <dd className={textRole.metadataValueStrong}>{migration}</dd>
          </div>
          <div className="border-t border-border/70 pt-2">
            <dt className={textRole.metadataLabel}>{t("completion.cancelledWork")}</dt>
            <dd className={textRole.metadataValueStrong}>{t("completion.cancelledWorkValue", { scans: completedOperation.cancelledScanCount, tasks: completedOperation.cancelledTaskCount })}</dd>
          </div>
          <div className="border-t border-border/70 pt-2 sm:col-span-2">
            <dt className={textRole.metadataLabel}>{t("completion.agents")}</dt>
            <dd className={textRole.metadataValueStrong}>{t("completion.agentsValue", { ready: completedOperation.agentSummary.ready, expected: completedOperation.agentSummary.expected, missing: completedOperation.agentSummary.missing, unhealthy: completedOperation.agentSummary.unhealthy })}</dd>
          </div>
        </dl>
        <p className={textRole.bodySubtle}>{t("completion.noReleaseNotes")}</p>
        <DialogFooter>
          <Button onClick={() => setOpen(false)}>{t("completion.continue")}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
