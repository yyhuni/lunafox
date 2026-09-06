import * as React from "react"
import { useQueryClient } from "@tanstack/react-query"

import { notificationKeys } from "@/hooks/use-notifications"
import { toastFeedback } from "@/lib/toast-helpers"
import { authTokenExpirationMs } from "@/services/auth.service"
import { NotificationService } from "@/services/notification.service"
import type { NotificationInboxItem } from "@/types/notification.types"

const MAX_RECONNECT_DELAY_MS = 30_000
const INITIAL_RECONNECT_DELAY_MS = 1_000
export const NOTIFICATION_STREAM_IDLE_TIMEOUT_MS = 45_000

export type NotificationStreamStatus = "idle" | "connecting" | "connected" | "offline" | "stopped"

export const notificationTokenExpirationMs = authTokenExpirationMs

export function createNotificationNameBaseline(
  notifications: readonly NotificationInboxItem[]
): Set<string> {
  return new Set(notifications.map((notification) => notification.name))
}

export function collectNewScanFailureNotifications(
  observedNames: ReadonlySet<string>,
  notifications: readonly NotificationInboxItem[]
): { observedNames: Set<string>; newFailures: NotificationInboxItem[] } {
  const nextObservedNames = new Set(observedNames)
  const newFailures: NotificationInboxItem[] = []
  for (const notification of notifications) {
    const isNew = !nextObservedNames.has(notification.name)
    nextObservedNames.add(notification.name)
    if (isNew && notification.kind === "scan-failed" && notification.priority === "high") {
      newFailures.push(notification)
    }
  }
  return { observedNames: nextObservedNames, newFailures }
}

export function showScanFailureNotification(notification: NotificationInboxItem): void {
  toastFeedback.error(notification.title, {
    description: notification.message,
    id: notification.name,
  })
}

export function notificationReconnectDelayMs(attempt: number, random = Math.random): number {
  const boundedAttempt = Math.max(0, Math.min(attempt, 5))
  const ceiling = Math.min(
    MAX_RECONNECT_DELAY_MS,
    INITIAL_RECONNECT_DELAY_MS * (2 ** boundedAttempt)
  )
  return Math.floor(Math.max(0, Math.min(1, random())) * ceiling)
}

export async function consumeNotificationRefreshStream(
  response: Response,
  onRefresh: () => Promise<void>,
  onActivity: () => void = () => {}
): Promise<void> {
  const reader = response.body?.getReader()
  if (!reader) {
    throw new Error("Notification stream body is unavailable")
  }

  const decoder = new TextDecoder()
  let buffer = ""
  const consumeEvent = async (rawEvent: string) => {
    const event = rawEvent
      .split("\n")
      .find((line) => line.startsWith("event:"))
      ?.slice("event:".length)
      .trim()
    if (event === "refresh") {
      await onRefresh()
    }
  }

  while (true) {
    const { done, value } = await reader.read()
    if (done) {
      break
    }
    if (value?.byteLength) {
      onActivity()
    }
    buffer = `${buffer}${decoder.decode(value, { stream: true })}`.replace(/\r\n/g, "\n")
    let delimiterIndex = buffer.indexOf("\n\n")
    while (delimiterIndex >= 0) {
      const event = buffer.slice(0, delimiterIndex)
      buffer = buffer.slice(delimiterIndex + 2)
      await consumeEvent(event)
      delimiterIndex = buffer.indexOf("\n\n")
    }
  }
}

function isOnline() {
  return typeof navigator === "undefined" || navigator.onLine
}

