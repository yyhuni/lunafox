import { useMemo, useRef } from "react"
import {
  buildInstallCommand,
  normalizeOrigin,
} from "@/lib/agent-install-helpers"
import type { RegistrationTokenResponse } from "@/types/agent.types"
import {
  useInstallCommandCopy,
} from "@/components/settings/workers/agent-install-dialog-state-hooks"

const FALLBACK_SERVER_URL = "https://your-lunafox-server"
type UseAgentInstallDialogStateProps = {
  open: boolean
  token: RegistrationTokenResponse | null
  tToast: (key: string, params?: Record<string, string | number | Date>) => string
}

export function useAgentInstallDialogState({
  open,
  token,
  tToast,
}: UseAgentInstallDialogStateProps) {
  const dialogRef = useRef<HTMLDivElement>(null)
  const safeServerUrl = typeof window !== "undefined" ? window.location.origin : FALLBACK_SERVER_URL
  const scriptBaseURL = normalizeOrigin(safeServerUrl)
  const { copied, handleCopy } = useInstallCommandCopy({
    open,
    dialogRef,
    tToast,
  })

  const hasToken = Boolean(token?.token)
  const tokenExpiresAt = useMemo(() => {
    if (!token?.expiresAt) return 0
    const timestamp = new Date(token.expiresAt).getTime()
    return Number.isNaN(timestamp) ? 0 : timestamp
  }, [token])
  const isTokenValid = hasToken && tokenExpiresAt > Date.now()

  const installCommand = useMemo(() => {
    return buildInstallCommand(token?.token ?? "", scriptBaseURL)
  }, [token, scriptBaseURL])

  const commandToCopy = installCommand

  const canCopyCommand = useMemo(() => {
    return Boolean(commandToCopy)
  }, [commandToCopy])

  const showLocalOnlyWarning = useMemo(() => {
    try {
      const hostname = new URL(scriptBaseURL).hostname
      return hostname === "localhost" || hostname === "127.0.0.1" || hostname === "::1"
    } catch {
      return false
    }
  }, [scriptBaseURL])

  return {
    dialogRef,
    copied,
    hasToken,
    tokenExpiresAt,
    isTokenValid,
    handleCopy,
    installCommand,
    canCopyCommand,
    showLocalOnlyWarning,
  }
}
