import Link from "next/link"
import {
  IconArrowUp,
  IconBrandGithub,
  IconBook,
  IconCheck,
  IconExternalLink,
  IconFileText,
  IconHeart,
  IconMessageReport,
  IconRefresh,
  semanticIcons,
} from "@/components/icons"
import { LunaFoxMark } from "@/components/brand/lunafox-mark"
import {
  AlertDialog,
  AlertDialogClose,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Separator } from "@/components/ui/separator"
import { textRole } from "@/lib/typography"
import type { ReleaseManifestSummary, UpgradeOperation } from "@/types/version.types"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

export function AboutDialogHeader({ t }: { t: TranslationFn }) {
  return <DialogHeader><DialogTitle>{t("title")}</DialogTitle></DialogHeader>
}

export function AboutDialogBranding({ t }: { t: TranslationFn }) {
  return (
    <div className="flex flex-col items-center py-2">
      <div className="mb-3 flex h-16 w-16 items-center justify-center"><LunaFoxMark className="h-12 w-12" /></div>
      <h2 className={textRole.panelTitle}>{t("productName")}</h2>
      <p className={textRole.bodySubtle}>{t("description")}</p>
    </div>
  )
}

interface AboutDialogVersionInfoProps {
  t: TranslationFn
  currentVersion: string
  candidate?: ReleaseManifestSummary
  hasUpdate?: boolean
  checkError: string | null
  isChecking: boolean
  isCreating: boolean
  canStartUpgrade?: boolean
  operation: { data?: UpgradeOperation; isReconnecting?: boolean; operationId?: string | null }
  onCheckUpdate: () => void
  onStartUpgrade: () => void
  onRetry: () => void
}

export function AboutDialogVersionInfo({
  t,
  currentVersion,
  candidate,
  hasUpdate,
  checkError,
  isChecking,
  isCreating,
  canStartUpgrade = false,
  operation,
  onCheckUpdate,
  onStartUpgrade,
  onRetry,
}: AboutDialogVersionInfoProps) {
  const operationStatus = operation.data?.status
  const isTerminalFailure = operationStatus === "failed" || operationStatus === "needs_recovery" || operationStatus === "needs_attention"
  return (
    <div className="space-y-3 rounded-lg border px-4 py-3">
      <div className="flex items-center justify-between gap-4">
        <span className="text-sm text-muted-foreground">{t("currentVersion")}</span>
        <span className="font-mono text-sm">{currentVersion}</span>
      </div>

      {candidate && (
        <div className="space-y-2 border-t pt-3">
          <div className="flex items-center justify-between gap-4">
            <span className="text-sm text-muted-foreground">{t("candidateVersion")}</span>
            <div className="flex items-center gap-2">
              <span className="font-mono text-sm">{candidate.releaseVersion}</span>
              {hasUpdate ? <Badge variant="info" className="gap-1"><IconArrowUp className="h-3 w-3" />{t("updateAvailable")}</Badge> : <Badge variant="success" className="gap-1"><IconCheck className="h-3 w-3" />{t("upToDate")}</Badge>}
            </div>
          </div>
          <div className="space-y-1 text-xs text-muted-foreground">
            <p>{t("manifestDigest")}: <code className="break-all font-mono">{candidate.manifestDigest}</code></p>
            <p>{t("maintenanceWindow", { minutes: candidate.maintenanceWindowMinutes })}</p>
            <p>{candidate.databaseMigration.hasDatabaseMigration ? t("migrationSummary", { type: candidate.databaseMigration.migrationType }) : t("noMigration")}</p>
          </div>
        </div>
      )}

      {checkError && <p className="text-sm text-destructive">{checkError}</p>}

      <div className="flex gap-2">
        <Button variant="outline" size="sm" className="flex-1" onClick={onCheckUpdate} disabled={isChecking || isCreating} loading={isChecking} loadingLabel={t("checking")}>
          <IconRefresh className="mr-2 h-4 w-4" />{t("checkUpdate")}
        </Button>
        {candidate && hasUpdate && !operation.data && (
          <Button size="sm" className="flex-1" onClick={onStartUpgrade} disabled={!canStartUpgrade || isCreating}>
            <semanticIcons.action.run className="mr-2 h-4 w-4" />{t("startUpgrade")}
          </Button>
        )}
      </div>

      {candidate && hasUpdate && !operation.data && (
        <Alert>
          <semanticIcons.status.warning className="h-4 w-4" />
          <AlertDescription>{candidate.databaseMigration.hasDatabaseMigration ? t("migrationRiskHint") : t("upgradeRiskHint")}</AlertDescription>
        </Alert>
      )}

      {operation.data && (
        <div className="space-y-3 border-t pt-3" data-testid="upgrade-operation-status">
          <div className="flex items-center justify-between gap-3">
            <span className="text-sm text-muted-foreground">{t("upgradeStatus")}</span>
            <Badge data-badge-type={operationStatus}>{t(`status.${operationStatus}`)}</Badge>
          </div>
          {operation.isReconnecting && <p className="text-sm text-muted-foreground">{t("reconnecting")}</p>}
          <p className="text-xs text-muted-foreground">{t("operationId")}: <code className="font-mono">{operation.data.operationId}</code></p>
          <p className="text-xs text-muted-foreground">{t("cancelledWork", { scans: operation.data.cancelledScanCount, tasks: operation.data.cancelledTaskCount })}</p>
          {operation.data.diagnostic && <p className="text-sm text-destructive">{operation.data.diagnostic}</p>}
          {isTerminalFailure && <Button variant="outline" size="sm" onClick={onRetry} disabled={isCreating}>{t("retryUpgrade")}</Button>}
        </div>
      )}
    </div>
  )
}

