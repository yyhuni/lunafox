import { useTranslations } from "next-intl"

interface ConfirmDialogStateOptions {
  confirmText?: string
  cancelText?: string
  processingText?: string
}

export function useConfirmDialogState({
  confirmText,
  cancelText,
  processingText,
}: ConfirmDialogStateOptions) {
  const t = useTranslations("common.actions")

  return {
    confirmLabel: confirmText || t("confirm"),
    cancelLabel: cancelText || t("cancel"),
    processingLabel: processingText || t("processing"),
  }
}

export type ConfirmDialogState = ReturnType<typeof useConfirmDialogState>
