"use client"

import { useTranslations } from "next-intl"
import { Dialog, DialogContent } from "@/components/ui/dialog"
import type { WorkerNode } from "@/types/worker.types"
import { useDeployTerminalDialogState } from "@/components/settings/workers/deploy-terminal-dialog-state"
import {
  DeployTerminalActions,
  DeployTerminalHeader,
  DeployTerminalUninstallDialog,
} from "@/components/settings/workers/deploy-terminal-dialog-sections"

interface DeployTerminalDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  worker: WorkerNode | null
  onDeployComplete?: () => void
}

export function DeployTerminalDialog({
  open,
  onOpenChange,
  worker,
  onDeployComplete,
}: DeployTerminalDialogProps) {
  const t = useTranslations("settings.workers")
  const tCommon = useTranslations("common.actions")
  const tTerminal = useTranslations("settings.workers.terminal")
  const {
    terminalRef,
    isConnected,
    currentStatus,
    uninstallDialogOpen,
    setUninstallDialogOpen,
    handleClose,
    handleDeploy,
    handleAttach,
    handleUninstallClick,
    handleUninstallConfirm,
    connectWs,
  } = useDeployTerminalDialogState({
    open,
    onOpenChange,
    worker,
    onDeployComplete,
    tTerminal,
  })

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="w-[50vw] max-w-[50vw] h-[80vh] flex flex-col p-0 gap-0 overflow-hidden [&>button]:hidden">
        <DeployTerminalHeader
          worker={worker}
          isConnected={isConnected}
          onClose={handleClose}
          t={t}
          tCommon={tCommon}
        />

        {/* xterm terminal container */}
        <div ref={terminalRef} className="flex-1 overflow-hidden bg-[#1a1b26]" />

        <DeployTerminalActions
          isConnected={isConnected}
          worker={worker}
          currentStatus={currentStatus}
          onReconnect={connectWs}
          onDeploy={handleDeploy}
          onAttach={handleAttach}
          onUninstall={handleUninstallClick}
          tTerminal={tTerminal}
        />
      </DialogContent>

      <DeployTerminalUninstallDialog
        open={uninstallDialogOpen}
        onOpenChange={setUninstallDialogOpen}
        onConfirm={handleUninstallConfirm}
        t={t}
        tCommon={tCommon}
      />
    </Dialog>
  )
}
