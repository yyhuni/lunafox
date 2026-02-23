"use client"

import * as React from "react"
import { toast, type ToasterProps } from "sonner"

import type { NudgeToastCardProps } from "@/components/nudges/nudge-toast-card"
import {
  isLocalStorageAvailable,
  isNudgeSuppressed,
  suppressNudge,
  withNudgeDismiss,
} from "@/lib/nudge-toast-helpers"

export type NudgeToastVariant = Omit<NudgeToastCardProps, "onDismiss">

export interface UseNudgeToastOptions {
  /**
   * Persistence suppression key.
   * - No transmission: no persistence suppression (every trigger may be bounced)
   * - Passed in: will be written to localStorage after the user closes/clicks the button
   */
  storageKey?: string

  /**
   * Cooldown time (ms).
   * - Not transmitted: write "true" and never play again (until reset)
   * - Pass in: write nextAllowedAt timestamp, which can be bounced again after expiration
   */
  cooldownMs?: number

  /**
   * Trigger probability (0-1), for test or grayscale
   * @default 1
   */
  probability?: number

  /**
   * Delay trigger time (ms)
   * @default 1500
   */
  delay?: number

  /**
   * Toast duration (ms)
   * @default 8000
   */
  duration?: number

  /**
   * Toast location
   * @default "bottom-right"
   */
  position?: ToasterProps["position"]

  /**
   * Optional different copy/style variations
   */
  variants: NudgeToastVariant[]
}

const withDismiss = withNudgeDismiss

/**
 * 全局唯一 nudge toast id，确保右下角只显示一个 nudge，不叠加。
 * 所有 nudge 来源（guardian / star / care / milestone）共享此 id。
 */
const NUDGE_TOAST_ID = "nudge-singleton"

let nudgeToastCardLoader: Promise<
  (typeof import("@/components/nudges/nudge-toast-card"))["NudgeToastCard"]
> | null = null

function loadNudgeToastCard() {
  if (!nudgeToastCardLoader) {
    nudgeToastCardLoader = import("@/components/nudges/nudge-toast-card").then(
      (mod) => mod.NudgeToastCard
    )
  }
  return nudgeToastCardLoader
}

export function useNudgeToast({
  storageKey,
  cooldownMs,
  probability = 1,
  delay = 1500,
  duration = 8000,
  position = "bottom-right",
  variants,
}: UseNudgeToastOptions) {
  const timerRef = React.useRef<number | null>(null)

  React.useEffect(() => {
    return () => {
      if (timerRef.current !== null) {
        window.clearTimeout(timerRef.current)
        timerRef.current = null
      }
    }
  }, [])

  const showVariant = React.useCallback((variant: NudgeToastVariant) => {
    void loadNudgeToastCard().then((NudgeToastCard) => {
      // 使用固定 id，sonner 会自动替换已有的同 id toast，不会叠加
      toast.custom(
        () => {
          const onDismiss = () => {
            toast.dismiss(NUDGE_TOAST_ID)
            if (storageKey) suppressNudge(storageKey, cooldownMs)
          }

          const primaryAction = withDismiss(variant.primaryAction, onDismiss) ?? {
            label: "OK",
            onClick: onDismiss,
          }

          return (
            <NudgeToastCard
              {...variant}
              onDismiss={onDismiss}
              primaryAction={primaryAction}
              secondaryAction={withDismiss(variant.secondaryAction, onDismiss)}
            />
          )
        },
        {
          id: NUDGE_TOAST_ID,
          duration,
          position,
        }
      )
    })
  }, [cooldownMs, duration, position, storageKey])

  const triggerInternal = React.useCallback((variantOverride?: NudgeToastVariant) => {
    if (typeof window === "undefined") return
    if (!variantOverride && (!variants || variants.length === 0)) return

    // Probability control
    if (Math.random() > probability) return

    // Storage suppression
    if (storageKey && isNudgeSuppressed(storageKey, cooldownMs)) return

    const variant = variantOverride ?? variants[Math.floor(Math.random() * variants.length)]
    if (!variant) return

    // Deduplicate pending triggers
    if (timerRef.current !== null) {
      window.clearTimeout(timerRef.current)
      timerRef.current = null
    }

    timerRef.current = window.setTimeout(() => {
      showVariant(variant)
    }, delay)
  }, [cooldownMs, delay, probability, showVariant, storageKey, variants])

  const trigger = React.useCallback(() => {
    triggerInternal()
  }, [triggerInternal])

  const triggerWithVariant = React.useCallback((variant: NudgeToastVariant) => {
    triggerInternal(variant)
  }, [triggerInternal])

  const isSuppressedNow = React.useCallback(() => {
    if (!isLocalStorageAvailable()) return true
    if (!storageKey) return false

    return isNudgeSuppressed(storageKey, cooldownMs)
  }, [cooldownMs, storageKey])

  const reset = React.useCallback(() => {
    if (!isLocalStorageAvailable()) return
    if (!storageKey) return

    try {
      localStorage.removeItem(storageKey)
    } catch {
      // ignore
    }
  }, [storageKey])

  return { trigger, triggerWithVariant, isSuppressedNow, reset }
}
