"use client"

import * as React from "react"
import { toast } from "sonner"
import confetti from "canvas-confetti"

import { IconBrandGithub } from "@/components/icons"
import { useNudgeToast, type NudgeToastVariant } from "@/hooks/use-nudge-toast"

const STORAGE_KEY = "lunafox:star-nudge-dismissed"
const GITHUB_REPO_URL = "https://github.com/yyhuni/xingrin"

const STAR_VARIANTS = [
  {
    icon: "🍗",
    title: "本狐狸表现得怎么样？",
    desc: "如果我帮你发现了漏洞，能不能赏我一颗星星当作鸡腿？",
    btn: "投喂星星",
    cancel: "下次一定",
    color: "bg-orange-100 text-orange-600 dark:bg-orange-900/30 dark:text-orange-400",
    btnColor:
      "bg-orange-600 hover:bg-orange-700 text-white dark:bg-orange-500 dark:hover:bg-orange-600",
  },
  {
    icon: "👔",
    title: "狐狸也是有 KPI 的...",
    desc: "老板说如果今天能收到星星，晚上就可以不加班了！救救孩子吧！🥺",
    btn: "帮它一把",
    cancel: "继续加班",
    color: "bg-blue-100 text-blue-600 dark:bg-blue-900/30 dark:text-blue-400",
    btnColor:
      "bg-blue-600 hover:bg-blue-700 text-white dark:bg-blue-500 dark:hover:bg-blue-600",
  },
  {
    icon: "✨",
    title: "嘿！我刚刚是不是很棒？",
    desc: "为了帮你扫出这个结果，我的 CPU 都快冒烟了。快夸夸我（点个 Star 也可以）！😤",
    btn: "你最棒了",
    cancel: "也就一般",
    color: "bg-purple-100 text-purple-600 dark:bg-purple-900/30 dark:text-purple-400",
    btnColor:
      "bg-purple-600 hover:bg-purple-700 text-white dark:bg-purple-500 dark:hover:bg-purple-600",
  },
  {
    icon: "📦",
    title: "你可以收留我吗？",
    desc: "我是一只来自 GitHub 的开源小狐狸，只要一颗星星，我就能在这个项目里安家。🏠",
    btn: "带你回家",
    cancel: "流浪去吧",
    color: "bg-amber-100 text-amber-600 dark:bg-amber-900/30 dark:text-amber-400",
    btnColor:
      "bg-amber-600 hover:bg-amber-700 text-white dark:bg-amber-500 dark:hover:bg-amber-600",
  },
  {
    icon: "🍩",
    title: "饿饿... 饭饭...",
    desc: "扫描虽然很快乐，但是肚子也会饿。听说 GitHub 的星星味道像甜甜圈？🤤",
    btn: "请你吃",
    cancel: "减肥吧你",
    color: "bg-pink-100 text-pink-600 dark:bg-pink-900/30 dark:text-pink-400",
    btnColor:
      "bg-pink-600 hover:bg-pink-700 text-white dark:bg-pink-500 dark:hover:bg-pink-600",
  },
]

interface UseStarNudgeOptions {
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
}

/**
 * Smart Star Guide Hook (little fox cute version 🦊)
 * trigger() is called after the user completes a critical operation (such as a scan complete)
 */
export function useStarNudge(options: UseStarNudgeOptions = {}) {
  const { probability = 1, delay = 1500 } = options

  const handleConfetti = React.useCallback(() => {
    // Simple confetti spray
    const count = 200
    const defaults = {
      origin: { y: 0.7 },
    }

    function fire(particleRatio: number, opts: confetti.Options) {
      confetti({
        ...defaults,
        ...opts,
        particleCount: Math.floor(count * particleRatio),
      })
    }

    fire(0.25, {
      spread: 26,
      startVelocity: 55,
    })
    fire(0.2, {
      spread: 60,
    })
    fire(0.35, {
      spread: 100,
      decay: 0.91,
      scalar: 0.8,
    })
    fire(0.1, {
      spread: 120,
      startVelocity: 25,
      decay: 0.92,
      scalar: 1.2,
    })
    fire(0.1, {
      spread: 120,
      startVelocity: 45,
    })
  }, [])

  const variants: NudgeToastVariant[] = React.useMemo(
    () =>
      STAR_VARIANTS.map((v) => ({
        icon: v.icon,
        title: v.title,
        description: v.desc,
        iconHref: GITHUB_REPO_URL,
        iconOnClick: handleConfetti,
        iconClassName: v.color,
        secondaryAction: {
          label: v.cancel,
          buttonVariant: "outline",
          className: "text-muted-foreground",
        },
        primaryAction: {
          label: v.btn,
          icon: <IconBrandGithub className="size-4" />,
          className: v.btnColor,
          onClick: () => {
            handleConfetti()
            // Delay navigation briefly so users can see the confetti first.
            setTimeout(() => {
              window.open(GITHUB_REPO_URL, "_blank")
            }, 300)
          },
        },
      })),
    [handleConfetti]
  )

  const { trigger, reset: resetNudge } = useNudgeToast({
    storageKey: STORAGE_KEY,
    probability,
    delay,
    duration: Infinity,
    position: "bottom-right",
    variants,
  })

  const reset = React.useCallback(() => {
    resetNudge()
    toast.success("已重置引导状态，再次触发将重新显示。")
  }, [resetNudge])

  return { trigger, reset }
}
