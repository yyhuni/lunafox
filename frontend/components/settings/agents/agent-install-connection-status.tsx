"use client"

import type { ReactNode } from "react"
import { AnimatePresence, motion, useReducedMotion } from "framer-motion"
import { useTranslations } from "next-intl"
import { Loader2, semanticIcons } from "@/components/icons"
import { formatPageRefreshTimestamp } from "@/components/common/page-refresh-status-button"
import type { AgentInstallConnectionViewState } from "@/hooks/use-agent-install-connection"
import type { AgentInstallConnectionPhase } from "@/lib/agent-install-connection"
import { getStatusToneTextClass, type StatusTone } from "@/lib/status-config"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

const AgentIcon = semanticIcons.concept.agent
const OnlineIcon = semanticIcons.status.success
const WarningIcon = semanticIcons.status.warning

const phasePresentation: Record<AgentInstallConnectionPhase, {
  borderClassName: string
  tone: StatusTone
}> = {
  waiting: { borderClassName: "border-info", tone: "info" },
  registered: { borderClassName: "border-warning", tone: "warning" },
  online: { borderClassName: "border-success", tone: "success" },
}

interface AgentInstallConnectionStatusProps {
  connection: AgentInstallConnectionViewState
}

export function AgentInstallConnectionStatus({
  connection,
}: AgentInstallConnectionStatusProps) {
  const t = useTranslations("settings.agents.install.connection")
  const prefersReducedMotion = useReducedMotion()
  const {
    phase,
    agents,
    endReason,
    hasSuccessfulProjection,
    hasDetectionError,
    isInitialError,
    isLastKnown,
    isTokenExpired,
    lastSuccessfulAt,
  } = connection
  const presentation = endReason === "deadline"
    ? { borderClassName: "border-warning", tone: "warning" as const }
    : phasePresentation[phase]
  const onlineCount = agents.filter((agent) => agent.status === "online").length
  const waitingCount = agents.length - onlineCount
  const title = endReason === "allOnline"
    ? t("completedTitle")
    : endReason === "deadline"
      ? t("endedTitle")
      : phase === "waiting"
        ? t("waitingTitle")
        : phase === "registered"
          ? t("registeredTitle")
          : t("onlineTitle")
  const lastSuccessTime = lastSuccessfulAt
    ? formatPageRefreshTimestamp(new Date(lastSuccessfulAt))
    : null

  const content = (
    <div
      className={cn(
        "border-l-2 py-1.5 pl-3 transition-colors duration-[var(--motion-duration-fast)] ease-[var(--motion-ease-standard)] motion-reduce:transition-none",
        presentation.borderClassName,
      )}
    >
      <div className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
        <ConnectionMark
          phase={phase}
          endReason={endReason}
          className={cn("size-4 shrink-0", getStatusToneTextClass(presentation.tone))}
        />
        <span className={textRole.bodyStrong}>{title}</span>
        {agents.length > 0 && (
          <span className={textRole.caption}>
            {t("summary", {
              total: agents.length,
              online: onlineCount,
              waiting: waitingCount,
            })}
          </span>
        )}
      </div>

      {agents.length === 0 ? (
        <p className={cn("ml-6 mt-1", textRole.caption)}>
          {endReason ? t("noAgentsFinal") : t("waitingDescription")}
        </p>
      ) : (
        <div className="ml-6 mt-2 flex min-w-0 flex-wrap gap-x-3 gap-y-1">
          {agents.map((agent) => (
            <div key={agent.id} className="flex min-w-0 items-center gap-1.5">
              <span
                aria-hidden="true"
                className={cn(
                  "size-1.5 shrink-0 rounded-full",
                  agent.status === "online" ? "bg-success" : "bg-warning",
                )}
              />
              <span className={textRole.caption}>{agent.name}</span>
              {agent.address && (
                <span className={textRole.caption}>{agent.address}</span>
              )}
              {agent.status === "registered" && (
                <span className={textRole.caption}>{t("waitingHeartbeat")}</span>
              )}
            </div>
          ))}
        </div>
      )}

      {isTokenExpired && !endReason && (
        <p className={cn("ml-6 mt-1", textRole.caption)}>{t("expiredObserving")}</p>
      )}

      {endReason && (
        <p
          className={cn("ml-6 mt-1", textRole.caption)}
          data-testid="agent-install-observation-ended"
        >
          {endReason === "allOnline"
            ? t("completedDescription")
            : hasSuccessfulProjection
              ? t("deadlineDescription")
              : t("deadlineEmptyDescription")}
        </p>
      )}

      {hasDetectionError && !endReason && (
        <div className={cn("ml-6 mt-1 flex items-center gap-1.5", textRole.caption, getStatusToneTextClass("warning"))}>
          <WarningIcon className="size-3.5 shrink-0" />
          <span>
            {isLastKnown && lastSuccessTime
              ? t("lastKnownError", { time: lastSuccessTime })
              : isInitialError
                ? t("initialError")
                : t("detectionError")}
          </span>
        </div>
      )}
    </div>
  )

  return (
    <div className="mt-4 min-h-14 border-t pt-4" aria-live="polite">
      <StatusTransition phase={phase} reducedMotion={prefersReducedMotion ?? false}>
        {content}
      </StatusTransition>
    </div>
  )
}

function ConnectionMark({
  phase,
  endReason,
  className,
}: {
  phase: AgentInstallConnectionPhase
  endReason: AgentInstallConnectionViewState["endReason"]
  className?: string
}) {
  if (endReason === "deadline") return <WarningIcon className={className} />
  if (phase === "online") return <OnlineIcon className={className} />
  if (phase === "registered") return <AgentIcon className={className} />
  return <Loader2 className={cn("animate-spin motion-reduce:animate-none", className)} />
}

function StatusTransition({
  phase,
  reducedMotion,
  children,
}: {
  phase: AgentInstallConnectionPhase
  reducedMotion: boolean
  children: ReactNode
}) {
  if (reducedMotion) return children

  return (
    <AnimatePresence mode="wait" initial={false}>
      <motion.div
        key={phase}
        initial={{ opacity: 0, y: 2 }}
        animate={{ opacity: 1, y: 0 }}
        exit={{ opacity: 0, y: -2 }}
        transition={{ duration: 0.16, ease: [0.22, 1, 0.36, 1] }}
      >
        {children}
      </motion.div>
    </AnimatePresence>
  )
}
