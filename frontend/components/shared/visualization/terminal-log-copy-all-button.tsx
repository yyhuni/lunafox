"use client"

import { CopyButton } from "@/components/shared/feedback/copy-button"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"

interface TerminalLogCopyAllButtonProps {
  value: string
  copyLabel: string
  copiedLabel: string
  toastId: string
}

export function TerminalLogCopyAllButton({
  value,
  copyLabel,
  copiedLabel,
  toastId,
}: TerminalLogCopyAllButtonProps) {
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger
          render={(
            <CopyButton
              value={value}
              copyLabel={copyLabel}
              copiedLabel={copiedLabel}
              toastId={toastId}
              disabled={!value}
              size="icon-sm"
              className="border-transparent text-[var(--terminal-log-foreground)] hover:text-[var(--terminal-log-foreground)] focus-visible:border-transparent focus-visible:text-[var(--terminal-log-info)] focus-visible:ring-0 data-[popup-open]:bg-transparent dark:data-[popup-open]:bg-transparent [&_svg]:size-4"
            />
          )}
        />
        <TooltipContent side="left" sideOffset={6}>
          {copyLabel}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}
