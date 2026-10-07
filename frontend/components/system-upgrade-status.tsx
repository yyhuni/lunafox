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
  useCreateUpgradeOperation,
  useRetryUpgradeOperation,
  useStopUpgradeOperation,
  useUpgradeOperation,
} from "@/hooks/use-version"
import {
  type UpgradeAgentDiagnostic,
  type UpgradeHostActivity,
  type UpgradeLogEntry,
  type UpgradeOperationFull,
  type UpgradeOperationStatus,
  type UpgradeUserStage,
} from "@/types/version.types"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import {
  AlertDialog,
  AlertDialogClose,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"
import { Progress } from "@/components/ui/progress"
import { ArrowRight, ChevronDown, ChevronUp, semanticIcons } from "@/components/icons"
import { getLoadingOwnerAttributes } from "@/components/shared/loading/loading-owner"
import { RawLogViewer } from "@/components/shared/visualization/raw-log-viewer"
import { TerminalLogCopyAllButton } from "@/components/shared/visualization/terminal-log-copy-all-button"
import { getStatusToneBadgeClass, getStatusToneSurfaceClass, getStatusToneTextClass, type StatusTone } from "@/lib/status-config"
import { compactSurfaceClassNames } from "@/lib/ui/compact-surface-contract"
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

function upgradeStatusTone(status: UpgradeOperationStatus | undefined): StatusTone {
  if (status === "succeeded") return "success"
  if (status === "failed") return "error"
  if (status === "needs_recovery" || status === "needs_attention") return "warning"
  return "info"
}

interface UpgradePreviewScenario {
  id: string
  labelZh: string
  labelEn: string
  operation: UpgradeOperationFull
}

const PREVIEW_TIME_BASE = "2026-10-07T12:00:00Z"

const PREVIEW_SCENARIOS: readonly UpgradePreviewScenario[] = [
  {
    id: "updating_pull_images",
    labelZh: "镜像拉取中（带实时心跳）",
    labelEn: "Pulling Images (Live Heartbeat)",
    operation: {
      name: "upgradeOperations/demo-updating-001",
      operationId: "demo-updating-001",
      requestId: "req-updating-001",
      operatorId: 1,
      manifestId: "release-v0.0.1-alpha.208",
      manifestDigest: "sha256:7b9a4c8e1234567890abcdef1234567890abcdef1234567890abcdef12345678",
      currentVersion: "v0.0.1-alpha.207",
      releaseVersion: "v0.0.1-alpha.208",
      compatibilityRange: ">=0.0.1-alpha.200 <0.0.2",
      maintenanceWindowMinutes: 15,
      status: "updating",
      migrationStatus: "not_started",
      migrationType: "none",
      cancelledScanCount: 2,
      cancelledTaskCount: 5,
      agentSummary: { expected: 3, ready: 0, missing: 0, unhealthy: 0 },
      observedDigests: {},
      logs: [
        {
          timestamp: "2026-10-07T11:58:00Z",
          level: "info",
          stage: "queued",
          messageKey: "requestAccepted",
          message: "Upgrade request accepted and validated",
        },
        {
          timestamp: "2026-10-07T11:58:30Z",
          level: "info",
          stage: "stopping",
          messageKey: "stoppingWork",
          message: "Active background scans safely paused",
        },
        {
          timestamp: "2026-10-07T11:59:00Z",
          level: "info",
          stage: "preflight",
          messageKey: "preflight",
          message: "Preflight verification passed: database profile matched",
        },
        {
          timestamp: "2026-10-07T11:59:30Z",
          level: "info",
          stage: "updating",
          messageKey: "updatingServices",
          message: "Pulling target release images in parallel (ghcr.io/yyhuni/lunafox:208)",
        },
      ],
      stageTimes: {
        queued: "2026-10-07T11:58:00Z",
        stopping: "2026-10-07T11:58:30Z",
        preflight: "2026-10-07T11:59:00Z",
        updating: "2026-10-07T11:59:30Z",
      },
      createdAt: "2026-10-07T11:58:00Z",
      updatedAt: PREVIEW_TIME_BASE,
      completedAt: null,
      executionMode: "full",
      workDisposition: "cancelled",
      planSummary: {
        touchedServices: ["server", "engine", "engine_runtime", "frontend", "nginx"],
      },
      confirmedDeploymentVersion: "v0.0.1-alpha.207",
      agentDiagnostics: [],
      hostActivity: {
        action: "pull_images",
        startedAt: "2026-10-07T11:59:30Z",
        lastHeartbeatAt: PREVIEW_TIME_BASE,
      },
    },
  },
  {
    id: "failed_preheat",
    labelZh: "升级失败（配置不匹配与诊断）",
    labelEn: "Upgrade Failed (Diagnostics)",
    operation: {
      name: "upgradeOperations/demo-failed-002",
      operationId: "demo-failed-002",
      requestId: "req-failed-002",
      operatorId: 1,
      manifestId: "release-v0.0.1-alpha.208",
      manifestDigest: "sha256:8899aabbccddeeff00112233445566778899aabbccddeeff0011223344556677",
      currentVersion: "v0.0.1-alpha.207",
      releaseVersion: "v0.0.1-alpha.208",
      compatibilityRange: ">=0.0.1-alpha.200 <0.0.2",
      maintenanceWindowMinutes: 15,
      status: "failed",
      migrationStatus: "not_started",
      migrationType: "none",
      cancelledScanCount: 1,
      cancelledTaskCount: 2,
      agentSummary: { expected: 2, ready: 0, missing: 0, unhealthy: 0 },
      observedDigests: {},
      diagnostic: "preheat manifest validation failed: COMPOSE_PROFILES must match DATABASE_MODE: COMPOSE_PROFILES=external requires DATABASE_MODE=external (got embedded)",
      logs: [
        {
          timestamp: "2026-10-07T11:55:00Z",
          level: "info",
          stage: "queued",
          messageKey: "requestAccepted",
          message: "Upgrade request accepted",
        },
        {
          timestamp: "2026-10-07T11:55:30Z",
          level: "info",
          stage: "stopping",
          messageKey: "stoppingWork",
          message: "Scan scheduler paused",
        },
        {
          timestamp: "2026-10-07T11:56:00Z",
          level: "error",
          stage: "preflight",
          messageKey: "failed",
          message: "Preheat snapshot check failed: profile mismatch detected",
        },
      ],
      stageTimes: {
        queued: "2026-10-07T11:55:00Z",
        stopping: "2026-10-07T11:55:30Z",
      },
      createdAt: "2026-10-07T11:55:00Z",
      updatedAt: "2026-10-07T11:56:00Z",
      completedAt: "2026-10-07T11:56:00Z",
      executionMode: "full",
      workDisposition: "cancelled",
      planSummary: {
        touchedServices: ["server", "engine", "frontend"],
      },
      confirmedDeploymentVersion: "v0.0.1-alpha.207",
      agentDiagnostics: [],
    },
  },
  {
    id: "needs_attention_agent",
    labelZh: "需要关注（Agent 异常诊断）",
    labelEn: "Needs Attention (Agent Diagnostics)",
    operation: {
      name: "upgradeOperations/demo-attention-003",
      operationId: "demo-attention-003",
      requestId: "req-attention-003",
      operatorId: 1,
      manifestId: "release-v0.0.1-alpha.208",
      manifestDigest: "sha256:11223344556677889900aabbccddeeff11223344556677889900aabbccddeeff",
      currentVersion: "v0.0.1-alpha.207",
      releaseVersion: "v0.0.1-alpha.208",
      compatibilityRange: ">=0.0.1-alpha.200 <0.0.2",
      maintenanceWindowMinutes: 15,
      status: "needs_attention",
      migrationStatus: "succeeded",
      migrationType: "schema_and_data",
      cancelledScanCount: 0,
      cancelledTaskCount: 0,
      agentSummary: { expected: 3, ready: 1, missing: 1, unhealthy: 1 },
      observedDigests: {},
      diagnostic: "Agent cluster verification timed out: 1 agent missing heartbeat, 1 agent version mismatch",
      logs: [
        {
          timestamp: "2026-10-07T11:50:00Z",
          level: "info",
          stage: "queued",
          messageKey: "requestAccepted",
          message: "Upgrade request accepted",
        },
        {
          timestamp: "2026-10-07T11:52:00Z",
          level: "info",
          stage: "updating",
          messageKey: "updatingServices",
          message: "Core service images pulled and updated",
        },
        {
          timestamp: "2026-10-07T11:54:00Z",
          level: "info",
          stage: "migrating",
          messageKey: "migratingDatabase",
          message: "Database schema migration executed successfully",
        },
        {
          timestamp: "2026-10-07T11:56:00Z",
          level: "info",
          stage: "restarting",
          messageKey: "restartingServices",
          message: "Services restarted, verifying agent connectivity",
        },
        {
          timestamp: "2026-10-07T11:58:00Z",
          level: "warn",
          stage: "agent_verifying",
          messageKey: "needsAttention",
          message: "Edge agent nodes took longer than expected to reconnect",
        },
      ],
      stageTimes: {
        queued: "2026-10-07T11:50:00Z",
        stopping: "2026-10-07T11:51:00Z",
        updating: "2026-10-07T11:52:00Z",
        restarting: "2026-10-07T11:56:00Z",
      },
      createdAt: "2026-10-07T11:50:00Z",
      updatedAt: "2026-10-07T11:58:00Z",
      completedAt: "2026-10-07T11:58:00Z",
      executionMode: "full",
      workDisposition: "cancelled",
      planSummary: {
        touchedServices: ["server", "migration", "engine", "agent", "frontend"],
      },
      confirmedDeploymentVersion: "v0.0.1-alpha.207",
      agentDiagnostics: [
        {
          agentId: 101,
          name: "agents/101",
          displayNameSnapshot: "edge-node-shanghai-01",
          reasonCode: "heartbeat_missing_or_stale",
          detail: "Heartbeat has not been received for 120 seconds, node might be rebooting or unreachable",
          source: "server_observation",
        },
        {
          agentId: 102,
          name: "agents/102",
          displayNameSnapshot: "crawler-worker-beijing-02",
          reasonCode: "version_mismatch",
          detail: "Agent binary is v0.0.1-alpha.205, expected target contract v0.0.1-alpha.208",
          source: "server_observation",
        },
      ],
    },
  },
  {
    id: "succeeded",
    labelZh: "升级成功（完成与可进入系统）",
    labelEn: "Upgrade Succeeded",
    operation: {
      name: "upgradeOperations/demo-succeeded-004",
      operationId: "demo-succeeded-004",
      requestId: "req-succeeded-004",
      operatorId: 1,
      manifestId: "release-v0.0.1-alpha.208",
      manifestDigest: "sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
      currentVersion: "v0.0.1-alpha.207",
      releaseVersion: "v0.0.1-alpha.208",
      compatibilityRange: ">=0.0.1-alpha.200 <0.0.2",
      maintenanceWindowMinutes: 15,
      status: "succeeded",
      migrationStatus: "succeeded",
      migrationType: "schema_and_data",
      cancelledScanCount: 3,
      cancelledTaskCount: 7,
      agentSummary: { expected: 2, ready: 2, missing: 0, unhealthy: 0 },
      observedDigests: {},
      logs: [
        {
          timestamp: "2026-10-07T11:40:00Z",
          level: "info",
          stage: "queued",
          messageKey: "requestAccepted",
          message: "Upgrade request accepted",
        },
        {
          timestamp: "2026-10-07T11:40:30Z",
          level: "info",
          stage: "stopping",
          messageKey: "stoppingWork",
          message: "Background tasks safely cancelled",
        },
        {
          timestamp: "2026-10-07T11:41:00Z",
          level: "info",
          stage: "updating",
          messageKey: "updatingServices",
          message: "Service containers updated",
        },
        {
          timestamp: "2026-10-07T11:42:00Z",
          level: "info",
          stage: "migrating",
          messageKey: "migratingDatabase",
          message: "Database migration applied cleanly",
        },
        {
          timestamp: "2026-10-07T11:42:30Z",
          level: "info",
          stage: "restarting",
          messageKey: "restartingServices",
          message: "System restarted with new version",
        },
        {
          timestamp: "2026-10-07T11:43:00Z",
          level: "info",
          stage: "verifying",
          messageKey: "completed",
          message: "All checks passed. Target release v0.0.1-alpha.208 is active",
        },
      ],
      stageTimes: {
        queued: "2026-10-07T11:40:00Z",
        stopping: "2026-10-07T11:40:30Z",
        updating: "2026-10-07T11:41:00Z",
        restarting: "2026-10-07T11:42:30Z",
      },
      createdAt: "2026-10-07T11:40:00Z",
      updatedAt: "2026-10-07T11:43:00Z",
      completedAt: "2026-10-07T11:43:00Z",
      executionMode: "full",
      workDisposition: "cancelled",
      planSummary: {
        touchedServices: ["server", "migration", "engine", "frontend", "nginx"],
      },
      confirmedDeploymentVersion: "v0.0.1-alpha.208",
      agentDiagnostics: [],
    },
  },
  {
    id: "frontend_only",
    labelZh: "仅前端更新（轻量热更新模式）",
    labelEn: "Frontend Only (Hot Replacement)",
    operation: {
      name: "upgradeOperations/demo-fe-only-005",
      operationId: "demo-fe-only-005",
      requestId: "req-fe-only-005",
      operatorId: 1,
      manifestId: "release-v0.0.1-alpha.208-fe",
      manifestDigest: "sha256:fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
      currentVersion: "v0.0.1-alpha.207",
      releaseVersion: "v0.0.1-alpha.208",
      compatibilityRange: ">=0.0.1-alpha.200 <0.0.2",
      maintenanceWindowMinutes: 5,
      status: "updating",
      migrationStatus: "not_started",
      migrationType: "none",
      cancelledScanCount: 0,
      cancelledTaskCount: 0,
      agentSummary: { expected: 0, ready: 0, missing: 0, unhealthy: 0 },
      observedDigests: {},
      logs: [
        {
          timestamp: "2026-10-07T11:59:00Z",
          level: "info",
          stage: "queued",
          messageKey: "requestAccepted",
          message: "Frontend hot replacement request accepted",
        },
        {
          timestamp: "2026-10-07T11:59:40Z",
          level: "info",
          stage: "updating",
          messageKey: "updatingServices",
          message: "Deploying updated Web frontend container and assets",
        },
      ],
      stageTimes: {
        queued: "2026-10-07T11:59:00Z",
        preflight: "2026-10-07T11:59:20Z",
        updating: "2026-10-07T11:59:40Z",
      },
      createdAt: "2026-10-07T11:59:00Z",
      updatedAt: PREVIEW_TIME_BASE,
      completedAt: null,
      executionMode: "frontend_only",
      workDisposition: "not_required",
      planSummary: {
        touchedServices: ["frontend"],
      },
      confirmedDeploymentVersion: "v0.0.1-alpha.207",
      agentDiagnostics: [],
      hostActivity: {
        action: "verify_frontend_container",
        startedAt: "2026-10-07T11:59:40Z",
        lastHeartbeatAt: PREVIEW_TIME_BASE,
      },
    },
  },
]

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
            <semanticIcons.status.running className={cn("size-5", getStatusToneTextClass("info"))} aria-hidden="true" />
            {title}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <p className={textRole.bodySubtle}>{description}</p>
        </CardContent>
      </Card>
    </main>
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
    <Card variant="compact" data-testid="system-upgrade-host-activity">
      <CardHeader>
        <CardTitle id="upgrade-host-activity-heading">{t("hostActivity.title")}</CardTitle>
        <CardDescription>{t("hostActivity.waiting")}</CardDescription>
      </CardHeader>
      <CardContent>
        <dl className="grid gap-3 sm:grid-cols-3">
          <div className="min-w-0">
            <dt className={textRole.metadataLabel}>{t("hostActivity.action")}</dt>
            <dd className={cn(textRole.metadataValueStrong, "mt-1 break-words")}>
              {t(`hostActivity.actions.${activity.action}`)}
            </dd>
          </div>
          <div className="min-w-0">
            <dt className={textRole.metadataLabel}>{t("hostActivity.startedAt")}</dt>
            <dd className={cn(textRole.metadataValue, "mt-1 break-words")}>
              {formatTimestamp(activity.startedAt, locale)}
            </dd>
          </div>
          <div className="min-w-0">
            <dt className={textRole.metadataLabel}>{t("hostActivity.lastHeartbeatAt")}</dt>
            <dd className={cn(textRole.metadataValue, "mt-1 break-words")}>
              {formatTimestamp(activity.lastHeartbeatAt, locale)}
            </dd>
          </div>
        </dl>
      </CardContent>
    </Card>
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
    <dl className="grid gap-2.5 divide-y divide-border/60">
      {facts.map(([label, value], idx) => (
        <div key={label} className={cn("flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between", idx > 0 && "pt-2.5")}>
          <dt className={textRole.metadataLabel}>{label}</dt>
          <dd className={cn(textRole.metadataValueStrong, "break-all sm:max-w-xs sm:text-right")}>
            {value}
          </dd>
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
        <CardTitle className="flex items-center gap-2">
          <semanticIcons.concept.agent className={cn("size-4", getStatusToneTextClass("warning"))} aria-hidden="true" />
          {t("agentDiagnostics.title")}
        </CardTitle>
        <CardDescription>{t(diagnostics.length === 0 ? "agentDiagnostics.empty" : "agentDiagnostics.description")}</CardDescription>
      </CardHeader>
      <CardContent>
        {isReconnecting && diagnostics.length > 0 ? <p className={cn(textRole.bodySubtle, "mb-3")}>{t("agentDiagnostics.reconnecting")}</p> : null}
        {terminal && diagnostics.length > 0 ? <p className={cn(textRole.bodySubtle, "mb-3")}>{t("agentDiagnostics.terminal")}</p> : null}
        {diagnostics.length === 0 ? null : (
          <ul className="grid gap-3" aria-label={t("agentDiagnostics.title")}>
            {diagnostics.map((item) => (
              <li key={item.name} className={cn("min-w-0 radius-control p-3", getStatusToneSurfaceClass("warning"))}>
                <div className="flex min-w-0 flex-wrap items-start justify-between gap-2">
                  <div className="min-w-0">
                    <p className={cn(textRole.bodyStrong, "break-words")}>{item.displayNameSnapshot || item.name}</p>
                    <p className={cn(textRole.code, "mt-1 break-all")}>{item.name}</p>
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
        <CardTitle className="flex items-center gap-2">
          <semanticIcons.concept.systemLog className="size-4" aria-hidden="true" />
          {t("logs.title")}
        </CardTitle>
        <CardDescription>{t("logs.description")}</CardDescription>
      </CardHeader>
      <CardContent>
        {!content ? (
          <p className={textRole.bodySubtle}>{t("logs.empty")}</p>
        ) : (
          <div className="h-80 overflow-hidden" aria-label={t("logs.title")}>
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

interface SchemeViewProps {
  operation: UpgradeOperationFull
  stages: readonly UpgradeUserStage[]
  currentStage: UpgradeUserStage
  currentIndex: number
  observedIndex: number
  terminal: boolean
  isSuccess: boolean
  isFailure: boolean
  isAttention: boolean
  isRecovery: boolean
  retryable: boolean
  frontendOnly: boolean
  hasAgentVerification?: boolean
  isReconnecting?: boolean
  locale: string
  t: (key: string, params?: Record<string, number | string>) => string
  onLeaveUpgrade?: () => void
  onRetry?: () => void
  onStop?: () => void
  onRefresh?: () => void
  isFetching?: boolean
  isRetrying?: boolean
  isStopping?: boolean
  onCopyDiagnostics?: () => void
  copiedDiagnostics?: boolean
}

function SchemeMinimal({
  operation,
  stages,
  currentStage,
  currentIndex,
  observedIndex,
  terminal,
  isSuccess,
  isFailure,
  isAttention,
  isRecovery,
  retryable,
  frontendOnly,
  hasAgentVerification = !frontendOnly && operation.agentSummary.expected > 0,
  isReconnecting = false,
  locale,
  t,
  onLeaveUpgrade,
  onRetry,
  onStop,
  onRefresh,
  isFetching,
  isRetrying,
  isStopping,
  onCopyDiagnostics,
  copiedDiagnostics,
}: SchemeViewProps) {
  const [detailsOpen, setDetailsOpen] = React.useState(false)
  const progress = progressValue(operation.status, stages)
  const tone = upgradeStatusTone(operation.status)
  const StatusIcon = tone === "success"
    ? semanticIcons.status.success
    : tone === "error"
      ? semanticIcons.status.failed
      : tone === "warning"
        ? semanticIcons.status.warning
        : semanticIcons.status.running
  const leaveIsSecondary = retryable || isRecovery || isAttention

  return (
    <div className="mx-auto flex w-full max-w-2xl flex-col items-center gap-6 py-6 text-center">
      <div
        className={cn(
          "flex size-20 items-center justify-center radius-round",
          getStatusToneSurfaceClass(tone),
          getStatusToneTextClass(tone),
        )}
        aria-hidden="true"
      >
        <StatusIcon className="size-10" />
      </div>

      <div className="space-y-3" aria-live="polite">
        <div className="inline-flex items-center gap-2 radius-pill border border-border bg-muted/30 px-3 py-1">
          <span className={textRole.code}>{operation.currentVersion}</span>
          <ArrowRight className="size-3" aria-hidden="true" />
          <span className={textRole.code}>{operation.releaseVersion}</span>
        </div>
        <h1 className={textRole.pageTitleDisplay}>
          {isSuccess
            ? t("outcomes.succeeded.title")
            : isFailure
              ? t("outcomes.failed.title")
              : isRecovery || isAttention
                ? t(`outcomes.${operation.status}.title`)
                : t("title")}
        </h1>
        <p className={cn(textRole.pageDescription, "mx-auto max-w-md")}>
          {isSuccess
            ? t("outcomes.succeeded.description", { version: operation.releaseVersion, duration: formatDuration(operation, t) })
            : isFailure
              ? t("outcomes.failed.description")
              : isRecovery || isAttention
                ? t(`outcomes.${operation.status}.description`)
                : t(`timeline.statusDescription.${operation.status}`)}
        </p>
      </div>

      <Card variant="compact" className="w-full text-left">
        <CardHeader>
          <div className="flex items-center justify-between gap-2">
            <CardTitle>{t("timeline.title")}</CardTitle>
            <Badge size="compact" variant={statusVariant(operation.status)} data-testid="system-upgrade-status-badge">
              {t(`status.${operation.status}`)}
            </Badge>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <Progress
            value={progress}
            aria-label={t("timeline.phaseProgress")}
            aria-valuetext={t("timeline.stageCount", { current: currentIndex + 1, total: stages.length })}
          />
          <ol className="grid grid-cols-2 gap-3 sm:grid-cols-3" aria-label={t("timeline.stageList")}>
            {stages.map((stage, idx) => {
              const isCurrent = terminal ? stage === "finished" : idx === currentIndex
              const isComplete = operation.status === "succeeded" ? idx < currentIndex : terminal ? stage !== "finished" && idx <= observedIndex : idx < currentIndex
              const stageTone: StatusTone = isComplete ? "success" : isCurrent ? "info" : "muted"
              const StageIcon = isComplete
                ? semanticIcons.status.success
                : isCurrent
                  ? semanticIcons.status.running
                  : semanticIcons.status.unknown
              return (
                <li
                  key={stage}
                  data-stage={stage}
                  data-stage-state={isCurrent ? "current" : isComplete ? "complete" : "pending"}
                  className="flex min-w-0 flex-col items-center gap-1.5 text-center"
                >
                  <div className={cn("flex size-5 items-center justify-center radius-round", getStatusToneBadgeClass(stageTone))}>
                    <StageIcon className="size-3" aria-hidden="true" />
                  </div>
                  <span className={cn(isCurrent ? textRole.compactPrimary : textRole.compactCaption, "break-words")}>
                    {t(`timeline.stages.${stage}`)}
                  </span>
                  <span className="sr-only">
                    {t(isCurrent ? "timeline.current" : isComplete ? "timeline.complete" : "timeline.pending")}
                  </span>
                </li>
              )
            })}
          </ol>
          <p className="text-center">
            <span className={textRole.compactCaption}>
              {t("timeline.stageCount", { current: Math.min(currentIndex + 1, stages.length), total: stages.length })}
              {": "}
            </span>
            <span className={textRole.compactPrimary}>{t(`timeline.stages.${currentStage}`)}</span>
          </p>
        </CardContent>
      </Card>

      <div className="flex w-full flex-wrap items-center justify-center gap-3">
        {retryable && onRetry ? (
          <Button onClick={onRetry} loading={isRetrying} loadingLabel={t("actions.retrying")}>
            <semanticIcons.action.refresh aria-hidden="true" />
            {t("actions.retry")}
          </Button>
        ) : null}
        {(isRecovery || isAttention) && onRefresh ? (
          <Button
            variant={retryable ? "outline" : "default"}
            onClick={onRefresh}
            loading={isFetching}
            loadingLabel={t("actions.refreshing")}
          >
            <semanticIcons.action.refresh aria-hidden="true" />
            {t("actions.recheck")}
          </Button>
        ) : null}
        {terminal && onLeaveUpgrade ? (
          <Button variant={leaveIsSecondary ? "outline" : "default"} onClick={onLeaveUpgrade}>
            <semanticIcons.navigation.overview aria-hidden="true" />
            {t("actions.enterSystem")}
          </Button>
        ) : null}
        {!terminal && onStop ? (
          <Button variant="outline" onClick={onStop} loading={isStopping} loadingLabel={t("actions.stopping")}>
            <semanticIcons.action.stop aria-hidden="true" />
            {t("actions.stop")}
          </Button>
        ) : null}
        {!isSuccess && !isRecovery && !isAttention && !isFailure && onRefresh ? (
          <Button variant="outline" onClick={onRefresh} loading={isFetching} loadingLabel={t("actions.refreshing")}>
            <semanticIcons.action.refresh aria-hidden="true" />
            {t("actions.refresh")}
          </Button>
        ) : null}
      </div>

      {operation.diagnostic ? (
        <div className="flex w-full flex-col items-stretch gap-2 text-left">
          <Alert variant={isFailure ? "destructive" : "default"}>
            <AlertTitle>{t("diagnostics.title")}</AlertTitle>
            <AlertDescription>
              <p className={cn(textRole.code, "break-all")}>{operation.diagnostic}</p>
            </AlertDescription>
          </Alert>
          {onCopyDiagnostics ? (
            <Button variant="outline" size="sm" className="self-start" onClick={onCopyDiagnostics}>
              <semanticIcons.action.copy aria-hidden="true" />
              {copiedDiagnostics ? t("actions.diagnosticsCopied") : t("actions.copyDiagnostics")}
            </Button>
          ) : null}
        </div>
      ) : null}

      {/* 6. Host Activity Telemetry */}
      {!terminal && operation.hostActivity ? (
        <div className="w-full text-left">
          <HostActivityFact activity={operation.hostActivity} locale={locale} t={t} />
        </div>
      ) : null}

      {/* 7. Agent Diagnostics */}
      {hasAgentVerification && (
        <div className="w-full text-left">
          <AgentDiagnostics
            diagnostics={operation.agentDiagnostics}
            hasVerification={hasAgentVerification}
            isReconnecting={isReconnecting}
            terminal={terminal}
            t={t}
          />
        </div>
      )}

      {/* 8. Live Events Stream */}
      <div className="w-full text-left">
        <UpgradeLogs logs={operation.logs} t={t} />
      </div>

      {/* 9. Deployment Metadata Drawer */}
      <div className="w-full text-left">
        <Collapsible open={detailsOpen} onOpenChange={setDetailsOpen}>
          <CollapsibleTrigger
            render={
              <Button variant="outline" size="sm" layout="between" />
            }
          >
            <span className={textRole.compactPrimary}>{t("facts.drawerTitle")}</span>
            <span className="inline-flex items-center gap-1">
              {detailsOpen ? <ChevronUp className="size-3.5" aria-hidden="true" /> : <ChevronDown className="size-3.5" aria-hidden="true" />}
              <span className={textRole.metadataLabel}>{detailsOpen ? t("actions.hideFacts") : t("actions.showFacts")}</span>
            </span>
          </CollapsibleTrigger>
          <CollapsibleContent className="mt-4">
            <OperationFacts operation={operation} t={t} />
          </CollapsibleContent>
        </Collapsible>
      </div>

      {/* 7. Subtle Footer */}
      <div className="space-y-1 pt-4">
        <p className={textRole.caption}>
          {t("facts.lastUpdated")}: {formatTimestamp(operation.updatedAt, locale)}
        </p>
        <p className={textRole.caption}>
          {t("facts.operationId")}: <code className={textRole.code}>{operation.operationId}</code>
        </p>
      </div>
    </div>
  )
}

export function SystemUpgradeStatus() {
  const t = useTranslations("systemUpgrade")
  const locale = useLocale()
  const router = useRouter()
  const [hydrated, setHydrated] = React.useState(false)
  const [stopConfirmOpen, setStopConfirmOpen] = React.useState(false)
  const [copiedDiagnostics, setCopiedDiagnostics] = React.useState(false)
  const [previewScenarioId, setPreviewScenarioId] = React.useState<string | null>(null)
  const [requestedOpId, setRequestedOpId] = React.useState<string | undefined>(undefined)

  const operation = useUpgradeOperation(requestedOpId, { enabled: hydrated })
  const createMutation = useCreateUpgradeOperation()
  const retryMutation = useRetryUpgradeOperation()
  const stopMutation = useStopUpgradeOperation()

  React.useEffect(() => {
    setHydrated(true)
    if (typeof window !== "undefined") {
      const params = new URLSearchParams(window.location.search)
      const opId = params.get("operationId")
      if (opId) setRequestedOpId(opId)
      const requestedScenario = params.get("scenario")
      if (requestedScenario && PREVIEW_SCENARIOS.some((s) => s.id === requestedScenario)) {
        setPreviewScenarioId(requestedScenario)
      }
    }
  }, [])

  const startLiveMockUpgrade = React.useCallback(() => {
    setPreviewScenarioId(null)
    if (typeof window !== "undefined") {
      try {
        window.localStorage.removeItem("lunafox.mock.upgrade.state.v2")
        window.localStorage.removeItem("lunafox.upgrade.operation.id")
        window.localStorage.removeItem("lunafox.upgrade.operationId")
      } catch {
        // storage disabled
      }
    }
    createMutation.mutate({
      requestId: crypto.randomUUID(),
      manifestId: "lunafox-0.0.1-alpha.208",
      manifestDigest: "sha256:" + "0123456789abcdef".repeat(4),
      confirmed: true,
    }, {
      onSuccess: (newOp) => {
        setRequestedOpId(newOp.operationId)
      }
    })
  }, [createMutation])

  const activePreviewScenario = React.useMemo(() => {
    if (!previewScenarioId) return null
    return PREVIEW_SCENARIOS.find((item) => item.id === previewScenarioId) ?? null
  }, [previewScenarioId])

  const leaveUpgrade = React.useCallback(() => {
    if (activePreviewScenario) {
      setPreviewScenarioId(null)
      return
    }
    if (operation.operationId) clearStoredUpgradeOperationId(operation.operationId)
    router.replace("/overview/")
  }, [activePreviewScenario, operation.operationId, router])

  const retry = React.useCallback(() => {
    if (activePreviewScenario) {
      const queuedScenario = PREVIEW_SCENARIOS.find((s) => s.id === "updating_pull_images")
      if (queuedScenario) setPreviewScenarioId(queuedScenario.id)
      return
    }
    if (!operation.operationId || retryMutation.isPending) return
    retryMutation.mutate(operation.operationId)
  }, [activePreviewScenario, operation.operationId, retryMutation])

  const stop = React.useCallback(() => {
    if (activePreviewScenario) {
      setStopConfirmOpen(false)
      const failedScenario = PREVIEW_SCENARIOS.find((s) => s.id === "failed_preheat")
      if (failedScenario) setPreviewScenarioId(failedScenario.id)
      return
    }
    if (!operation.operationId || stopMutation.isPending) return
    stopMutation.mutate(operation.operationId, {
      onSuccess: () => setStopConfirmOpen(false),
    })
  }, [activePreviewScenario, operation.operationId, stopMutation])

  const handleCopyDiagnostics = React.useCallback((targetOp: UpgradeOperationFull) => {
    const lines = [
      `Operation ID: ${targetOp.operationId}`,
      `Status: ${targetOp.status}`,
      `Current Version: ${targetOp.currentVersion}`,
      `Target Version: ${targetOp.releaseVersion}`,
      `Diagnostic: ${targetOp.diagnostic || "None"}`,
      `Updated At: ${targetOp.updatedAt}`,
    ]
    void navigator.clipboard.writeText(lines.join("\n")).then(() => {
      setCopiedDiagnostics(true)
      setTimeout(() => setCopiedDiagnostics(false), 2000)
    }).catch(() => {
      // fallback
    })
  }, [])

  if (!hydrated || (!activePreviewScenario && operation.isResolving)) {
    return <UpgradeLoadingOwner title={t("loading.title")} description={t("loading.description")} />
  }

  // When no operation exists and no preview scenario is active
  if (!activePreviewScenario && !operation.operationId && !operation.isError) {
    return (
      <main
        className="flex h-svh min-h-0 w-full flex-col items-center justify-center overflow-y-auto overscroll-contain bg-background px-4 py-8 sm:px-8"
        data-testid="system-upgrade-empty"
      >
        <Card className="w-full max-w-lg" variant="compact">
          <CardHeader>
            <CardTitle>{t("empty.title")}</CardTitle>
            <CardDescription>{t("empty.description")}</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className={compactSurfaceClassNames.mutedInfo}>
              <p className={cn(textRole.bodyStrong, "mb-2")}>{t("actions.demoMode")}:</p>
              <div className="flex flex-wrap gap-1.5">
                {PREVIEW_SCENARIOS.map((scenario) => (
                  <Button
                    key={scenario.id}
                    variant="outline"
                    size="sm"
                    onClick={() => setPreviewScenarioId(scenario.id)}
                  >
                    {locale === "zh" ? scenario.labelZh : scenario.labelEn}
                  </Button>
                ))}
              </div>
            </div>
          </CardContent>
          <CardFooter className="flex flex-wrap items-center gap-2">
            {process.env.NEXT_PUBLIC_USE_MOCK === "true" ? (
              <Button
                onClick={startLiveMockUpgrade}
                loading={createMutation.isPending}
                loadingLabel={t("actions.startingLiveMock")}
              >
                <semanticIcons.status.running className="size-4" aria-hidden="true" />
                {t("actions.startLiveMock")}
              </Button>
            ) : null}
            <Button variant={process.env.NEXT_PUBLIC_USE_MOCK === "true" ? "outline" : "default"} render={<Link href="/overview/" />}>
              <semanticIcons.navigation.overview aria-hidden="true" />
              {t("actions.enterSystem")}
            </Button>
          </CardFooter>
        </Card>
      </main>
    )
  }

  if (!activePreviewScenario && !operation.data) {
    const reconnecting = operation.isReconnecting || operation.isError
    return (
      <main
        className="flex h-svh min-h-0 w-full items-center justify-center overflow-y-auto overscroll-contain bg-background px-4 py-8 sm:px-8"
        data-testid="system-upgrade-reconnect"
      >
        <Card className="w-full max-w-lg" variant="compact">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <semanticIcons.status.unknown className={cn("size-5", getStatusToneTextClass("info"))} aria-hidden="true" />
              {t(reconnecting ? "reconnecting.title" : "loading.title")}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className={textRole.bodySubtle}>{reconnecting ? t("reconnecting.description") : t("loading.description")}</p>
            {operation.error ? <p className={cn("mt-3", textRole.body, getStatusToneTextClass("error"))}>{getUpgradeErrorMessage(operation.error)}</p> : null}
          </CardContent>
          <CardFooter className="gap-2">
            <Button
              onClick={() => void operation.refetch()}
              loading={operation.isFetching}
              loadingLabel={t("actions.refreshing")}
            >
              <semanticIcons.action.refresh aria-hidden="true" />
              {t("actions.refresh")}
            </Button>
          </CardFooter>
        </Card>
      </main>
    )
  }

  const currentOperation = activePreviewScenario ? activePreviewScenario.operation : operation.data!
  const terminal = isUpgradeOperationTerminal(currentOperation.status)
  const isSuccess = currentOperation.status === "succeeded"
  const isFailure = currentOperation.status === "failed"
  const isRecovery = currentOperation.status === "needs_recovery"
  const isAttention = currentOperation.status === "needs_attention"
  const retryable = isFailure || isAttention
  const frontendOnly = isFrontendOnlyUpgrade(currentOperation)
  const hasAgentVerification = !frontendOnly && currentOperation.agentSummary.expected > 0
  const stages = upgradeUserStagesForExecutionMode(currentOperation.executionMode)
  const currentStage = upgradeUserStageForStatus(currentOperation.status)
  const currentIndex = stages.indexOf(currentStage)
  const observedIndex = highestObservedStageIndex(currentOperation, stages)

  // Retain contract references for host activity and agent diagnostics bindings:
  void currentOperation.hostActivity
  void currentOperation.agentDiagnostics

  const schemeProps: SchemeViewProps = {
    operation: currentOperation,
    stages,
    currentStage,
    currentIndex,
    observedIndex,
    terminal,
    isSuccess,
    isFailure,
    isAttention,
    isRecovery,
    retryable,
    frontendOnly,
    hasAgentVerification,
    isReconnecting: operation.isReconnecting,
    locale,
    t,
    onLeaveUpgrade: terminal ? leaveUpgrade : undefined,
    onRetry: retryable ? retry : undefined,
    onStop: !terminal ? () => setStopConfirmOpen(true) : undefined,
    onRefresh: () => void operation.refetch(),
    isFetching: operation.isFetching,
    isRetrying: retryMutation.isPending,
    isStopping: stopMutation.isPending,
    onCopyDiagnostics: currentOperation.diagnostic ? () => handleCopyDiagnostics(currentOperation) : undefined,
    copiedDiagnostics,
  }

  return (
    <main
      {...getLoadingOwnerAttributes({ owner: "system-upgrade-status", layer: "route", intent: "status" })}
      className="h-svh min-h-0 w-full overflow-y-auto overscroll-contain bg-background px-4 py-6 sm:px-8 sm:py-10"
      data-testid="system-upgrade-status"
      tabIndex={0}
    >
      <div className="mx-auto flex w-full max-w-4xl flex-col gap-6">
        {/* Mock Control Banner (Rendered only when mock mode is enabled) */}
        {process.env.NEXT_PUBLIC_USE_MOCK === "true" ? (
          <div className="flex flex-wrap items-center justify-between gap-2 radius-surface border border-border bg-muted/30 px-3 py-2">
            <div className="flex items-center gap-2">
              <span className={textRole.monoLabel}>{t("mock.mode")}</span>
              <span className={textRole.compactPrimary}>
                {activePreviewScenario
                  ? (locale === "zh" ? activePreviewScenario.labelZh : activePreviewScenario.labelEn)
                  : t("mock.live")}
              </span>
            </div>
            <div className="flex flex-wrap items-center gap-1.5">
              <Button
                variant="outline"
                size="sm"
                onClick={startLiveMockUpgrade}
                loading={createMutation.isPending}
                loadingLabel={t("actions.startingLiveMock")}
              >
                <semanticIcons.status.running className="size-3.5" aria-hidden="true" />
                {t("actions.startLiveMock")}
              </Button>
              {PREVIEW_SCENARIOS.map((item) => (
                <Button
                  key={item.id}
                  variant={item.id === previewScenarioId ? "secondary" : "ghost"}
                  size="sm"
                  onClick={() => setPreviewScenarioId(item.id)}
                >
                  {locale === "zh" ? item.labelZh.split("（")[0] : item.labelEn.split(" (")[0]}
                </Button>
              ))}
              {activePreviewScenario ? (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setPreviewScenarioId(null)}
                >
                  {t("actions.exitDemo")}
                </Button>
              ) : null}
            </div>
          </div>
        ) : null}

        {/* Global Notifications / Alerts */}
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

        {/* Scheme 1: Minimalist Center View */}
        <SchemeMinimal {...schemeProps} />
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
            <p className={cn(textRole.body, getStatusToneTextClass("error"))}>{getUpgradeErrorMessage(stopMutation.error)}</p>
          ) : null}
          <AlertDialogFooter>
            <AlertDialogClose variant="outline" disabled={stopMutation.isPending}>{t("actions.keepRunning")}</AlertDialogClose>
            <Button
              variant="destructive"
              onClick={stop}
              loading={stopMutation.isPending}
              loadingLabel={t("actions.stopping")}
            >
              <semanticIcons.action.stop aria-hidden="true" />{t("actions.confirmStop")}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </main>
  )
}
