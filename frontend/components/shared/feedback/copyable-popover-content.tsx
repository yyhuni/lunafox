"use client"

import { useTranslations } from "next-intl"
import { CopyButton } from "@/components/shared/feedback/copy-button"

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
  const tActions = useTranslations("common.actions")
  const tToast = useTranslations("toast")
  const tTooltips = useTranslations("tooltips")
  
  return (
    <div className="group relative">
      <CopyButton
        value={value}
        copyLabel={tActions("copy")}
        copiedLabel={tTooltips("copied")}
        copyFailedLabel={tToast("copyFailed")}
        className="-right-1 -top-1 absolute opacity-60 hover:opacity-100"
      />
      <div className={`text-sm break-all pr-6 max-h-48 overflow-y-auto ${className}`}>
        {value}
      </div>
    </div>
  )
}
