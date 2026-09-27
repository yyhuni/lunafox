"use client"

import * as React from "react"
import Link from "next/link"
import { useLocale, useTranslations } from "next-intl"
import { useRouter } from "next/navigation"

import {
  clearStoredUpgradeOperationId,
  getUpgradeErrorMessage,
  isFrontendOnlyUpgrade,
  isUpgradeOperationTerminal,
  upgradeUserStageForStatus,
  upgradeUserStagesForExecutionMode,
  useRetryUpgradeOperation,
  useStopUpgradeOperation,
  useUpgradeOperation,
} from "@/hooks/use-version"
import { type UpgradeAgentDiagnostic, type UpgradeHostActivity, type UpgradeLogEntry, type UpgradeOperationFull, type UpgradeOperationStatus, type UpgradeUserStage } from "@/types/version.types"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { AlertDialog, AlertDialogClose, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Progress } from "@/components/ui/progress"
import { semanticIcons } from "@/components/icons"
import { getLoadingOwnerAttributes } from "@/components/shared/loading/loading-owner"
import { RawLogViewer } from "@/components/shared/visualization/raw-log-viewer"
import { TerminalLogCopyAllButton } from "@/components/shared/visualization/terminal-log-copy-all-button"
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

function formatDuration(operation: UpgradeOperationFull, t: (key: string, params?: Record<string, number | string>) => string): string {
  const start = Date.parse(operation.createdAt)
  const end = Date.parse(operation.completedAt ?? operation.updatedAt)
  if (!Number.isFinite(start) || !Number.isFinite(end) || end < start) return t("summary.durationUnknown")
  const seconds = Math.max(1, Math.round((end - start) / 1000))
  if (seconds < 60) return t("summary.durationSeconds", { seconds })
  return t("summary.durationMinutes", { minutes: Math.max(1, Math.round(seconds / 60)) })
}

function stageTimestamp(operation: UpgradeOperationFull, stage: UpgradeUserStage): string | undefined {
  const values = STAGE_TIME_KEYS[stage]
    .map((key) => operation.stageTimes[key])
    .filter((value): value is string => Boolean(value))
    .sort()
  return values[0]
}

function highestObservedStageIndex(operation: UpgradeOperationFull, stages: readonly UpgradeUserStage[]): number {
  return stages.reduce((highest, stage, index) => {
    if (stage === "finished") return highest
    return stageTimestamp(operation, stage) ? Math.max(highest, index) : highest
  }, -1)
}

