"use client"

import * as React from "react"
import Link from "next/link"
import { useLocale, useTranslations } from "next-intl"
import { useRouter } from "next/navigation"

import {
  clearStoredUpgradeOperationId,
  getUpgradeErrorMessage,
  isUpgradeOperationTerminal,
  upgradeUserStageForStatus,
  useRetryUpgradeOperation,
  useStopUpgradeOperation,
  useUpgradeOperation,
} from "@/hooks/use-version"
import { UPGRADE_USER_STAGES, type UpgradeLogEntry, type UpgradeOperation, type UpgradeOperationStatus, type UpgradeUserStage } from "@/types/version.types"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { AlertDialog, AlertDialogClose, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Progress } from "@/components/ui/progress"
import { ScrollArea } from "@/components/ui/scroll-area"
import { semanticIcons } from "@/components/icons"
import { getLoadingOwnerAttributes } from "@/components/shared/loading/loading-owner"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

const STAGE_TIME_KEYS: Record<UpgradeUserStage, string[]> = {
  preparing: ["queued", "preflight"],
  stopping: ["stopping"],
  updating: ["updating", "migrating"],
  restarting: ["restarting"],
  verifying: ["agent_verifying", "verifying"],
  finished: ["succeeded", "failed", "needs_recovery", "needs_attention"],
}

function formatTimestamp(value: string | undefined, locale: string): string {
  if (!value || !Number.isFinite(Date.parse(value))) return "-"
  return new Intl.DateTimeFormat(locale === "zh" ? "zh-CN" : "en-US", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value))
}

function formatDuration(operation: UpgradeOperation, t: (key: string, params?: Record<string, number | string>) => string): string {
  const start = Date.parse(operation.createdAt)
  const end = Date.parse(operation.completedAt ?? operation.updatedAt)
  if (!Number.isFinite(start) || !Number.isFinite(end) || end < start) return t("summary.durationUnknown")
  const seconds = Math.max(1, Math.round((end - start) / 1000))
  if (seconds < 60) return t("summary.durationSeconds", { seconds })
  return t("summary.durationMinutes", { minutes: Math.max(1, Math.round(seconds / 60)) })
}

function stageTimestamp(operation: UpgradeOperation, stage: UpgradeUserStage): string | undefined {
  const values = STAGE_TIME_KEYS[stage]
    .map((key) => operation.stageTimes[key])
    .filter((value): value is string => Boolean(value))
    .sort()
  return values[0]
}

function highestObservedStageIndex(operation: UpgradeOperation): number {
  return UPGRADE_USER_STAGES.reduce((highest, stage, index) => {
    if (stage === "finished") return highest
    return stageTimestamp(operation, stage) ? Math.max(highest, index) : highest
  }, -1)
}

function progressValue(status: UpgradeOperationStatus | undefined): number {
  if (!status) return 0
  const stage = upgradeUserStageForStatus(status)
  const index = UPGRADE_USER_STAGES.indexOf(stage)
  if (index < 0) return 0
  if (stage === "finished") return 100
  return Math.max(8, Math.round((index / (UPGRADE_USER_STAGES.length - 1)) * 100))
}

function statusVariant(status: UpgradeOperationStatus | undefined): "default" | "info" | "success" | "warning" | "error" {
  if (status === "succeeded") return "success"
  if (status === "failed") return "error"
  if (status === "needs_recovery" || status === "needs_attention") return "warning"
  return "info"
}

function UpgradeLoadingOwner({ title, description }: { title: string; description: string }) {
  return (
    <main
      {...getLoadingOwnerAttributes({ owner: "system-upgrade-status", layer: "route", intent: "route" })}
      className="flex min-h-svh w-full items-center justify-center bg-background px-4 py-8 sm:px-8"
      data-testid="system-upgrade-loading"
    >
      <Card className="w-full max-w-lg" variant="compact">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <semanticIcons.status.running className="size-5 text-primary" aria-hidden="true" />
            {title}
          </CardTitle>
        </CardHeader>
        <CardContent><p className={textRole.bodySubtle}>{description}</p></CardContent>
      </Card>
    </main>
  )
}

