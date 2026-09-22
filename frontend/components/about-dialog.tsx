"use client"

import { useState, type ReactElement } from "react"
import { Dialog, DialogContent, DialogTrigger } from "@/components/ui/dialog"
import { useAboutDialogState } from "@/components/about-dialog-state"
import {
  AboutDialogBranding,
  AboutDialogFooter,
  AboutDialogHeader,
  AboutDialogLinks,
  AboutDialogUpgradeConfirmation,
  AboutDialogVersionInfo,
  Separator,
} from "@/components/about-dialog-sections"

interface AboutDialogProps {
  children: ReactElement
}

export function AboutDialog({ children }: AboutDialogProps) {
  const [open, setOpen] = useState(false)
  const [acknowledged, setAcknowledged] = useState(false)
  const state = useAboutDialogState({
    enabled: open,
    onUpgradeAccepted: () => {
      setOpen(false)
      setAcknowledged(false)
    },
  })

  const handleOpenChange = (nextOpen: boolean) => {
    // Nested confirmation is still inside this Dialog. Dismissing the about
    // shell first would also take the upgrade confirm with it.
    if (!nextOpen && state.confirmOpen) {
      state.setConfirmOpen(false)
      setAcknowledged(false)
      return
    }
    setOpen(nextOpen)
    if (!nextOpen) setAcknowledged(false)
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger render={children} />
      <DialogContent className="max-h-[calc(100dvh-2rem)] overflow-y-auto sm:max-w-md">
        <AboutDialogHeader t={state.t} />
        <div className="space-y-4">
          <AboutDialogBranding t={state.t} />
          <AboutDialogVersionInfo
            t={state.t}
            githubRepo={state.githubRepo}
            currentVersion={state.currentVersion}
            candidate={state.candidate}
            hasUpdate={state.hasUpdate}
            checkError={state.checkError}
            isChecking={state.isChecking}
            isCreating={state.isCreating}
            isRetrying={state.isRetrying}
            canStartUpgrade={Boolean(state.candidate && state.hasUpdate && state.updateResult?.eligible)}
            operation={state.operation}
            onCheckUpdate={() => void state.handleCheckUpdate()}
            onStartUpgrade={state.handleStartUpgrade}
            onRetry={state.handleRetry}
            onViewStatus={state.handleViewStatus}
          />
          <Separator />
          <AboutDialogLinks t={state.t} />
          <AboutDialogFooter t={state.t} />
        </div>
      </DialogContent>
      <AboutDialogUpgradeConfirmation
        t={state.t}
        open={state.confirmOpen}
        candidate={state.candidate}
        acknowledged={acknowledged}
        pending={state.isCreating}
        onOpenChange={(nextOpen) => {
          state.setConfirmOpen(nextOpen)
          if (!nextOpen) setAcknowledged(false)
        }}
        onAcknowledgedChange={setAcknowledged}
        onConfirm={() => state.handleConfirmUpgrade(acknowledged)}
      />
    </Dialog>
  )
}
