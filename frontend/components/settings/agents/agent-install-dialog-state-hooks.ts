import { useCallback, useEffect, useRef, useState } from "react"
import { toastFeedback } from "@/lib/toast-helpers"
import { copyTextToClipboard } from "@/components/shared/feedback/copy-button"

const COPY_TOAST_ID = "install-command-copy"

type UseInstallCommandCopyProps = {
  open: boolean
  dialogRef: React.RefObject<HTMLDivElement | null>
  tToast: (key: string, params?: Record<string, string | number | Date>) => string
}

export function useInstallCommandCopy({
  open,
  dialogRef,
  tToast,
}: UseInstallCommandCopyProps) {
  const [copied, setCopied] = useState(false)
  const copyResetRef = useRef<number | null>(null)

  useEffect(() => {
    if (copyResetRef.current) {
      window.clearTimeout(copyResetRef.current)
      copyResetRef.current = null
    }
    setCopied(false)
  }, [open])

  void dialogRef

  const handleCopy = useCallback(async (text: string) => {
    if (!text) {
      toastFeedback.error(tToast("copyFailed"), { id: COPY_TOAST_ID })
      return
    }
    const success = await copyTextToClipboard(text)
    if (success) {
      if (copyResetRef.current) {
        window.clearTimeout(copyResetRef.current)
      }
      setCopied(true)
      copyResetRef.current = window.setTimeout(() => {
        setCopied(false)
        copyResetRef.current = null
      }, 2000)
      toastFeedback.success(tToast("copied"), { id: COPY_TOAST_ID })
    } else {
      setCopied(false)
      toastFeedback.error(tToast("copyFailed"), { id: COPY_TOAST_ID })
    }
  }, [tToast])

  return {
    copied,
    handleCopy,
  }
}
