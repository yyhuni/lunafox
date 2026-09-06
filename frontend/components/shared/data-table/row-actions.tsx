"use client"

import type * as React from "react"
import { Button } from "@/components/ui/button"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import {
  CopyButton,
  COPY_BUTTON_HOVER_REVEAL_CLASSNAME,
} from "@/components/shared/feedback/copy-button"
import { cn } from "@/lib/utils"

export const DENSE_ROW_ACTION_HOVER_REVEAL_CLASSNAME =
  COPY_BUTTON_HOVER_REVEAL_CLASSNAME

export function DenseRowActionOwner({
  children,
  className,
}: React.PropsWithChildren<{ className?: string }>) {
  return (
    <TooltipProvider delay={300}>
      <div
        data-row-click-exempt="true"
        className={cn("flex items-center justify-end gap-1", className)}
      >
        {children}
      </div>
    </TooltipProvider>
  )
}

interface DenseRowActionButtonProps {
  icon: React.ReactNode
  label: string
  onClick: () => void
}

export function DenseRowActionButton({
  icon,
  label,
  onClick,
}: DenseRowActionButtonProps) {
  return (
    <Tooltip>
      <TooltipTrigger
        render={(
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            // Keep tooltip-open colors aligned with hover to avoid a second transition.
            className="data-[popup-open]:bg-primary/10 data-[popup-open]:text-primary dark:data-[popup-open]:bg-primary/20"
            aria-label={label}
            onClick={onClick}
          />
        )}
      >
        {icon}
      </TooltipTrigger>
      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  )
}

interface QuietCopyButtonProps {
  value: string
  copyLabel: string
  copiedLabel: string
  toastId: string
  className?: string
}

export function QuietCopyButton({
  value,
  copyLabel,
  copiedLabel,
  toastId,
  className,
}: QuietCopyButtonProps) {
  return (
    <CopyButton
      value={value}
      copyLabel={copyLabel}
      copiedLabel={copiedLabel}
      toastId={toastId}
      hideUntilHover
      className={className}
    />
  )
}
