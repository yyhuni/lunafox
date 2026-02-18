"use client"

import * as React from "react"
import { getCareNudgeVariants } from "@/hooks/nudge-rules"
import { useNudgeToast, type NudgeToastVariant } from "@/hooks/use-nudge-toast"

// Persistent key to track last trigger time (once per day)
const STORAGE_KEY = "lunafox:care-nudge:last-seen"

interface UseCareNudgeOptions {
  /**
   * Trigger probability (0-1)
   * @default 0.3 (30% probability)
   */
  probability?: number
  /**
   * Delay before trigger (ms)
   * @default 3000
   */
  delay?: number
}

interface AiNudgeResponse {
  title?: unknown
  description?: unknown
  icon?: unknown
  primaryAction?: {
    label?: unknown
  } | null
  secondaryAction?: {
    label?: unknown
  } | null
}

function toAiVariant(data: AiNudgeResponse): NudgeToastVariant | null {
  if (typeof data.title !== "string" || typeof data.description !== "string") {
    return null
  }

  return {
    title: data.title,
    description: data.description,
    icon: <span className="text-2xl">{typeof data.icon === "string" ? data.icon : "🤖"}</span>,
    primaryAction: {
      label:
        typeof data.primaryAction?.label === "string" && data.primaryAction.label.trim() !== ""
          ? data.primaryAction.label
          : "OK",
    },
    secondaryAction:
      typeof data.secondaryAction?.label === "string" && data.secondaryAction.label.trim() !== ""
        ? {
            label: data.secondaryAction.label,
            buttonVariant: "outline",
          }
        : undefined,
  }
}

function pickRandomVariant(variants: NudgeToastVariant[]): NudgeToastVariant | undefined {
  if (variants.length === 0) return undefined
  return variants[Math.floor(Math.random() * variants.length)]
}

/**
 * Smart care nudge hook
 * Provides hacker-style care messages based on time, date, and random events
 */
export function useCareNudge(options: UseCareNudgeOptions = {}) {
  const { probability = 0.3, delay = 3000 } = options

  // Cooldown: ~16 hours, ensures roughly once per day if user visits daily
  const COOLDOWN_MS = 16 * 60 * 60 * 1000

  const aiApiUrl =
    process.env.NEXT_PUBLIC_CARE_NUDGE_AI_URL?.trim() ||
    "https://lunafox-ai-proxy-fzqn2tz4eb4f.yyhunisec.deno.net/"

  const { triggerWithVariant, isSuppressedNow, reset } = useNudgeToast({
    storageKey: STORAGE_KEY,
    cooldownMs: COOLDOWN_MS,
    probability,
    delay,
    duration: 8000,
    position: "bottom-right",
    variants: [],
  })

  const triggerLocalFallback = React.useCallback(() => {
    const variant = pickRandomVariant(getCareNudgeVariants(new Date()))
    if (!variant) return
    triggerWithVariant(variant)
  }, [triggerWithVariant])

  // Wrap trigger: try AI generation first, fallback to local static variants
  const triggerWithAi = React.useCallback(async () => {
    if (isSuppressedNow()) return

    if (!aiApiUrl) {
      triggerLocalFallback()
      return
    }

    try {
      const now = new Date()
      const context = {
        hour: now.getHours(),
        day: now.getDay(),
        event: "daily_care",
      }

      const res = await fetch(aiApiUrl, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ context }),
      })

      if (!res.ok) throw new Error("AI API failed")

      const aiData = (await res.json()) as AiNudgeResponse
      const aiVariant = toAiVariant(aiData)
      if (!aiVariant) throw new Error("Invalid AI nudge payload")

      triggerWithVariant(aiVariant)
    } catch (err) {
      console.warn("AI Nudge failed, falling back to static:", err)
      triggerLocalFallback()
    }
  }, [aiApiUrl, isSuppressedNow, triggerLocalFallback, triggerWithVariant])

  // Auto-trigger: attempt to trigger after component mount
  React.useEffect(() => {
    triggerWithAi()
  }, [triggerWithAi])

  return { trigger: triggerWithAi, reset }
}