export function useNotificationSSE(options?: { enabled?: boolean }) {
  const enabled = options?.enabled ?? true
  const queryClient = useQueryClient()
  const [status, setStatus] = React.useState<NotificationStreamStatus>("idle")

  React.useEffect(() => {
    if (!enabled) {
      setStatus("idle")
      return
    }

    let disposed = false
    let controller: AbortController | null = null
    let reconnectTimer: ReturnType<typeof setTimeout> | undefined
    let expirationTimer: ReturnType<typeof setTimeout> | undefined
    let idleTimer: ReturnType<typeof setTimeout> | undefined
    let reconnectAttempt = 0
    let renewalUsedForCurrentConnection = false
    let expiredLocally = false
    let idleTimedOutLocally = false
    let reconnectWhenReleased = false

    const clearReconnectTimer = () => {
      if (reconnectTimer !== undefined) {
        clearTimeout(reconnectTimer)
        reconnectTimer = undefined
      }
    }

    const clearTimers = () => {
      clearReconnectTimer()
      if (expirationTimer !== undefined) {
        clearTimeout(expirationTimer)
        expirationTimer = undefined
      }
      if (idleTimer !== undefined) {
        clearTimeout(idleTimer)
        idleTimer = undefined
      }
    }

    const refreshInbox = async () => {
      const [inbox] = await Promise.all([
        NotificationService.list({ pageSize: 100 }),
        queryClient.refetchQueries({
          queryKey: notificationKeys.all,
          type: "active",
        }),
      ])
      return inbox.results
    }

    const scheduleReconnect = (connect: () => void) => {
      if (disposed || !isOnline()) {
        return
      }
      const delay = notificationReconnectDelayMs(reconnectAttempt)
      reconnectAttempt += 1
      reconnectTimer = setTimeout(connect, delay)
    }

    const connect = async () => {
      if (disposed || controller || !isOnline()) {
        if (!disposed && !isOnline()) {
          setStatus("offline")
        }
        return
      }

      const currentController = new AbortController()
      controller = currentController
      expiredLocally = false
      idleTimedOutLocally = false
      setStatus("connecting")

      let reconnectImmediately = false
      let freshnessFailed = false
      try {
        let token: string | null
        try {
          token = await NotificationService.getFreshStreamToken()
        } catch {
          freshnessFailed = true
          throw new Error("Notification stream token renewal failed")
        }
        if (currentController.signal.aborted || disposed) {
          return
        }
        if (!token) {
          setStatus("stopped")
          return
        }

        const expiry = notificationTokenExpirationMs(token)
        if (expiry !== null) {
          const timeout = Math.max(0, expiry - Date.now())
          expirationTimer = setTimeout(() => {
            expiredLocally = true
            currentController.abort()
          }, timeout)
        }

        const response = await NotificationService.openStream(token, currentController.signal)
        if (currentController.signal.aborted) {
          return
        }

        if (response.status === 401) {
          const reason = await NotificationService.streamErrorReason(response)
          if (reason === "TOKEN_EXPIRED" && !renewalUsedForCurrentConnection) {
            renewalUsedForCurrentConnection = true
            try {
              await NotificationService.renewStreamToken()
              reconnectImmediately = true
            } catch {
              // A renewal failure is terminal for this stream. The shared auth
              // boundary owns token cleanup and navigation; do not schedule a
              // reconnect loop with credentials that can no longer recover.
              NotificationService.handleStreamSessionFailure()
              setStatus("stopped")
            }
            return
          }
          NotificationService.handleStreamSessionFailure()
          setStatus("stopped")
          return
        }
        if (!response.ok) {
          throw new Error(`Notification stream returned ${response.status}`)
        }
        if (!response.headers.get("content-type")?.toLowerCase().includes("text/event-stream")) {
          throw new Error("Notification stream returned an unexpected content type")
        }

        // A pending fetch can outlive a broken transport, so missing Server
        // heartbeats must actively end the connection before reconnecting.
        const resetIdleWatchdog = () => {
          if (idleTimer !== undefined) {
            clearTimeout(idleTimer)
          }
          idleTimer = setTimeout(() => {
            idleTimedOutLocally = true
            currentController.abort()
          }, NOTIFICATION_STREAM_IDLE_TIMEOUT_MS)
        }

        reconnectAttempt = 0
        renewalUsedForCurrentConnection = false
        setStatus("connected")
        resetIdleWatchdog()
        let observedNames = createNotificationNameBaseline(await refreshInbox())
        const handleRefresh = async () => {
          const refreshed = collectNewScanFailureNotifications(observedNames, await refreshInbox())
          observedNames = refreshed.observedNames
          refreshed.newFailures.forEach(showScanFailureNotification)
        }
        await consumeNotificationRefreshStream(response, handleRefresh, resetIdleWatchdog)

        if (!disposed && !currentController.signal.aborted) {
          setStatus("idle")
          scheduleReconnect(() => {
            void connect()
          })
        }
      } catch {
        if (disposed) {
          return
        }
        if (freshnessFailed) {
          // The shared auth boundary already cleared credentials and redirected
          // after renewal failure; this stream must not schedule a retry loop.
          setStatus("stopped")
          return
        }
        if (currentController.signal.aborted) {
          if ((expiredLocally || idleTimedOutLocally) && isOnline()) {
            setStatus("idle")
            scheduleReconnect(() => {
              void connect()
            })
          }
          return
        }
        setStatus(isOnline() ? "idle" : "offline")
        scheduleReconnect(() => {
          void connect()
        })
      } finally {
        if (controller === currentController) {
          controller = null
        }
        if (expirationTimer !== undefined) {
          clearTimeout(expirationTimer)
          expirationTimer = undefined
        }
        if (idleTimer !== undefined) {
          clearTimeout(idleTimer)
          idleTimer = undefined
        }
        if (reconnectImmediately && !disposed) {
          queueMicrotask(() => {
            void connect()
          })
        } else if (reconnectWhenReleased && !disposed && isOnline()) {
          reconnectWhenReleased = false
          queueMicrotask(() => {
            void connect()
          })
        }
      }
    }

    const handleOnline = () => {
      reconnectAttempt = 0
      clearReconnectTimer()
      if (controller?.signal.aborted) {
        reconnectWhenReleased = true
        return
      }
      if (!controller) {
        void connect()
      }
    }
    const handleOffline = () => {
      reconnectWhenReleased = false
      clearTimers()
      setStatus("offline")
      controller?.abort()
    }

    if (typeof window !== "undefined") {
      window.addEventListener("online", handleOnline)
      window.addEventListener("offline", handleOffline)
    }
    void connect()

    return () => {
      disposed = true
      clearTimers()
      controller?.abort()
      if (typeof window !== "undefined") {
        window.removeEventListener("online", handleOnline)
        window.removeEventListener("offline", handleOffline)
      }
    }
  }, [enabled, queryClient])

  return {
    status,
    isConnected: status === "connected",
  }
}
