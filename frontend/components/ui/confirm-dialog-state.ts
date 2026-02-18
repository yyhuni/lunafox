import { useTranslations } from "next-intl"

interface ConfirmDialogStateOptions {
  confirmText?: string
  cancelText?: string
}

export function useConfirmDialogState({
  confirmText,
  cancelText,
}: ConfirmDialogStateOptions) {
  const t = useTranslations("common.actions")

  return {
    confirmLabel: confirmText || t("confirm"),
    cancelLabel: cancelText || t("cancel"),
    processingLabel: t("processing"),
  }
}

export type ConfirmDialogState = ReturnType<typeof useConfirmDialogState>
