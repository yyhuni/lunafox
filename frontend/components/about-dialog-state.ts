import React from "react"
import { useQueryClient } from "@tanstack/react-query"
import { useTranslations } from "next-intl"
import { useVersion } from "@/hooks/use-version"
import { VersionService } from "@/services/version.service"
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
  logoSrc: string
  handleCheckUpdate: () => Promise<void>
}

interface UseAboutDialogStateOptions {
  enabled?: boolean
}

export function useAboutDialogState({ enabled = true }: UseAboutDialogStateOptions = {}): AboutDialogState {
  const t = useTranslations("about")
  const { data: versionData } = useVersion({ enabled })
  const queryClient = useQueryClient()

  const [isChecking, setIsChecking] = React.useState(false)
  const [updateResult, setUpdateResult] = React.useState<UpdateCheckResult | null>(null)
  const [checkError, setCheckError] = React.useState<string | null>(null)

  const handleCheckUpdate = React.useCallback(async () => {
    setIsChecking(true)
    setCheckError(null)
    try {
      const result = await VersionService.checkUpdate()
      setUpdateResult(result)
      queryClient.setQueryData(["check-update"], result)
    } catch {
      setCheckError(t("checkFailed"))
    } finally {
      setIsChecking(false)
    }
  }, [queryClient, t])

  const currentVersion = updateResult?.currentVersion || versionData?.version || "-"
  const latestVersion = updateResult?.latestVersion
  const hasUpdate = updateResult?.hasUpdate
  const releaseUrl = updateResult?.releaseUrl
  const logoSrc = "/images/icon-64.png"

  return {
    t,
    isChecking,
    updateResult,
    checkError,
    currentVersion,
    latestVersion,
    hasUpdate,
    releaseUrl,
    logoSrc,
    handleCheckUpdate,
  }
}
