"use client"

import * as React from "react"
import { useTranslations } from "next-intl"
import { IconX } from "@/components/icons"
import { cn } from "@/lib/utils"
import type { NudgeToastCardProps } from "./nudge-toast-card"

export function NudgeTerminal({
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
        "relative flex w-full max-w-sm flex-col overflow-hidden rounded-md border border-zinc-800 bg-zinc-950 p-0 shadow-xl font-mono text-sm",
        "dark:border-zinc-700",
        className
      )}
    >
      {/* Terminal Header */}
      <div className="flex items-center justify-between border-b border-zinc-800 bg-zinc-900 px-3 py-1.5">
        <div className="flex items-center gap-1.5">
          <div className="size-2.5 rounded-full bg-red-500/80" />
          <div className="size-2.5 rounded-full bg-yellow-500/80" />
          <div className="size-2.5 rounded-full bg-green-500/80" />
        </div>
        <div className="text-[10px] text-zinc-500">zsh — 80x24</div>
      </div>

      <div className="p-4">
        <div className="mb-2 flex items-start gap-2 text-green-500">
          {icon ? <span className="shrink-0 [&>svg]:size-4">{icon}</span> : null}
          <span className="shrink-0 text-lg leading-none mt-0.5">➜</span>
          <div className="space-y-1">
            <h3 className="font-bold leading-tight text-green-400">{title}</h3>
          </div>
        </div>
        
        <div className="pl-6 text-xs leading-relaxed text-zinc-300 opacity-90">
          <span className="mr-2 text-zinc-500">[info]</span>
          {description}
          <span className="ml-1 inline-block h-3 w-1.5 animate-pulse bg-green-500 align-middle" />
        </div>

        <div className="mt-4 flex justify-end gap-2 pl-6">
          {secondaryAction && (
            <button type="button"
              onClick={secondaryAction.onClick}
              className="px-2 py-1 text-xs text-zinc-500 hover:text-zinc-300 hover:underline decoration-zinc-500 underline-offset-4 transition-colors"
            >
              ./{secondaryAction.label}
            </button>
          )}
          <button type="button"
            onClick={primaryAction.onClick}
            className="border border-green-500/30 bg-green-500/10 px-3 py-1 text-xs text-green-400 hover:bg-green-500/20 transition-colors"
          >
            $ {primaryAction.label}
          </button>
        </div>
      </div>

      <button type="button"
        onClick={onDismiss}
        className="absolute right-2 top-1.5 p-0.5 text-zinc-600 hover:text-zinc-400 opacity-0 hover:opacity-100"
        aria-label={tActions("close")}
      >
        <IconX className="size-3" />
      </button>
    </div>
  )
}
