import Link from "next/link"
import {
  IconArrowUp,
  IconBrandGithub,
  IconBook,
  IconCheck,
  IconFileText,
  IconHeart,
  IconMessageReport,
  IconRefresh,
  ChevronRight,
  ExternalLink,
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
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"
import { DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Separator } from "@/components/ui/separator"
import { isFrontendOnlyUpgrade } from "@/hooks/use-version"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import type { ReleaseManifestSummary, UpgradeOperationFull } from "@/types/version.types"

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
  githubRepo?: string
  currentVersion: string
  candidate?: ReleaseManifestSummary
  hasUpdate?: boolean
  checkError: string | null
  isChecking: boolean
  isCreating: boolean
  isRetrying?: boolean
  canStartUpgrade?: boolean
  operation: { data?: UpgradeOperationFull; isActive?: boolean; isReconnecting?: boolean; operationId?: string | null }
  onCheckUpdate: () => void
  onStartUpgrade: () => void
  onRetry: () => void
  onViewStatus?: () => void
}

export function AboutDialogVersionInfo({
  t,
  githubRepo = "https://github.com/yyhuni/lunafox",
  currentVersion,
  candidate,
  hasUpdate,
  checkError,
  isChecking,
  isCreating,
  isRetrying = false,
  canStartUpgrade = false,
  operation,
  onCheckUpdate,
  onStartUpgrade,
  onRetry,
  onViewStatus = () => undefined,
}: AboutDialogVersionInfoProps) {
  const operationStatus = operation.data?.status
  const isTerminalFailure = operationStatus === "failed"
  const needsAttention = operationStatus === "needs_recovery" || operationStatus === "needs_attention"
  const releaseTag = candidate && (candidate.releaseVersion.startsWith("v") ? candidate.releaseVersion : `v${candidate.releaseVersion}`)
  const releaseHref = candidate && `${githubRepo.replace(/\/+$/, "")}/releases/tag/${encodeURIComponent(releaseTag ?? "")}`
  const frontendOnly = isFrontendOnlyUpgrade(operation.data)
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
          {hasUpdate ? (
            <div className="space-y-1 text-xs text-muted-foreground">
              <p>{t("manifestDigest")}: <code className="break-all font-mono">{candidate.manifestDigest}</code></p>
              <p>{t("maintenanceWindow", { minutes: candidate.maintenanceWindowMinutes })}</p>
              <p>{candidate.databaseMigration.hasDatabaseMigration ? t("migrationSummary", { type: candidate.databaseMigration.migrationType }) : t("noMigration")}</p>
              <Collapsible defaultOpen className="space-y-2 border-t border-border/60 pt-2">
                <CollapsibleTrigger
                  render={(
                    <Button type="button" variant="ghost" size="sm" layout="between" className="group text-left hover:text-foreground dark:hover:text-foreground" />
                  )}
                >
                  <span className={cn("min-w-0", textRole.bodyStrong)}>{t("releaseNotes")}</span>
                  <ChevronRight className="size-4 shrink-0 transition-transform duration-200 motion-reduce:transition-none group-data-[panel-open]:rotate-90" aria-hidden="true" />
                </CollapsibleTrigger>
                <CollapsibleContent className="space-y-2">
                  {candidate.releaseNotes ? (
                    <div data-testid="release-notes-body" className={cn("max-h-64 overflow-y-auto rounded-md border border-border/60 bg-muted/30 p-3", textRole.bodySubtle, "whitespace-pre-wrap break-words")}>{candidate.releaseNotes.body}</div>
                  ) : (
                    <p data-testid="release-notes-unavailable" className={textRole.helperText}>{t("releaseNotesUnavailable")}</p>
                  )}
                  <Button
                    type="button"
                    variant="link"
                    size="sm"
                    render={<a href={releaseHref} target="_blank" rel="noopener noreferrer" />}
                  >
                    <ExternalLink className="size-3.5" aria-hidden="true" />{t("viewRelease")}
                  </Button>
                </CollapsibleContent>
              </Collapsible>
            </div>
          ) : null}
        </div>
      )}

      {checkError && <p className="text-sm text-destructive">{checkError}</p>}

      <div className="flex gap-2">
        <Button variant="outline" size="sm" className="flex-1" onClick={onCheckUpdate} disabled={isChecking || isCreating} loading={isChecking} loadingLabel={t("checking")}>
          <IconRefresh className="mr-2 h-4 w-4" />{t("checkUpdate")}
        </Button>
        {candidate && hasUpdate && !operation.isActive && (
          <Button size="sm" className="flex-1" onClick={onStartUpgrade} disabled={!canStartUpgrade || isCreating}>
            <semanticIcons.action.run className="mr-2 h-4 w-4" />{t("startUpgrade")}
          </Button>
        )}
      </div>

      {candidate && hasUpdate && !operation.isActive && (
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
          {frontendOnly ? (
            <p className="text-xs text-muted-foreground">{t("frontendOnlyScope")}</p>
          ) : operation.data.workDisposition === "cancelled" ? (
            <p className="text-xs text-muted-foreground">{t("cancelledWork", { scans: operation.data.cancelledScanCount, tasks: operation.data.cancelledTaskCount })}</p>
          ) : (
            <p className="text-xs text-muted-foreground">{t(`workDisposition.${operation.data.workDisposition}`)}</p>
          )}
          {operation.data.diagnostic && <p className="text-sm text-destructive">{operation.data.diagnostic}</p>}
          {isTerminalFailure && <Button variant="outline" size="sm" onClick={onRetry} disabled={isRetrying} loading={isRetrying} loadingLabel={t("retryingUpgrade")}>{t("retryUpgrade")}</Button>}
          {needsAttention && <Button variant="outline" size="sm" onClick={onViewStatus}>{t("viewUpgradeStatus")}</Button>}
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
            <p>{t("scopeConfirmedByServer")}</p>
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
