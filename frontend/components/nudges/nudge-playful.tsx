"use client"

import * as React from "react"
import { useTranslations } from "next-intl"
import { IconX, IconActivity } from "@/components/icons"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import type { NudgeToastCardProps } from "./nudge-toast-card"

export function NudgeReport({
  title,
  description,
  icon,
  primaryAction,
  secondaryAction,
  onDismiss,
  className,
}: NudgeToastCardProps) {
  const tActions = useTranslations("common.actions")

  return (
    <div
      className={cn(
        "relative flex w-full max-w-sm flex-col overflow-hidden rounded-lg border bg-card p-0 shadow-lg",
        "dark:border-border",
        className
      )}
    >
      <div className="absolute left-0 top-0 bottom-0 w-1 bg-primary" />
      
      {/* Background Pattern */}
      <IconActivity className="absolute -right-4 -top-4 size-24 text-muted/5 rotate-12" />

      <div className="relative p-4 pl-5">
        <div className="flex items-start justify-between gap-4">
          <div className="flex items-center gap-3">
            <div className="flex size-9 items-center justify-center rounded-md bg-primary/10 text-primary">
              {icon}
            </div>
            <div>
              <h3 className="font-semibold text-sm leading-none tracking-tight">{title}</h3>
              <p className="mt-1 text-xs text-muted-foreground">System Notification</p>
            </div>
          </div>
          <button type="button"
            onClick={onDismiss}
            className="text-muted-foreground hover:text-foreground"
            aria-label={tActions("close")}
          >
            <IconX className="size-4" />
          </button>
        </div>

        <div className="mt-3">
          <p className="text-sm leading-relaxed text-muted-foreground">
            {description}
          </p>
        </div>

        <div className="mt-4 flex items-center justify-end gap-2 border-t pt-3">
          {secondaryAction && (
            <Button
              variant="ghost"
              size="sm"
              className="h-8 text-xs font-normal"
              onClick={secondaryAction.onClick}
            >
              {secondaryAction.label}
            </Button>
          )}
          <Button
            size="sm"
            className="h-8 text-xs"
            onClick={primaryAction.onClick}
          >
            {primaryAction.label}
          </Button>
        </div>
      </div>
    </div>
  )
}