function UpgradeStageTimeline({
  operation,
  t,
  locale,
}: {
  operation: UpgradeOperation
  t: (key: string, params?: Record<string, number | string>) => string
  locale: string
}) {
  const currentStage = upgradeUserStageForStatus(operation.status)
  const currentIndex = UPGRADE_USER_STAGES.indexOf(currentStage)
  const observedIndex = highestObservedStageIndex(operation)
  const terminal = isUpgradeOperationTerminal(operation.status)

  return (
    <section aria-labelledby="upgrade-stage-heading" className="space-y-4">
      <div className="flex flex-wrap items-end justify-between gap-2">
        <div>
          <h2 id="upgrade-stage-heading" className={textRole.sectionTitle}>{t("timeline.title")}</h2>
          <p className={textRole.bodySubtle}>{t(`timeline.statusDescription.${operation.status}`)}</p>
        </div>
        <span className={textRole.metadataLabel}>{t("timeline.phaseProgress")}</span>
      </div>
      {operation.status === "succeeded" || !isUpgradeOperationTerminal(operation.status) ? (
        <Progress
          value={progressValue(operation.status)}
          aria-label={t("timeline.phaseProgress")}
          aria-valuetext={t("timeline.stageCount", {
            current: Math.min(UPGRADE_USER_STAGES.indexOf(currentStage) + 1, UPGRADE_USER_STAGES.length),
            total: UPGRADE_USER_STAGES.length,
          })}
        />
      ) : null}
      <ol className="grid gap-2 sm:grid-cols-3 lg:grid-cols-6" aria-label={t("timeline.stageList")}>
        {UPGRADE_USER_STAGES.map((stage, index) => {
          const isCurrent = terminal ? stage === "finished" : index === currentIndex
          // Terminal states do not imply that every later phase ran. Use the
          // server's stage evidence so a stopped/failed operation cannot look
          // like a completed upgrade.
          const isComplete = operation.status === "succeeded"
            ? index < currentIndex
            : terminal
              ? stage !== "finished" && index <= observedIndex
              : index < currentIndex
          const StageIcon = isComplete
            ? semanticIcons.status.success
            : isCurrent
              ? semanticIcons.status.running
              : semanticIcons.status.unknown
          const timestamp = stageTimestamp(operation, stage)
          return (
            <li
              key={stage}
              data-stage={stage}
              data-stage-state={isCurrent ? "current" : isComplete ? "complete" : "pending"}
              className={cn(
                "min-w-0 border-l-2 px-3 py-2 sm:border-l-0 sm:border-t-2",
                isCurrent ? "border-primary bg-primary/5" : isComplete ? "border-success/60" : "border-border",
              )}
            >
              <div className="flex items-start gap-2">
                <StageIcon className={cn("mt-0.5 size-4 shrink-0", isCurrent ? "text-primary" : isComplete ? "text-success" : "text-muted-foreground")} aria-hidden="true" />
                <div className="min-w-0">
                  <p className={cn(textRole.bodyStrong, "break-words leading-snug")}>{t(`timeline.stages.${stage}`)}</p>
                  <p className={textRole.compactCaption}>{timestamp ? formatTimestamp(timestamp, locale) : t("timeline.pending")}</p>
                </div>
              </div>
            </li>
          )
        })}
      </ol>
    </section>
  )
}

function OperationFacts({ operation, t }: { operation: UpgradeOperation; t: (key: string, params?: Record<string, number | string>) => string }) {
  const facts = [
    [t("facts.currentVersion"), operation.currentVersion],
    [t("facts.targetVersion"), operation.releaseVersion],
    [t("facts.manifest"), operation.manifestId],
    [t("facts.migration"), operation.migrationStatus === "not_started" ? t("facts.notStarted") : operation.migrationStatus],
    [t("facts.cancelledWork"), t("facts.cancelledWorkValue", { scans: operation.cancelledScanCount, tasks: operation.cancelledTaskCount })],
    [t("facts.agents"), t("facts.agentsValue", { ready: operation.agentSummary.ready, expected: operation.agentSummary.expected, missing: operation.agentSummary.missing, unhealthy: operation.agentSummary.unhealthy })],
  ]
  return (
    <dl className="grid gap-3 sm:grid-cols-2">
      {facts.map(([label, value]) => (
        <div key={label} className="min-w-0 border-t border-border/70 pt-2">
          <dt className={textRole.metadataLabel}>{label}</dt>
          <dd className={cn(textRole.metadataValueStrong, "mt-1 break-words")}>{value}</dd>
        </div>
      ))}
    </dl>
  )
}

function logIcon(level: UpgradeLogEntry["level"]) {
  if (level === "error") return semanticIcons.status.failed
  if (level === "warn") return semanticIcons.status.warning
  return semanticIcons.status.unknown
}

