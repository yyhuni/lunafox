import React from "react"
import { useRouter } from "next/navigation"
import { useTranslations } from "next-intl"

import {
  getUpgradeErrorMessage,
  readStoredUpgradeOperationId,
  useCheckForUpdatesAction,
  useCreateUpgradeOperation,
  useRetryUpgradeOperation,
  useUpgradeOperation,
  useVersion,
} from "@/hooks/use-version"
import { replaceWithRouteProgress } from "@/components/route-progress"
import type { ReleaseManifestSummary, UpdateCheckResult } from "@/types/version.types"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

export type AboutDialogState = {
  t: TranslationFn
  isChecking: boolean
  updateResult: UpdateCheckResult | null
  checkError: string | null
  currentVersion: string
  hasUpdate?: boolean
  candidate?: ReleaseManifestSummary
  operation: ReturnType<typeof useUpgradeOperation>
  isCreating: boolean
  isRetrying: boolean
  confirmOpen: boolean
  setConfirmOpen: (open: boolean) => void
  handleCheckUpdate: () => Promise<void>
  handleStartUpgrade: () => void
  handleConfirmUpgrade: (acknowledged: boolean) => void
  handleRetry: () => void
  handleViewStatus: () => void
}

interface UseAboutDialogStateOptions {
  enabled?: boolean
  onUpgradeAccepted?: () => void
}

type AppRouter = ReturnType<typeof useRouter>

// The state hook is also exercised in isolation by contract tests without an
// App Router provider. Production callers always run under Next's router; the
// fallback keeps those pure hook consumers from failing before an action runs.
function useOptionalAppRouter(): AppRouter | null {
  try {
    return useRouter()
  } catch {
    return null
  }
}

export function useAboutDialogState({ enabled = true, onUpgradeAccepted }: UseAboutDialogStateOptions = {}): AboutDialogState {
  const t = useTranslations("about")
  const router = useOptionalAppRouter()
  const { data: versionData } = useVersion({ enabled })
  const checkUpdate = useCheckForUpdatesAction()
  const createMutation = useCreateUpgradeOperation()
  const retryMutation = useRetryUpgradeOperation()
  const operation = useUpgradeOperation(createMutation.operationId ?? readStoredUpgradeOperationId(), { enabled })

  const [isChecking, setIsChecking] = React.useState(false)
  const [updateResult, setUpdateResult] = React.useState<UpdateCheckResult | null>(null)
  const [checkError, setCheckError] = React.useState<string | null>(null)
  const [confirmOpen, setConfirmOpen] = React.useState(false)
  // React Query's pending flag is updated on the next render. Keep a local
  // synchronous fence so two same-tick confirmations cannot create two
  // server operations before that render happens.
  const createInFlightRef = React.useRef(false)

  const handleCheckUpdate = React.useCallback(async () => {
    if (isChecking) return
    setIsChecking(true)
    setCheckError(null)
    try {
      const result = await checkUpdate()
      setUpdateResult(result)
      setCheckError(!result.eligible && result.diagnostic?.reason ? result.diagnostic.reason : null)
    } catch (error) {
      setCheckError(getUpgradeErrorMessage(error) || t("checkFailed"))
    } finally {
      setIsChecking(false)
    }
  }, [checkUpdate, isChecking, t])

  const handleStartUpgrade = React.useCallback(() => {
    const candidate = updateResult?.candidate
    if (!candidate || !updateResult.hasUpdate || !updateResult.eligible || createMutation.isPending || operation.isActive) return
    setConfirmOpen(true)
  }, [createMutation.isPending, operation.isActive, updateResult])

  const handleConfirmUpgrade = React.useCallback((acknowledged: boolean) => {
    const candidate = updateResult?.candidate
    if (!acknowledged || !candidate || !updateResult?.eligible || createMutation.isPending || createInFlightRef.current) return
    createInFlightRef.current = true
    try {
      createMutation.mutate({
        requestId: crypto.randomUUID(),
        manifestId: candidate.manifestId,
        manifestDigest: candidate.manifestDigest,
        confirmed: true,
      }, {
        onSuccess: () => {
          setConfirmOpen(false)
          onUpgradeAccepted?.()
          if (router) replaceWithRouteProgress(router, "/system-upgrade/")
        },
        onError: (error) => setCheckError(getUpgradeErrorMessage(error)),
        onSettled: () => { createInFlightRef.current = false },
      })
    } catch (error) {
      createInFlightRef.current = false
      setCheckError(getUpgradeErrorMessage(error))
    }
  }, [createMutation, onUpgradeAccepted, router, updateResult])

  const handleRetry = React.useCallback(() => {
    if (!operation.operationId || retryMutation.isPending) return
    retryMutation.mutate(operation.operationId, {
      onSuccess: () => {
        onUpgradeAccepted?.()
        if (router) replaceWithRouteProgress(router, "/system-upgrade/")
      },
      onError: (error) => setCheckError(getUpgradeErrorMessage(error)),
    })
  }, [onUpgradeAccepted, operation.operationId, retryMutation, router])

  const handleViewStatus = React.useCallback(() => {
    if (router) replaceWithRouteProgress(router, "/system-upgrade/")
  }, [router])

  const currentVersion = updateResult?.currentVersion || versionData?.version || process.env.NEXT_PUBLIC_IMAGE_TAG?.trim() || "-"
  return {
    t,
    isChecking,
    updateResult,
    checkError: checkError || (operation.isError && !operation.isReconnecting ? getUpgradeErrorMessage(operation.error) : null),
    currentVersion,
    hasUpdate: updateResult?.hasUpdate,
    candidate: updateResult?.candidate,
    operation,
    isCreating: createMutation.isPending,
    isRetrying: retryMutation.isPending,
    confirmOpen,
    setConfirmOpen,
    handleCheckUpdate,
    handleStartUpgrade,
    handleConfirmUpgrade,
    handleRetry,
    handleViewStatus,
  }
}