interface UpgradeConfirmationProps {
  t: TranslationFn
  open: boolean
  candidate?: ReleaseManifestSummary
  acknowledged: boolean
  pending: boolean
  onOpenChange: (open: boolean) => void
  onAcknowledgedChange: (checked: boolean) => void
  onConfirm: () => void
}

export function AboutDialogUpgradeConfirmation({
  t,
  open,
  candidate,
  acknowledged,
  pending,
  onOpenChange,
  onAcknowledgedChange,
  onConfirm,
}: UpgradeConfirmationProps) {
  if (!candidate) return null
  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t("confirmUpgradeTitle")}</AlertDialogTitle>
          <AlertDialogDescription>{t("confirmUpgradeDescription", { version: candidate.releaseVersion })}</AlertDialogDescription>
        </AlertDialogHeader>
        <div className="space-y-3 text-sm">
          <div className="rounded-md border p-3 text-muted-foreground">
            <p>{t("maintenanceWindow", { minutes: candidate.maintenanceWindowMinutes })}</p>
            <p>{candidate.databaseMigration.hasDatabaseMigration ? t("migrationRisk") : t("noMigration")}</p>
            <p>{t("tasksWillBeCancelled")}</p>
            <p>{t("noBackupRollback")}</p>
          </div>
          <label className="flex items-start gap-2">
            <Checkbox checked={acknowledged} onCheckedChange={(value) => onAcknowledgedChange(value === true)} />
            <span>{t("confirmAcknowledgement")}</span>
          </label>
        </div>
        <AlertDialogFooter>
          <AlertDialogClose variant="outline">{t("cancel")}</AlertDialogClose>
          <Button onClick={onConfirm} disabled={!acknowledged || pending} loading={pending} loadingLabel={t("startingUpgrade")}>
            {t("confirmAndUpgrade")}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}

export function AboutDialogLinks({ t }: { t: TranslationFn }) {
  return (
    <div className="grid grid-cols-2 gap-2">
      <Button variant="ghost" size="sm" className="justify-start" render={<a href="https://github.com/yyhuni/lunafox" target="_blank" rel="noopener noreferrer" />}><IconBrandGithub className="mr-2 h-4 w-4" />GitHub</Button>
      <Button variant="ghost" size="sm" className="justify-start" render={<a href="https://github.com/yyhuni/lunafox/releases" target="_blank" rel="noopener noreferrer" />}><IconFileText className="mr-2 h-4 w-4" />{t("changelog")}</Button>
      <Button variant="ghost" size="sm" className="justify-start" render={<a href="https://github.com/yyhuni/lunafox/issues" target="_blank" rel="noopener noreferrer" />}><IconMessageReport className="mr-2 h-4 w-4" />{t("feedback")}</Button>
      <Button variant="ghost" size="sm" className="justify-start" render={<a href="https://github.com/yyhuni/lunafox#readme" target="_blank" rel="noopener noreferrer" />}><IconBook className="mr-2 h-4 w-4" />{t("docs")}</Button>
      <Button variant="outline" size="sm" className="col-span-2 justify-center" render={<Link href="/settings/support/" />}><IconHeart className="mr-2 h-4 w-4" />{t("supportAuthor")}</Button>
    </div>
  )
}

export function AboutDialogFooter({ t }: { t: TranslationFn }) {
  return <p className="text-center text-xs text-muted-foreground">© 2026 {t("productName")} · GPL-3.0</p>
}

export { Separator }
