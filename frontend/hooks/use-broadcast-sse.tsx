"use client"

import { useEffect, useRef, useCallback } from "react"
import { toast } from "sonner"

/**
 * LunaFox SSE Broadcast Push Hook
 * Monitor the fixed Deno SSE service and pop up a Toast when receiving a broadcast message
 */

// Fixed SSE service address - only this URL is allowed
const SSE_URL = "https://lunafox-sse-push.yyhunisec.deno.net/sse"

// localStorage key - used for "do not remind again" function
const SUPPRESS_KEY = "lunafox:broadcast-suppress"

// Suppression time (milliseconds) - No pop-ups will occur within 24 hours after clicking Don't remind
const SUPPRESS_DURATION = 24 * 60 * 60 * 1000

interface BroadcastMessage {
  type: "broadcast" | "heartbeat" | "connected"
  icon?: string
  title?: string
  description?: string
  primaryAction?: { label: string; href?: string }
  secondaryAction?: { label: string; href?: string } | null
  timestamp?: number
}

let nudgeToastCardLoader: Promise<
  (typeof import("@/components/nudges/nudge-toast-card"))["NudgeToastCard"]
> | null = null

function loadNudgeToastCard() {
  if (!nudgeToastCardLoader) {
    nudgeToastCardLoader = import("@/components/nudges/nudge-toast-card").then(
      (mod) => mod.NudgeToastCard
    )
  }
  return nudgeToastCardLoader
}

function isSuppressed(): boolean {
  try {
    const raw = localStorage.getItem(SUPPRESS_KEY)
    if (!raw) return false
    const until = Number(raw)
    if (Date.now() < until) return true
    localStorage.removeItem(SUPPRESS_KEY)
    return false
  } catch {
    return false
  }
}

function suppress() {
  try {
    localStorage.setItem(SUPPRESS_KEY, String(Date.now() + SUPPRESS_DURATION))
  } catch {
    // ignore
  }
}

// Fixed toast ID for new messages to overwrite old popups
const BROADCAST_TOAST_ID = "lunafox-broadcast"

export function useBroadcastSSE() {
  const eventSourceRef = useRef<EventSource | null>(null)
  const reconnectTimerRef = useRef<number | null>(null)
  const reconnectAttemptsRef = useRef(0)
  const MAX_RECONNECT_DELAY = 60000 // Maximum 60 seconds

  const showBroadcastToast = useCallback((data: BroadcastMessage) => {
    // Check if suppressed
    if (isSuppressed()) return

    // Close the previous pop-up window first
    toast.dismiss(BROADCAST_TOAST_ID)

    // Show new popups with a slight delay, ensuring old ones are closed
    setTimeout(() => {
      void loadNudgeToastCard().then((NudgeToastCard) => {
        toast.custom(
          (t) => (
            <NudgeToastCard
              title={data.title || "系统通知"}
              description={data.description || ""}
              icon={<span className="text-2xl">{data.icon || "📢"}</span>}
              primaryAction={{
                label: data.primaryAction?.label || "知道了",
                href: data.primaryAction?.href,
                onClick: () => toast.dismiss(t),
              }}
              secondaryAction={
                data.secondaryAction
                  ? {
                      label: data.secondaryAction.label,
                      href: data.secondaryAction.href,
                      buttonVariant: "outline",
                      onClick: () => {
                        suppress()
                        toast.dismiss(t)
                      },
                    }
                  : undefined
              }
              onDismiss={() => toast.dismiss(t)}
            />
          ),
          {
            id: BROADCAST_TOAST_ID,
            duration: 15000,
            position: "bottom-right",
          }
        )
      })
    }, 100)
  }, [])

  const connect = useCallback(() => {
    // Prevent duplicate connections
    if (eventSourceRef.current?.readyState === EventSource.OPEN) {
      return
    }

    // Clean up old connections
    if (eventSourceRef.current) {
      eventSourceRef.current.close()
    }

    try {
      const es = new EventSource(SSE_URL)
      eventSourceRef.current = es

      es.onopen = () => {
        // Connection successful, reset retry counter
        reconnectAttemptsRef.current = 0
      }

      es.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data) as BroadcastMessage

          // Only handle broadcast messages
          if (data.type === "broadcast") {
            showBroadcastToast(data)
          }
          // heartbeat and connected messages ignored
        } catch {
          // JSON parsing failed, ignored
        }
      }

      es.onerror = () => {
        es.close()
        eventSourceRef.current = null

        // Exponential backoff: 5s, 10s, 20s, 40s, 60s (max)
        const delay = Math.min(
          5000 * Math.pow(2, reconnectAttemptsRef.current),
          MAX_RECONNECT_DELAY
        )
        reconnectAttemptsRef.current++

        reconnectTimerRef.current = window.setTimeout(() => {
          connect()
        }, delay)
      }
    } catch {
      // Connection failed, ignored
    }
  }, [showBroadcastToast])

  const disconnect = useCallback(() => {
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current)
      reconnectTimerRef.current = null
    }

    if (eventSourceRef.current) {
      eventSourceRef.current.close()
      eventSourceRef.current = null
    }
  }, [])

  // The component is connected when it is mounted and disconnected when it is unmounted.
  useEffect(() => {
    connect()
    return () => disconnect()
  }, [connect, disconnect])

  return { connect, disconnect }
}
