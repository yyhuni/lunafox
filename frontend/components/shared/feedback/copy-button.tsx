"use client"

import * as React from "react"
import { Check, Copy } from "@/components/icons"
import { Button } from "@/components/ui/button"
import { getStatusToneTextClass } from "@/lib/status-config"
import { cn } from "@/lib/utils"
import { toastFeedback } from "@/lib/toast-helpers"

const COPY_RESET_DELAY_MS = 2000

export const COPY_BUTTON_BASE_CLASSNAME =
  "flex-shrink-0 text-muted-foreground hover:bg-transparent hover:text-foreground dark:hover:bg-transparent transition-[color,opacity]"

export const COPY_BUTTON_HOVER_REVEAL_CLASSNAME =
  "opacity-100 md:opacity-0 md:group-hover:opacity-100"

export async function copyTextToClipboard(text: string): Promise<boolean> {
  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text)
    } else {
      const textArea = document.createElement("textarea")
      textArea.value = text
      textArea.style.position = "fixed"
      textArea.style.left = "-9999px"
      textArea.style.top = "-9999px"
      document.body.appendChild(textArea)
      textArea.focus()
      textArea.select()
      document.execCommand("copy")
      textArea.remove()
    }

    return true
  } catch {
    return false
  }
}

type CopyButtonProps = Omit<
  React.ComponentPropsWithoutRef<typeof Button>,
  "aria-label" | "children" | "onClick" | "size" | "variant"
> & {
  value: string
  copyLabel: string
  copiedLabel: string
  copyFailedLabel?: string
  toastId?: string
  hideUntilHover?: boolean
  stopPropagation?: boolean
  size?: React.ComponentProps<typeof Button>["size"]
}

export function CopyButton({
  value,
  copyLabel,
  copiedLabel,
  copyFailedLabel,
  toastId,
  hideUntilHover = false,
  stopPropagation = true,
  size = "icon-sm",
  className,
  disabled,
  ...props
}: CopyButtonProps) {
  const [copied, setCopied] = React.useState(false)
  const resetTimerRef = React.useRef<number | null>(null)

  const handleCopy = React.useCallback(async (event: React.MouseEvent<HTMLButtonElement>) => {
    if (stopPropagation) {
      event.stopPropagation()
    }

    const success = await copyTextToClipboard(value)
    if (!success) {
      if (copyFailedLabel) {
        if (toastId) {
          toastFeedback.error(copyFailedLabel, { id: toastId })
        } else {
          toastFeedback.error(copyFailedLabel)
        }
      }
      return
    }

    if (resetTimerRef.current) {
      window.clearTimeout(resetTimerRef.current)
    }

    setCopied(true)
    if (toastId) {
      toastFeedback.success(copiedLabel, { id: toastId })
    } else {
      toastFeedback.success(copiedLabel)
    }
    resetTimerRef.current = window.setTimeout(() => {
      setCopied(false)
      resetTimerRef.current = null
    }, COPY_RESET_DELAY_MS)
  }, [copiedLabel, copyFailedLabel, stopPropagation, toastId, value])

  React.useEffect(() => {
    return () => {
      if (resetTimerRef.current) {
        window.clearTimeout(resetTimerRef.current)
      }
    }
  }, [])

  // Base UI render composition injects trigger props at runtime; copy behavior must retain event ownership.
  return (
    <Button
      {...props}
      type="button"
      variant="ghost"
      size={size}
      className={cn(
        COPY_BUTTON_BASE_CLASSNAME,
        hideUntilHover && COPY_BUTTON_HOVER_REVEAL_CLASSNAME,
        copied && "md:opacity-100",
        className
      )}
      disabled={disabled}
      onClick={handleCopy}
      aria-label={copied ? copiedLabel : copyLabel}
    >
      {copied ? (
        <Check className={cn("h-3.5 w-3.5", getStatusToneTextClass("success"))} />
      ) : (
        <Copy className="h-3.5 w-3.5" />
      )}
    </Button>
  )
}
