import React from "react"
import { useTranslations } from "next-intl"
import { useCheckUpdateAction, useVersion } from "@/hooks/use-version"
import type { UpdateCheckResult } from "@/types/version.types"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

type AboutDialogState = {
  t: TranslationFn
  isChecking: boolean
  updateResult: UpdateCheckResult | null
  checkError: string | null
  currentVersion: string
  latestVersion?: string
  hasUpdate?: boolean
  releaseUrl?: string
  handleCheckUpdate: () => Promise<void>
}

interface UseAboutDialogStateOptions {
  enabled?: boolean
}

export function useAboutDialogState({ enabled = true }: UseAboutDialogStateOptions = {}): AboutDialogState {
  const t = useTranslations("about")
  const { data: versionData } = useVersion({ enabled })
  const checkUpdate = useCheckUpdateAction()

  const [isChecking, setIsChecking] = React.useState(false)
  const [updateResult, setUpdateResult] = React.useState<UpdateCheckResult | null>(null)
  const [checkError, setCheckError] = React.useState<string | null>(null)

  const handleCheckUpdate = React.useCallback(async () => {
    setIsChecking(true)
    setCheckError(null)
    try {
      const result = await checkUpdate()
      setUpdateResult(result)
    } catch {
      setCheckError(t("checkFailed"))
    } finally {
      setIsChecking(false)
    }
  }, [checkUpdate, t])

  const currentVersion = updateResult?.currentVersion || versionData?.version || "-"
  const latestVersion = updateResult?.latestVersion
  const hasUpdate = updateResult?.hasUpdate
  const releaseUrl = updateResult?.releaseUrl

  return {
    t,
    isChecking,
    updateResult,
    checkError,
    currentVersion,
    latestVersion,
    hasUpdate,
    releaseUrl,
    handleCheckUpdate,
  }
}
