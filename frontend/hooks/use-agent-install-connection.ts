"use client"

import { useEffect, useState } from "react"
import { useQueryClient } from "@tanstack/react-query"

import { agentKeys, useRegistrationToken } from "@/hooks/use-agents"
import {
  EMPTY_AGENT_INSTALL_CONNECTION_STATE,
  projectAgentInstallConnection,
  type AgentInstallConnectionState,
} from "@/lib/agent-install-connection"
import type { RegistrationTokenResponse } from "@/types/agent.types"

export const AGENT_INSTALL_POLL_INTERVAL_MS = 2_000
export const AGENT_INSTALL_POST_EXPIRY_GRACE_MS = 2 * 60_000

export type AgentInstallObservationEndReason = "allOnline" | "deadline"

type RegistrationTokenIdentity = Pick<RegistrationTokenResponse, "resourceName" | "expiresAt">

interface AgentInstallSessionState extends AgentInstallConnectionState {
  resourceName: string
  hasSuccessfulProjection: boolean
  hasDetectionError: boolean
  lastSuccessfulAt: number | null
  endReason: AgentInstallObservationEndReason | null
  tokenState: "active" | "expired" | null
}

export interface AgentInstallConnectionViewState extends AgentInstallConnectionState {
  hasSuccessfulProjection: boolean
  hasDetectionError: boolean
  isInitialError: boolean
  isLastKnown: boolean
  lastSuccessfulAt: number | null
  isLive: boolean
  isTokenExpired: boolean
  endReason: AgentInstallObservationEndReason | null
}

const EMPTY_SESSION: AgentInstallSessionState = {
  resourceName: "",
  ...EMPTY_AGENT_INSTALL_CONNECTION_STATE,
  hasSuccessfulProjection: false,
  hasDetectionError: false,
  lastSuccessfulAt: null,
  endReason: null,
  tokenState: null,
}

function emptySession(resourceName: string): AgentInstallSessionState {
  return { ...EMPTY_SESSION, resourceName }
}

function parseRequiredTimestamp(value: string, field: string): number {
  const timestamp = Date.parse(value)
  if (!Number.isFinite(timestamp)) {
    throw new Error(`Invalid ${field}`)
  }
  return timestamp
}

export function useAgentInstallConnection(
  token: RegistrationTokenIdentity | null,
  open: boolean,
): AgentInstallConnectionViewState {
  const queryClient = useQueryClient()
  const [session, setSession] = useState<AgentInstallSessionState>(EMPTY_SESSION)
  const resourceName = token?.resourceName ?? ""
  const expiresAt = token
    ? parseRequiredTimestamp(token.expiresAt, "registration token expiresAt")
    : 0
  const deadline = expiresAt + AGENT_INSTALL_POST_EXPIRY_GRACE_MS
  const current = session.resourceName === resourceName
    ? session
    : emptySession(resourceName)
  const deadlineReached = Boolean(token) && Date.now() >= deadline
  const endReason = current.endReason ?? (deadlineReached ? "deadline" : null)
  const shouldPoll = Boolean(token && open && endReason === null)
  const query = useRegistrationToken(token, {
    enabled: shouldPoll,
    refetchInterval: shouldPoll ? AGENT_INSTALL_POLL_INTERVAL_MS : false,
  })

  useEffect(() => {
    const data = query.data
    if (!data || data.resourceName !== resourceName || query.dataUpdatedAt <= 0) return

    const dataExpiresAt = parseRequiredTimestamp(
      data.expiresAt,
      "registration token resource expiresAt",
    )
    const dataDeadline = dataExpiresAt + AGENT_INSTALL_POST_EXPIRY_GRACE_MS
    const succeededAt = query.dataUpdatedAt

    setSession((previous) => {
      const matching = previous.resourceName === resourceName
        ? previous
        : emptySession(resourceName)
      if (succeededAt >= dataDeadline) {
        return { ...matching, endReason: "deadline" }
      }

      const projection = projectAgentInstallConnection(data.agents)
      const tokenExpired = data.state === "expired" || succeededAt >= dataExpiresAt
      const nextEndReason = tokenExpired
        && projection.agents.length > 0
        && projection.phase === "online"
        ? "allOnline"
        : matching.endReason

      return {
        resourceName,
        ...projection,
        hasSuccessfulProjection: true,
        hasDetectionError: false,
        lastSuccessfulAt: succeededAt,
        endReason: nextEndReason,
        tokenState: data.state,
      }
    })
  }, [query.data, query.dataUpdatedAt, resourceName])

  useEffect(() => {
    if (!resourceName || !query.isError) return

    setSession((previous) => ({
      ...(previous.resourceName === resourceName
        ? previous
        : emptySession(resourceName)),
      hasDetectionError: true,
    }))
  }, [query.errorUpdatedAt, query.isError, resourceName])

  useEffect(() => {
    if (!token || !open || current.endReason) return

    const timeout = window.setTimeout(() => {
      setSession((previous) => {
        const matching = previous.resourceName === resourceName
          ? previous
          : emptySession(resourceName)
        if (matching.endReason === "allOnline") return matching
        return { ...matching, endReason: "deadline" }
      })
    }, Math.max(0, deadline - Date.now()))

    return () => window.clearTimeout(timeout)
  }, [current.endReason, deadline, open, resourceName, token])

  useEffect(() => {
    if (!resourceName || shouldPoll) return
    void queryClient.cancelQueries({
      queryKey: agentKeys.registrationToken({ resourceName }),
      exact: true,
    })
  }, [queryClient, resourceName, shouldPoll])

  useEffect(() => {
    if (!resourceName) return
    const queryKey = agentKeys.registrationToken({ resourceName })
    return () => {
      void queryClient.cancelQueries({ queryKey, exact: true })
    }
  }, [queryClient, resourceName])

  const isTokenExpired = Boolean(token) && (
    current.tokenState === "expired" || Date.now() >= expiresAt
  )

  return {
    phase: current.phase,
    agents: current.agents,
    hasSuccessfulProjection: current.hasSuccessfulProjection,
    hasDetectionError: current.hasDetectionError,
    isInitialError: current.hasDetectionError && !current.hasSuccessfulProjection,
    isLastKnown: current.hasDetectionError && current.hasSuccessfulProjection,
    lastSuccessfulAt: current.lastSuccessfulAt,
    isLive: shouldPoll,
    isTokenExpired,
    endReason,
  }
}
