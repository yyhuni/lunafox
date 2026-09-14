import React from "react"
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
}

interface UseAboutDialogStateOptions {
  enabled?: boolean
}

export function useAboutDialogState({ enabled = true }: UseAboutDialogStateOptions = {}): AboutDialogState {
  const t = useTranslations("about")
  const { data: versionData } = useVersion({ enabled: false })
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
      setUpdateResult(await checkUpdate())
    } catch (error) {
      setCheckError(getUpgradeErrorMessage(error) || t("checkFailed"))
    } finally {
      setIsChecking(false)
    }
  }, [checkUpdate, isChecking, t])

  const handleStartUpgrade = React.useCallback(() => {
    const candidate = updateResult?.candidate
    if (!candidate || !updateResult.hasUpdate || !updateResult.eligible || createMutation.isPending || operation.data) return
    setConfirmOpen(true)
  }, [createMutation.isPending, operation.data, updateResult])

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
        onSuccess: () => setConfirmOpen(false),
        onError: (error) => setCheckError(getUpgradeErrorMessage(error)),
        onSettled: () => { createInFlightRef.current = false },
      })
    } catch (error) {
      createInFlightRef.current = false
      setCheckError(getUpgradeErrorMessage(error))
    }
  }, [createMutation, updateResult])

  const handleRetry = React.useCallback(() => {
    if (!operation.operationId || retryMutation.isPending) return
    retryMutation.mutate(operation.operationId, {
      onError: (error) => setCheckError(getUpgradeErrorMessage(error)),
    })
  }, [operation.operationId, retryMutation])

  const currentVersion = updateResult?.currentVersion || versionData?.version || "-"
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
  }
}
