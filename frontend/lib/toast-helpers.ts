import type * as React from "react"
import { useTranslations } from "next-intl"
import { toast, type ExternalToast, type ToasterProps } from "sonner"

import { DEFAULT_ERROR_KEY, getErrorI18nKey } from "./error-code-map"

export type ToastId = string | number
export type ToastParams = Record<string, string | number>
export type ToastOptions = ExternalToast
export type ToastPosition = NonNullable<ToasterProps["position"]>

export interface ToastMessages {
  success: (key: string, params?: ToastParams, toastId?: string) => void
  error: (key: string, params?: ToastParams, toastId?: string) => void
  errorFromCode: (code: string | null, fallbackKey?: string, toastId?: string) => void
  loading: (key: string, params?: ToastParams, toastId?: string) => void
  warning: (key: string, params?: ToastParams, toastId?: string) => void
  dismiss: (toastId: string) => void
}

function optionsFromToastId(toastId?: string): ToastOptions | undefined {
  return toastId ? { id: toastId } : undefined
}

function callToast(
  show: (message: React.ReactNode, options?: ToastOptions) => ToastId,
  message: React.ReactNode,
  options?: ToastOptions
) {
  return options ? show(message, options) : show(message)
}

export const toastFeedback = {
  success: (message: React.ReactNode, options?: ToastOptions) => callToast(toast.success, message, options),
  error: (message: React.ReactNode, options?: ToastOptions) => callToast(toast.error, message, options),
  warning: (message: React.ReactNode, options?: ToastOptions) => callToast(toast.warning, message, options),
  loading: (message: React.ReactNode, options?: ToastOptions) => callToast(toast.loading, message, options),
  custom: (render: (toastId: ToastId) => React.ReactElement, options?: ToastOptions) =>
    options ? toast.custom(render, options) : toast.custom(render),
  dismiss: (toastId?: ToastId) => toast.dismiss(toastId),
}

export function useToastMessages(): ToastMessages {
  const t = useTranslations()

  return {
    success: (key, params, toastId) => {
      toastFeedback.success(t(key, params), optionsFromToastId(toastId))
    },
    error: (key, params, toastId) => {
      toastFeedback.error(t(key, params), optionsFromToastId(toastId))
    },
    errorFromCode: (code, fallbackKey = DEFAULT_ERROR_KEY, toastId) => {
      toastFeedback.error(t(code ? getErrorI18nKey(code) : fallbackKey), optionsFromToastId(toastId))
    },
    loading: (key, params, toastId) => {
      toastFeedback.loading(t(key, params), optionsFromToastId(toastId))
    },
    warning: (key, params, toastId) => {
      toastFeedback.warning(t(key, params), optionsFromToastId(toastId))
    },
    dismiss: (toastId) => {
      toastFeedback.dismiss(toastId)
    },
  }
}
