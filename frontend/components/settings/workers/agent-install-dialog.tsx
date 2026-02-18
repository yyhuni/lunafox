"use client"

import { useAgentInstallDialogState } from "@/components/settings/workers/agent-install-dialog-state"
import { useTranslations } from "next-intl"
import { DialogClose, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { useFormatRelativeTime } from "@/lib/i18n-format"
import { Button } from "@/components/ui/button"
import type { RegistrationTokenResponse } from "@/types/agent.types"
import {
  AgentInstallCommandPanel,
  AgentInstallCommandTips,
  AgentInstallTokenCard,
} from "@/components/settings/workers/agent-install-dialog-sections"

type AgentInstallDialogProps = {
  open: boolean
  token: RegistrationTokenResponse | null
  isGenerating: boolean
  onGenerate: () => void
}

export function AgentInstallDialog({
  open,
  token,
  isGenerating,
  onGenerate,
}: AgentInstallDialogProps) {
  const t = useTranslations("settings.workers")
  const tActions = useTranslations("common.actions")
  const tToast = useTranslations("toast")
  const formatRelativeTime = useFormatRelativeTime()
  const {
    dialogRef,
    copied,
    hasToken,
    isTokenValid,
    handleCopy,
    installCommand,
    canCopyCommand,
    showLocalOnlyWarning,
  } = useAgentInstallDialogState({
    open,
    token,
    tToast,
  })

  return (
    <DialogContent
      ref={dialogRef}
      className="sm:max-w-[860px] max-h-[calc(100vh-2rem)] overflow-hidden p-0 gap-0 flex flex-col"
    >
      <DialogHeader className="shrink-0 border-b px-6 pt-6 pb-4 pr-12">
        <DialogTitle>{t("install.title")}</DialogTitle>
        <DialogDescription>{t("install.desc")}</DialogDescription>
      </DialogHeader>
      <div className="flex-1 min-h-0 overflow-y-auto px-6 py-4 space-y-4">
        <AgentInstallTokenCard
          t={t}
          token={token}
          hasToken={hasToken}
          isTokenValid={isTokenValid}
          isGenerating={isGenerating}
          formatRelativeTime={formatRelativeTime}
          onGenerate={onGenerate}
        />

        <AgentInstallCommandPanel
          t={t}
          tActions={tActions}
          tToast={tToast}
          token={token}
          installCommand={installCommand}
          isGenerating={isGenerating}
          canCopyCommand={canCopyCommand}
          copied={copied}
          onCopy={handleCopy}
        />
        {showLocalOnlyWarning && (
          <div className="rounded-lg border border-amber-500/30 bg-amber-500/5 p-3">
            <p className="text-[11px] text-amber-700 dark:text-amber-400">{t("install.localOnlyWarning")}</p>
          </div>
        )}
        <AgentInstallCommandTips t={t} />

        <div className="flex justify-end pt-1">
          <DialogClose asChild>
            <Button size="sm">{tActions("close")}</Button>
          </DialogClose>
        </div>
      </div>
    </DialogContent>
  )
}