const LOCALIZED_LOG_MESSAGES = new Set([
  "requestAccepted",
  "stoppingWork",
  "preflight",
  "updatingServices",
  "migratingDatabase",
  "restartingServices",
  "verifyingAgents",
  "verifyingSystem",
  "completed",
  "failed",
  "needsRecovery",
  "needsAttention",
  "workStopped",
  "diagnostic",
])

const LOCALIZED_LOG_STAGES = new Set<UpgradeOperationStatus>([
  "queued", "stopping", "preflight", "updating", "migrating", "restarting",
  "agent_verifying", "verifying", "succeeded", "failed", "needs_recovery", "needs_attention",
])

function upgradeLogMessage(entry: UpgradeLogEntry, t: (key: string, params?: Record<string, number | string>) => string): string {
  if (LOCALIZED_LOG_MESSAGES.has(entry.messageKey)) {
    return entry.messageKey === "diagnostic" ? entry.message : t(`logs.messages.${entry.messageKey}`)
  }
  return t("logs.messages.unknown")
}

function upgradeLogStage(entry: UpgradeLogEntry, t: (key: string, params?: Record<string, number | string>) => string): string {
  return LOCALIZED_LOG_STAGES.has(entry.stage as UpgradeOperationStatus) ? t(`status.${entry.stage}`) : entry.stage
}

function upgradeLogMetadataLabel(key: string, t: (key: string, params?: Record<string, number | string>) => string): string {
  if (key === "cancelledScans" || key === "cancelledTasks") return t(`logs.metadata.${key}`)
  return key
}

function UpgradeLogs({
  logs,
  locale,
  t,
}: {
  logs: UpgradeLogEntry[]
  locale: string
  t: (key: string, params?: Record<string, number | string>) => string
}) {
  const orderedLogs = React.useMemo(() => [...logs].sort((left, right) => Date.parse(left.timestamp) - Date.parse(right.timestamp)), [logs])

  return (
    <Card variant="compact">
      <CardHeader>
        <CardTitle>{t("logs.title")}</CardTitle>
        <CardDescription>{t("logs.description")}</CardDescription>
      </CardHeader>
      <CardContent>
        {orderedLogs.length === 0 ? (
          <p className={textRole.bodySubtle}>{t("logs.empty")}</p>
        ) : (
          <ScrollArea className="max-h-72 border-y border-border/70" type="always" contentClassName="min-w-0">
            <ol className="divide-y divide-border/70" aria-label={t("logs.title")}>
              {orderedLogs.map((entry, index) => {
                const LogIcon = logIcon(entry.level)
                const metadata = Object.entries(entry.metadata ?? {})
                return (
                  <li key={`${entry.timestamp}-${entry.messageKey}-${index}`} data-log-level={entry.level} className="flex min-w-0 gap-3 px-1 py-3">
                    <LogIcon className="mt-0.5 size-4 text-muted-foreground" aria-hidden="true" />
                    <time className={cn(textRole.compactCaption, "hidden w-36 shrink-0 sm:block")}>{formatTimestamp(entry.timestamp, locale)}</time>
                    <div className="min-w-0 flex-1 space-y-1">
                      <p className={textRole.bodyStrong}>{upgradeLogMessage(entry, t)}</p>
                      <p className={textRole.compactCaption}>{formatTimestamp(entry.timestamp, locale)} · {upgradeLogStage(entry, t)}</p>
                      {metadata.length > 0 ? (
                        <dl className="flex flex-wrap gap-x-3 gap-y-1">
                          {metadata.map(([key, value]) => (
                            <div key={key} className="flex min-w-0 gap-1">
                              <dt className={textRole.compactCaption}>{upgradeLogMetadataLabel(key, t)}:</dt>
                              <dd className={cn(textRole.compactCaption, "break-all")}>{value}</dd>
                            </div>
                          ))}
                        </dl>
                      ) : null}
                    </div>
                  </li>
                )
              })}
            </ol>
          </ScrollArea>
        )}
      </CardContent>
    </Card>
  )
}