function progressValue(status: UpgradeOperationStatus | undefined, stages: readonly UpgradeUserStage[]): number {
  if (!status) return 0
  const stage = upgradeUserStageForStatus(status)
  const index = stages.indexOf(stage)
  if (index < 0) return 0
  if (stage === "finished") return 100
  return Math.max(8, Math.round((index / (stages.length - 1)) * 100))
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
      className="flex h-svh min-h-0 w-full items-center justify-center overflow-y-auto overscroll-contain bg-background px-4 py-8 sm:px-8"
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
  operation: UpgradeOperationFull
  t: (key: string, params?: Record<string, number | string>) => string
  locale: string
}) {
  const stages = upgradeUserStagesForExecutionMode(operation.executionMode)
  const currentStage = upgradeUserStageForStatus(operation.status)
  const currentIndex = stages.indexOf(currentStage)
  const observedIndex = highestObservedStageIndex(operation, stages)
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
          value={progressValue(operation.status, stages)}
          aria-label={t("timeline.phaseProgress")}
          aria-valuetext={t("timeline.stageCount", {
            current: Math.min(stages.indexOf(currentStage) + 1, stages.length),
            total: stages.length,
          })}
        />
      ) : null}
      <ol className="grid gap-2 sm:grid-cols-3 lg:grid-cols-6" aria-label={t("timeline.stageList")}>
        {stages.map((stage, index) => {
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

function HostActivityFact({
  activity,
  locale,
  t,
}: {
  activity: UpgradeHostActivity
  locale: string
  t: (key: string, params?: Record<string, number | string>) => string
}) {
  return (
    <section aria-labelledby="upgrade-host-activity-heading" className="border-t border-border/70 pt-4" data-testid="system-upgrade-host-activity">
      <div className="flex min-w-0 items-start gap-2">
        <semanticIcons.status.running className="mt-0.5 size-4 shrink-0 text-primary" aria-hidden="true" />
        <div className="min-w-0">
          <h2 id="upgrade-host-activity-heading" className={textRole.sectionTitle}>{t("hostActivity.title")}</h2>
          <p className={textRole.bodySubtle}>{t("hostActivity.waiting")}</p>
        </div>
      </div>
      <dl className="mt-3 grid gap-3 sm:grid-cols-3">
        <div className="min-w-0">
          <dt className={textRole.metadataLabel}>{t("hostActivity.action")}</dt>
          <dd className={cn(textRole.metadataValueStrong, "mt-1 break-words")}>{t(`hostActivity.actions.${activity.action}`)}</dd>
        </div>
        <div className="min-w-0">
          <dt className={textRole.metadataLabel}>{t("hostActivity.startedAt")}</dt>
          <dd className={cn(textRole.metadataValue, "mt-1 break-words")}>{formatTimestamp(activity.startedAt, locale)}</dd>
        </div>
        <div className="min-w-0">
          <dt className={textRole.metadataLabel}>{t("hostActivity.lastHeartbeatAt")}</dt>
          <dd className={cn(textRole.metadataValue, "mt-1 break-words")}>{formatTimestamp(activity.lastHeartbeatAt, locale)}</dd>
        </div>
      </dl>
    </section>
  )
}

function OperationFacts({ operation, t }: { operation: UpgradeOperationFull; t: (key: string, params?: Record<string, number | string>) => string }) {
  const frontendOnly = isFrontendOnlyUpgrade(operation)
  const facts: Array<[string, string]> = [
    [t("facts.currentVersion"), operation.currentVersion],
    [t("facts.confirmedDeploymentVersion"), operation.confirmedDeploymentVersion],
    [t("facts.targetVersion"), operation.releaseVersion],
    [t("facts.manifest"), operation.manifestId],
    [t("facts.scope"), t(frontendOnly ? "facts.scopeFrontendOnly" : "facts.scopeFull")],
  ]
  if (!frontendOnly) {
    facts.push([t("facts.migration"), operation.migrationStatus === "not_started" ? t("facts.notStarted") : operation.migrationStatus])
    if (operation.workDisposition === "cancelled") {
      facts.push([t("facts.cancelledWork"), t("facts.cancelledWorkValue", { scans: operation.cancelledScanCount, tasks: operation.cancelledTaskCount })])
    } else {
      facts.push([t("facts.workDispositionLabel"), t(`facts.workDisposition.${operation.workDisposition}`)])
    }
    facts.push([t("facts.agents"), t("facts.agentsValue", { ready: operation.agentSummary.ready, expected: operation.agentSummary.expected, missing: operation.agentSummary.missing, unhealthy: operation.agentSummary.unhealthy })])
  }
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

function AgentDiagnostics({
  diagnostics,
  hasVerification,
  isReconnecting,
  terminal,
  t,
}: {
  diagnostics: UpgradeAgentDiagnostic[]
  hasVerification: boolean
  isReconnecting: boolean
  terminal: boolean
  t: (key: string, params?: Record<string, number | string>) => string
}) {
  if (!hasVerification) return null
  return (
    <Card variant="compact" data-testid="system-upgrade-agent-diagnostics">
      <CardHeader>
        <CardTitle>{t("agentDiagnostics.title")}</CardTitle>
        <CardDescription>{t(diagnostics.length === 0 ? "agentDiagnostics.empty" : "agentDiagnostics.description")}</CardDescription>
      </CardHeader>
      <CardContent>
        {isReconnecting && diagnostics.length > 0 ? <p className={cn(textRole.bodySubtle, "mb-3")}>{t("agentDiagnostics.reconnecting")}</p> : null}
        {terminal && diagnostics.length > 0 ? <p className={cn(textRole.bodySubtle, "mb-3")}>{t("agentDiagnostics.terminal")}</p> : null}
        {diagnostics.length === 0 ? null : (
          <ul className="grid gap-3" aria-label={t("agentDiagnostics.title")}>
            {diagnostics.map((item) => (
              <li key={item.name} className="min-w-0 border-t border-border/70 pt-3 first:border-t-0 first:pt-0">
                <div className="flex min-w-0 flex-wrap items-start justify-between gap-2">
                  <div className="min-w-0">
                    <p className={cn(textRole.bodyStrong, "break-words")}>{item.displayNameSnapshot || item.name}</p>
                    <p className={cn(textRole.code, "mt-1 break-all text-muted-foreground")}>{item.name}</p>
                  </div>
                  <Badge size="compact" variant="warning">{item.reasonCode}</Badge>
                </div>
                <p className={cn(textRole.bodySubtle, "mt-2 break-words")}>{item.detail}</p>
                <p className={cn(textRole.compactCaption, "mt-2")}>{t(`agentDiagnostics.source.${item.source}`)}</p>
              </li>
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  )
}

function upgradeEventTimestamp(value: string): string {
  return new Date(value).toISOString().replace("T", " ").replace(/\.\d{3}Z$/, "")
}

function upgradeEventLevel(level: UpgradeLogEntry["level"]): "INFO" | "WARN" | "ERROR" {
  if (level === "error") return "ERROR"
  if (level === "warn") return "WARN"
  return "INFO"
}

function upgradeEventLine(entry: UpgradeLogEntry): string {
  const metadata = Object.entries(entry.metadata ?? {})
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([key, value]) => `${key}=${value}`)
    .join(" ")
  return `[${upgradeEventTimestamp(entry.timestamp)}] [${upgradeEventLevel(entry.level)}] ${entry.message}${metadata ? ` ${metadata}` : ""}`
}

function UpgradeLogs({
  logs,
  t,
}: {
  logs: UpgradeLogEntry[]
  t: (key: string, params?: Record<string, number | string>) => string
}) {
  const content = React.useMemo(() => [...logs]
    .sort((left, right) => Date.parse(left.timestamp) - Date.parse(right.timestamp) || left.messageKey.localeCompare(right.messageKey))
    .map(upgradeEventLine)
    .join("\n"), [logs])

  return (
    <Card variant="compact">
      <CardHeader>
        <CardTitle>{t("logs.title")}</CardTitle>
        <CardDescription>{t("logs.description")}</CardDescription>
      </CardHeader>
      <CardContent>
        {!content ? (
          <p className={textRole.bodySubtle}>{t("logs.empty")}</p>
        ) : (
          <div className="h-72 overflow-hidden border border-border/70" aria-label={t("logs.title")}>
            <RawLogViewer
              content={content}
              topRightAction={(
                <TerminalLogCopyAllButton
                  value={content}
                  copyLabel={t("logs.copy")}
                  copiedLabel={t("logs.copied")}
                  toastId="system-upgrade-event-copy"
                />
              )}
            />
          </div>
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
      <main className="flex h-svh min-h-0 w-full items-center justify-center overflow-y-auto overscroll-contain bg-background px-4 py-8 sm:px-8" data-testid="system-upgrade-empty">
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
      <main className="flex h-svh min-h-0 w-full items-center justify-center overflow-y-auto overscroll-contain bg-background px-4 py-8 sm:px-8" data-testid="system-upgrade-reconnect">
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
  const frontendOnly = isFrontendOnlyUpgrade(currentOperation)
  const hasAgentVerification = !frontendOnly && currentOperation.agentSummary.expected > 0

  return (
    <main
      {...getLoadingOwnerAttributes({ owner: "system-upgrade-status", layer: "route", intent: "status" })}
      className="h-svh min-h-0 w-full overflow-y-auto overscroll-contain bg-background px-4 py-6 sm:px-8 sm:py-10"
      data-testid="system-upgrade-status"
      tabIndex={0}
    >
      <div className="mx-auto flex w-full max-w-5xl flex-col gap-6">
        <header className="flex flex-wrap items-start justify-between gap-4">
          <div className="min-w-0">
            <p className={textRole.monoLabel}>UPGRADE</p>
            <h1 className={cn(textRole.pageTitleDisplay, "mt-2 text-2xl")}>{t("title")}</h1>
            <p className={cn(textRole.pageDescription, "mt-2 max-w-2xl")}>{t(frontendOnly ? "frontendOnly.description" : terminal ? `statusDescription.${currentOperation.status}` : "description")}</p>
          </div>
          <Badge size="compact" variant={statusVariant(currentOperation.status)} data-testid="system-upgrade-status-badge">
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

        {frontendOnly ? (
          <Alert data-testid="system-upgrade-frontend-only-scope">
            <semanticIcons.status.unknown aria-hidden="true" />
            <AlertTitle>{t("frontendOnly.title")}</AlertTitle>
            <AlertDescription>{t("frontendOnly.activeWork")}</AlertDescription>
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
            {!terminal && currentOperation.hostActivity ? <HostActivityFact activity={currentOperation.hostActivity} locale={locale} t={t} /> : null}
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

        <AgentDiagnostics
          diagnostics={currentOperation.agentDiagnostics}
          hasVerification={hasAgentVerification}
          isReconnecting={operation.isReconnecting}
          terminal={terminal}
          t={t}
        />

        <UpgradeLogs logs={currentOperation.logs} t={t} />

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
          <AlertDialogDescription>{t(frontendOnly ? "stop.frontendOnlyDescription" : "stop.description")}</AlertDialogDescription>
          </AlertDialogHeader>
          <Alert>
            <semanticIcons.status.warning aria-hidden="true" />
            <AlertDescription>{t(frontendOnly ? "stop.frontendOnlyWarning" : "stop.warning")}</AlertDescription>
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
