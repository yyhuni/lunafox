import * as React from "react"

import {
  authTokenRefreshDelayMs,
  ensureFreshAuthToken,
} from "@/services/auth.service"

const MINIMUM_RESCHEDULE_DELAY_MS = 1_000

export function sessionRenewalDelayMs(token: string, now = Date.now()): number | null {
  const delay = authTokenRefreshDelayMs(token, now)
  return delay === null ? null : Math.max(MINIMUM_RESCHEDULE_DELAY_MS, delay)
}

/**
 * Keeps the authenticated shell ahead of JWT expiry. It lives at the shell
 * boundary because dynamically loaded consumers such as the notification drawer
 * cannot reliably own the lifetime of the browser session.
 */
export function useSessionRenewal(): void {
  React.useEffect(() => {
    let disposed = false
    let renewalTimer: ReturnType<typeof setTimeout> | undefined

    const clearRenewalTimer = () => {
      if (renewalTimer !== undefined) {
        clearTimeout(renewalTimer)
        renewalTimer = undefined
      }
    }

    const scheduleRenewal = (token: string | null) => {
      clearRenewalTimer()
      if (disposed || !token) {
        return
      }

      const delay = sessionRenewalDelayMs(token)
      if (delay === null) {
        return
      }

      renewalTimer = setTimeout(() => {
        renewalTimer = undefined
        void checkFreshness()
      }, delay)
    }

    const checkFreshness = async () => {
      if (disposed) {
        return
      }

      try {
        const token = await ensureFreshAuthToken()
        if (!disposed) {
          scheduleRenewal(token)
        }
      } catch {
        // renewPrimaryToken owns the terminal token cleanup and login redirect.
        // Do not keep a timer alive that would create a renewal retry loop.
        clearRenewalTimer()
      }
    }

    const handleVisibilityChange = () => {
      if (document.visibilityState === "visible") {
        void checkFreshness()
      }
    }

    const handleOnline = () => {
      void checkFreshness()
    }

    document.addEventListener("visibilitychange", handleVisibilityChange)
    window.addEventListener("online", handleOnline)
    void checkFreshness()

    return () => {
      disposed = true
      clearRenewalTimer()
      document.removeEventListener("visibilitychange", handleVisibilityChange)
      window.removeEventListener("online", handleOnline)
    }
  }, [])
}