export function SystemUpgradeStatus() {
  const t = useTranslations("systemUpgrade")
  const locale = useLocale()
  const router = useRouter()
  const [hydrated, setHydrated] = React.useState(false)
  const [stopConfirmOpen, setStopConfirmOpen] = React.useState(false)
  const operation = useUpgradeOperation(undefined, { enabled: hydrated })
  const retryMutation = useRetryUpgradeOperation()
  const stopMutation = useStopUpgradeOperation()

  React.useEffect(() => {
    setHydrated(true)
  }, [])

  const leaveUpgrade = React.useCallback(() => {
    if (operation.operationId) clearStoredUpgradeOperationId(operation.operationId)
    router.replace("/overview/")
  }, [operation.operationId, router])

  const retry = React.useCallback(() => {
    if (!operation.operationId || retryMutation.isPending) return
    retryMutation.mutate(operation.operationId)
  }, [operation.operationId, retryMutation])

  const stop = React.useCallback(() => {
    if (!operation.operationId || stopMutation.isPending) return
    stopMutation.mutate(operation.operationId, {
      onSuccess: () => setStopConfirmOpen(false),
    })
  }, [operation.operationId, stopMutation])

  if (!hydrated || operation.isResolving) {
    return <UpgradeLoadingOwner title={t("loading.title")} description={t("loading.description")} />
  }

  if (!operation.operationId && !operation.isError) {
    return (
      <main className="flex min-h-svh w-full items-center justify-center bg-background px-4 py-8 sm:px-8" data-testid="system-upgrade-empty">
        <Card className="w-full max-w-lg" variant="compact">
          <CardHeader><CardTitle>{t("empty.title")}</CardTitle><CardDescription>{t("empty.description")}</CardDescription></CardHeader>
          <CardFooter><Button render={<Link href="/overview/" />}><semanticIcons.navigation.overview aria-hidden="true" />{t("actions.enterSystem")}</Button></CardFooter>
        </Card>
      </main>
    )
  }

  if (!operation.data) {
    const reconnecting = operation.isReconnecting || operation.isError
    return (
      <main className="flex min-h-svh w-full items-center justify-center bg-background px-4 py-8 sm:px-8" data-testid="system-upgrade-reconnect">
        <Card className="w-full max-w-lg" variant="compact">
          <CardHeader><CardTitle className="flex items-center gap-2"><semanticIcons.status.unknown className="size-5 text-primary" aria-hidden="true" />{t(reconnecting ? "reconnecting.title" : "loading.title")}</CardTitle></CardHeader>
          <CardContent>
            <p className={textRole.bodySubtle}>{reconnecting ? t("reconnecting.description") : t("loading.description")}</p>
            {operation.error ? <p className="mt-3 text-sm text-destructive">{getUpgradeErrorMessage(operation.error)}</p> : null}
          </CardContent>
          <CardFooter className="gap-2">
            <Button onClick={() => void operation.refetch()} loading={operation.isFetching} loadingLabel={t("actions.refreshing")}><semanticIcons.action.refresh aria-hidden="true" />{t("actions.refresh")}</Button>
          </CardFooter>
        </Card>
      </main>
    )
  }

  const currentOperation = operation.data
  const terminal = isUpgradeOperationTerminal(currentOperation.status)
  const isSuccess = currentOperation.status === "succeeded"
  const isFailure = currentOperation.status === "failed"
  const isRecovery = currentOperation.status === "needs_recovery"
  const isAttention = currentOperation.status === "needs_attention"
  const retryable = isFailure || isAttention

  return (
    <main
      {...getLoadingOwnerAttributes({ owner: "system-upgrade-status", layer: "route", intent: "status" })}
      className="min-h-svh w-full bg-background px-4 py-6 sm:px-8 sm:py-10"
      data-testid="system-upgrade-status"
    >
      <div className="mx-auto flex w-full max-w-5xl flex-col gap-6">
        <header className="flex flex-wrap items-start justify-between gap-4">
          <div className="min-w-0">
            <p className={textRole.monoLabel}>UPGRADE</p>
            <h1 className={cn(textRole.pageTitleDisplay, "mt-2 text-2xl")}>{t("title")}</h1>
            <p className={cn(textRole.pageDescription, "mt-2 max-w-2xl")}>{t(terminal ? `statusDescription.${currentOperation.status}` : "description")}</p>
          </div>
          <Badge variant={statusVariant(currentOperation.status)} data-testid="system-upgrade-status-badge">
            {t(`status.${currentOperation.status}`)}
          </Badge>
        </header>

        {operation.isReconnecting ? (
          <Alert>
            <semanticIcons.status.unknown aria-hidden="true" />
            <AlertTitle>{t("reconnecting.title")}</AlertTitle>
            <AlertDescription>{t("reconnecting.description")}</AlertDescription>
          </Alert>
        ) : null}

        {isFailure || isRecovery || isAttention ? (
          <Alert variant={isFailure ? "destructive" : "default"}>
            {isFailure ? <semanticIcons.status.failed aria-hidden="true" /> : <semanticIcons.status.warning aria-hidden="true" />}
            <AlertTitle>{t(`outcomes.${currentOperation.status}.title`)}</AlertTitle>
            <AlertDescription>
              {currentOperation.diagnostic || t(`outcomes.${currentOperation.status}.description`)}
            </AlertDescription>
          </Alert>
        ) : null}

        <Card>
          <CardContent className="space-y-6 pt-6">
            <UpgradeStageTimeline operation={currentOperation} t={t} locale={locale} />
            <div className="grid gap-2 border-t border-border/70 pt-4 sm:grid-cols-2">
              <p className={textRole.metadataLabel}>{t("facts.lastUpdated")}: <span className={textRole.metadataValue}>{formatTimestamp(currentOperation.updatedAt, locale)}</span></p>
              <p className={cn(textRole.metadataLabel, "sm:text-right")}>{t("facts.operationId")}: <code className={cn(textRole.code, "break-all")}>{currentOperation.operationId}</code></p>
            </div>
          </CardContent>
        </Card>

        <Card variant="compact">
          <CardHeader><CardTitle>{t("facts.title")}</CardTitle><CardDescription>{t("facts.description")}</CardDescription></CardHeader>
          <CardContent><OperationFacts operation={currentOperation} t={t} /></CardContent>
        </Card>

        <UpgradeLogs logs={currentOperation.logs} locale={locale} t={t} />

        {isSuccess ? (
          <Alert>
            <semanticIcons.status.success aria-hidden="true" />
            <AlertTitle>{t("outcomes.succeeded.title")}</AlertTitle>
            <AlertDescription>{t("outcomes.succeeded.description", { version: currentOperation.releaseVersion, duration: formatDuration(currentOperation, t) })}</AlertDescription>
          </Alert>
        ) : null}

        <footer className="flex flex-wrap items-center justify-end gap-2 border-t border-border pt-4">
          {!isSuccess && !isRecovery && !isAttention && !isFailure ? (
            <Button variant="outline" onClick={() => void operation.refetch()} loading={operation.isFetching} loadingLabel={t("actions.refreshing")}>
              <semanticIcons.action.refresh aria-hidden="true" />{t("actions.refresh")}
            </Button>
          ) : null}
          {!terminal ? (
            <Button variant="outline" onClick={() => setStopConfirmOpen(true)} disabled={stopMutation.isPending}>
              <semanticIcons.action.stop aria-hidden="true" />{t("actions.stop")}
            </Button>
          ) : null}
          {retryable ? (
            <Button onClick={retry} loading={retryMutation.isPending} loadingLabel={t("actions.retrying")}>
              <semanticIcons.action.refresh aria-hidden="true" />{t("actions.retry")}
            </Button>
          ) : null}
          {isRecovery || isAttention ? (
            <Button variant="outline" onClick={() => void operation.refetch()} loading={operation.isFetching} loadingLabel={t("actions.refreshing")}>
              <semanticIcons.action.refresh aria-hidden="true" />{t("actions.recheck")}
            </Button>
          ) : null}
          {terminal ? (
            <Button onClick={leaveUpgrade}>
              <semanticIcons.navigation.overview aria-hidden="true" />{t("actions.enterSystem")}
            </Button>
          ) : null}
        </footer>
      </div>

      <AlertDialog open={stopConfirmOpen} onOpenChange={setStopConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("stop.title")}</AlertDialogTitle>
            <AlertDialogDescription>{t("stop.description")}</AlertDialogDescription>
          </AlertDialogHeader>
          <Alert>
            <semanticIcons.status.warning aria-hidden="true" />
            <AlertDescription>{t("stop.warning")}</AlertDescription>
          </Alert>
          {stopMutation.isError ? (
            <p className="text-sm text-destructive">{getUpgradeErrorMessage(stopMutation.error)}</p>
          ) : null}
          <AlertDialogFooter>
            <AlertDialogClose variant="outline" disabled={stopMutation.isPending}>{t("actions.keepRunning")}</AlertDialogClose>
            <Button variant="destructive" onClick={stop} loading={stopMutation.isPending} loadingLabel={t("actions.stopping")}>
              <semanticIcons.action.stop aria-hidden="true" />{t("actions.confirmStop")}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </main>
  )
}
