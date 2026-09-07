"use client"

import { ConfirmDialogLayout } from "@/components/shared/feedback/confirm-dialog-sections"
import { useConfirmDialogState } from "@/components/shared/feedback/confirm-dialog-state"

interface ConfirmDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  description: string
  onConfirm: () => void | Promise<void>
  loading?: boolean
  variant?: "default" | "destructive"
  confirmText?: string
  cancelText?: string
  processingText?: string
}

export function ConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  onConfirm,
  loading = false,
  variant = "default",
  confirmText,
  cancelText,
  processingText,
}: ConfirmDialogProps) {
  const state = useConfirmDialogState({
    confirmText,
    cancelText,
    processingText,
  })

  return (
    <ConfirmDialogLayout
      state={state}
      open={open}
      onOpenChange={onOpenChange}
      title={title}
      description={description}
      onConfirm={onConfirm}
      loading={loading}
      variant={variant}
    />
  )
}
