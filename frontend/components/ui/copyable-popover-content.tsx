"use client"

import React from "react"
import { Button } from "@/components/ui/button"
import { Copy, Check } from "@/components/icons"
import { toast } from "sonner"
import { useTranslations } from "next-intl"

/**
 * Copyable Popover content component
 * Displays content directly with a copy button in the top right corner
 */
export function CopyablePopoverContent({ 
  value, 
  className = "" 
}: { 
  value: string
  className?: string 
}) {
  const [copied, setCopied] = React.useState(false)
  const tActions = useTranslations("common.actions")
  const tToast = useTranslations("toast")
  const tTooltips = useTranslations("tooltips")
  
  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(value)
      setCopied(true)
      toast.success(tToast("copied"))
      setTimeout(() => setCopied(false), 2000)
    } catch {
      toast.error(tToast("copyFailed"))
    }
  }
  
  return (
    <div className="relative">
      <Button
        variant="ghost"
        size="icon"
        className="absolute -top-1 -right-1 h-6 w-6 opacity-60 hover:opacity-100"
        onClick={handleCopy}
        aria-label={copied ? tTooltips("copied") : tActions("copy")}
      >
        {copied ? (
          <Check className="h-3.5 w-3.5 text-[var(--success)]" />
        ) : (
          <Copy className="h-3.5 w-3.5" />
        )}
      </Button>
      <div className={`text-sm break-all pr-6 max-h-48 overflow-y-auto ${className}`}>
        {value}
      </div>
    </div>
  )
}
