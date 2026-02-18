"use client"

import type { WorkerNode } from "@/types/worker.types"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { IconEye, IconRefresh, IconRocket, IconTrash } from "@/components/icons"

type TranslationFn = (key: string) => string

interface DeployTerminalHeaderProps {
  worker: WorkerNode | null
  isConnected: boolean
  onClose: () => void
  t: TranslationFn
  tCommon: TranslationFn
}

export function DeployTerminalHeader({
  worker,
  isConnected,
  onClose,
  t,
  tCommon,
}: DeployTerminalHeaderProps) {
  return (
    <div className="flex items-center justify-between px-4 py-3 bg-[#1a1b26] border-b border-[#32344a]">
      <div className="flex items-center gap-3">
        <div className="flex items-center gap-1.5">
          <button
            type="button"
            onClick={onClose}
            aria-label={tCommon("close")}
            className="w-3 h-3 rounded-full bg-[#ff5f56] hover:bg-[#ff5f56]/80 transition-colors"
            title={tCommon("close")}
          />
          <div className="w-3 h-3 rounded-full bg-[#ffbd2e]" />
          <div className="w-3 h-3 rounded-full bg-[#27c93f]" />
        </div>
        <span className="text-sm text-[#a9b1d6] font-medium">
          {worker?.username}@{worker?.ipAddress}
        </span>
      </div>
      <div className="flex items-center gap-1.5">
        <span className={`w-2 h-2 rounded-full ${isConnected ? "bg-[#9ece6a]" : "bg-[#f7768e]"}`} />
        <span className="text-xs text-[#a9b1d6]">
          {isConnected ? t("terminal.connected") : t("terminal.disconnected")}
        </span>
      </div>
    </div>
  )
}

interface DeployTerminalActionsProps {
  isConnected: boolean
  worker: WorkerNode | null
  currentStatus: string | null
  onReconnect: () => void
  onDeploy: () => void
  onAttach: () => void
  onUninstall: () => void
  tTerminal: TranslationFn
}

export function DeployTerminalActions({
  isConnected,
  worker,
  currentStatus,
  onReconnect,
  onDeploy,
  onAttach,
  onUninstall,
  tTerminal,
}: DeployTerminalActionsProps) {
  return (
    <div className="flex items-center justify-between px-4 py-3 bg-[#1a1b26] border-t border-[#32344a]">
      <div className="text-xs text-[#565f89]">
        {!isConnected && tTerminal("waitingConnection")}
        {isConnected && currentStatus === "pending" && tTerminal("pendingHint")}
        {isConnected && currentStatus === "deploying" && tTerminal("deployingHint")}
        {isConnected && currentStatus === "online" && tTerminal("onlineHint")}
        {isConnected && currentStatus === "offline" && tTerminal("offlineHint")}
        {isConnected && currentStatus === "updating" && tTerminal("updatingHint")}
        {isConnected && currentStatus === "outdated" && tTerminal("outdatedHint")}
      </div>

      <div className="flex items-center gap-2">
        {!isConnected && (
          <button
            type="button"
            onClick={onReconnect}
            className="inline-flex items-center px-3 py-1.5 text-sm rounded-md bg-[#32344a] text-[#a9b1d6] hover:bg-[#414868] transition-colors"
          >
            <IconRefresh className="mr-1.5 h-4 w-4" />
            {tTerminal("reconnect")}
          </button>
        )}
        {isConnected && worker && (
          <>
            {currentStatus === "pending" && (
              <button
                type="button"
                onClick={onDeploy}
                className="inline-flex items-center px-3 py-1.5 text-sm rounded-md bg-[#7aa2f7] text-[#1a1b26] hover:bg-[#7aa2f7]/80 transition-colors"
              >
                <IconRocket className="mr-1.5 h-4 w-4" />
                {tTerminal("startDeploy")}
              </button>
            )}

            {currentStatus === "deploying" && (
              <button
                type="button"
                onClick={onAttach}
                className="inline-flex items-center px-3 py-1.5 text-sm rounded-md bg-[#7aa2f7] text-[#1a1b26] hover:bg-[#7aa2f7]/80 transition-colors"
              >
                <IconEye className="mr-1.5 h-4 w-4" />
                {tTerminal("viewProgress")}
              </button>
            )}

            {currentStatus === "updating" && (
              <button
                type="button"
                onClick={onAttach}
                className="inline-flex items-center px-3 py-1.5 text-sm rounded-md bg-[#e0af68] text-[#1a1b26] hover:bg-[#e0af68]/80 transition-colors"
              >
                <IconEye className="mr-1.5 h-4 w-4" />
                {tTerminal("viewProgress")}
              </button>
            )}

            {currentStatus === "outdated" && (
              <button
                type="button"
                onClick={onDeploy}
                className="inline-flex items-center px-3 py-1.5 text-sm rounded-md bg-[#f7768e] text-[#1a1b26] hover:bg-[#f7768e]/80 transition-colors"
              >
                <IconRocket className="mr-1.5 h-4 w-4" />
                {tTerminal("redeploy")}
              </button>
            )}

            {(currentStatus === "online" || currentStatus === "offline") && (
              <>
                <button
                  type="button"
                  onClick={onDeploy}
                  className="inline-flex items-center px-3 py-1.5 text-sm rounded-md bg-[#32344a] text-[#a9b1d6] hover:bg-[#414868] transition-colors"
                >
                  <IconRocket className="mr-1.5 h-4 w-4" />
                  {tTerminal("redeploy")}
                </button>
                <button
                  type="button"
                  onClick={onUninstall}
                  className="inline-flex items-center px-3 py-1.5 text-sm rounded-md bg-[#32344a] text-[#f7768e] hover:bg-[#414868] transition-colors"
                >
                  <IconTrash className="mr-1.5 h-4 w-4" />
                  {tTerminal("uninstall")}
                </button>
              </>
            )}
          </>
        )}
      </div>
    </div>
  )
}

interface DeployTerminalUninstallDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: () => void
  t: TranslationFn
  tCommon: TranslationFn
}

export function DeployTerminalUninstallDialog({
  open,
  onOpenChange,
  onConfirm,
  t,
  tCommon,
}: DeployTerminalUninstallDialogProps) {
  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t("confirmUninstall")}</AlertDialogTitle>
          <AlertDialogDescription>
            {t("confirmUninstallDesc")}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{tCommon("cancel")}</AlertDialogCancel>
          <AlertDialogAction
            onClick={onConfirm}
            className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
          >
            {tCommon("delete")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
