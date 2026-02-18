"use client"

import * as React from "react"
import { useTranslations } from "next-intl"
import { IconX, ArrowRight as IconArrowRight } from "@/components/icons"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import type { NudgeToastCardProps } from "./nudge-toast-card"

export function NudgeGlass({
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
        "relative flex w-full max-w-sm flex-col overflow-hidden rounded-xl border border-white/10 bg-black/60 p-4 shadow-2xl backdrop-blur-xl",
        "before:absolute before:inset-0 before:-z-10 before:bg-gradient-to-tr before:from-indigo-500/10 before:via-purple-500/5 before:to-transparent",
        "dark:bg-zinc-950/50 dark:border-white/5",
        className
      )}
    >
      {/* Glow effect on top border */}
      <div className="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-white/20 to-transparent" />

      <div className="flex gap-4">
        <div className="relative flex size-10 shrink-0 items-center justify-center rounded-lg border border-white/10 bg-white/5 text-xl shadow-inner">
          <div className="absolute inset-0 rounded-lg bg-indigo-500/20 blur opacity-50" />
          <span className="relative z-10">{icon}</span>
        </div>

        <div className="flex-1 min-w-0">
          <h3 className="text-sm font-semibold text-white/90">{title}</h3>
          <p className="mt-1 text-xs leading-relaxed text-white/60 line-clamp-2">
            {description}
          </p>
        </div>

        <button type="button"
          onClick={onDismiss}
          className="shrink-0 -mt-1 -mr-1 rounded-md p-1.5 text-white/40 hover:bg-white/10 hover:text-white transition-colors"
          aria-label={tActions("close")}
        >
          <IconX className="size-4" />
        </button>
      </div>

      <div className="mt-4 flex items-center justify-end gap-2">
        {secondaryAction && (
          <Button
            variant="ghost"
            size="sm"
            className="h-7 px-3 text-xs text-white/60 hover:bg-white/5 hover:text-white"
            onClick={secondaryAction.onClick}
          >
            {secondaryAction.label}
          </Button>
        )}
        <Button
          size="sm"
          className="h-7 rounded-md bg-white/10 px-3 text-xs text-white shadow-none hover:bg-white/20 border border-white/5 backdrop-blur-sm"
          onClick={primaryAction.onClick}
        >
          {primaryAction.label}
          <IconArrowRight className="ml-1.5 size-3 opacity-70" />
        </Button>
      </div>
    </div>
  )
}
