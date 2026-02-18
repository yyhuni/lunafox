"use client"

import * as React from "react"
import {
  IconActivity,
  IconHeart,
  IconMoon,
  IconTrophy,
} from "@/components/icons"
import { useNudgeToast, type NudgeToastVariant } from "@/hooks/use-nudge-toast"

const STORAGE_FIRST_SEEN_KEY = "lunafox:milestone:first-seen"
const STORAGE_MILESTONE_PREFIX = "lunafox:milestone:triggered:"

// Define milestones (number of days -> variations)
const MILESTONES: Record<number, NudgeToastVariant> = {
  1: {
    icon: <IconHeart className="size-6 text-pink-500" />,
    title: "Hello World! 👋",
    description: "这是我们在月狐控制台共度的第一天。很高兴认识你，指挥官。",
    primaryAction: { label: "你好呀" },
  },
  7: {
    icon: <IconTrophy className="size-6 text-yellow-500" />,
    title: "第一周达成！🏅",
    description: "已经过去一周了。你的资产库是不是也跟着胖了一圈？保持这个节奏！",
    primaryAction: { label: "继续冲" },
  },
  30: {
    icon: <IconMoon className="size-6 text-indigo-500" />,
    title: "满月纪念 🌕",
    description: "30 天的陪伴。这一个月里，感谢你为了互联网安全所做的每一次扫描。",
    primaryAction: { label: "干杯" },
  },
  100: {
    icon: <IconActivity className="size-6 text-emerald-500" />,
    title: "百日修仙达成 💯",
    description: "100 天的坚持。今天的你，一定比 100 天前更强了。",
    primaryAction: { label: "确实" },
    secondaryAction: { label: "还得练" },
  },
}

/**
 * Pure front-end milestone Hook
 * Record the user's first visit time and trigger a one-time commemorative pop-up window on a specific number of days (1, 7, 30, 100)
 */
export function useMilestoneNudge() {
  const [targetVariant, setTargetVariant] = React.useState<NudgeToastVariant[]>([])
  const [storageKey, setStorageKey] = React.useState<string | undefined>(undefined)

  // Initialization check
  React.useEffect(() => {
    if (typeof window === "undefined") return

    const now = Date.now()
    const firstSeenStr = localStorage.getItem(STORAGE_FIRST_SEEN_KEY)

    // 1. If this is your first time, record the time
    if (!firstSeenStr) {
      localStorage.setItem(STORAGE_FIRST_SEEN_KEY, now.toString())
      // Day 1 milestones can be triggered immediately (or wait for the next refresh)
      // Here choose to trigger Day 1 immediately
      const variant = MILESTONES[1]
      const key = `${STORAGE_MILESTONE_PREFIX}1`
      if (!localStorage.getItem(key)) {
        setTargetVariant([variant])
        setStorageKey(key)
      }
      return
    }

    // 2. Calculate the number of days used
    const firstSeen = parseInt(firstSeenStr, 10)
    const daysPassed = Math.floor((now - firstSeen) / (1000 * 60 * 60 * 24)) + 1

    // 3. Check if a certain milestone is hit
    const milestone = MILESTONES[daysPassed]
    if (milestone) {
      const key = `${STORAGE_MILESTONE_PREFIX}${daysPassed}`
      // Check whether this milestone has been ejected (it will never be ejected, so there is no need for cooldown and it is directly controlled by storageKey)
      if (!localStorage.getItem(key)) {
        setTargetVariant([milestone])
        setStorageKey(key)
      }
    }
  }, [])

  // Trigger using universal Hook
  const { trigger } = useNudgeToast({
    storageKey, // The key here is dynamic (e.g. triggered:30). If it is played once, true will be written.
    delay: 2000, // Play a little later to let the page load first
    probability: 1, // Milestones are hard triggers, not random
    variants: targetVariant,
  })

  // When a milestone to be triggered is detected, execute
  React.useEffect(() => {
    if (targetVariant.length > 0 && storageKey) {
      trigger()
    }
  }, [targetVariant, storageKey, trigger])

  return null // No need to expose methods, fully automatic
}
