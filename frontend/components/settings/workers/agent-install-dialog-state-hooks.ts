import { useCallback, useEffect, useRef, useState } from "react"
import { toast } from "sonner"

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

  const copyToClipboard = useCallback(async (text: string): Promise<boolean> => {
    const value = text ?? ""
    if (navigator.clipboard?.writeText) {
      try {
        await navigator.clipboard.writeText(value)
        return true
      } catch {
        // fallback below
      }
    }

    try {
      const container = dialogRef.current || document.body
      const textArea = document.createElement("textarea")
      textArea.value = value
      textArea.style.position = "fixed"
      textArea.style.left = "-9999px"
      textArea.style.top = "-9999px"
      textArea.style.opacity = "0"
      container.appendChild(textArea)
      textArea.focus()
      textArea.select()
      textArea.setSelectionRange(0, textArea.value.length)
      const success = document.execCommand("copy")
      textArea.remove()
      return success
    } catch {
      return false
    }
  }, [dialogRef])

  const handleCopy = useCallback(async (text: string) => {
    if (!text) {
      toast.error(tToast("copyFailed"), { id: COPY_TOAST_ID })
      return
    }
    toast.dismiss("agent-token")
    const success = await copyToClipboard(text)
    if (success) {
      if (copyResetRef.current) {
        window.clearTimeout(copyResetRef.current)
      }
      setCopied(true)
      copyResetRef.current = window.setTimeout(() => {
        setCopied(false)
        copyResetRef.current = null
      }, 2000)
      toast.success(tToast("copied"), { id: COPY_TOAST_ID })
    } else {
      setCopied(false)
      toast.error(tToast("copyFailed"), { id: COPY_TOAST_ID })
    }
  }, [copyToClipboard, tToast])

  return {
    copied,
    handleCopy,
  }
}
