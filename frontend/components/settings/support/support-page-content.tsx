"use client"

import * as React from "react"
import { AnimatePresence, motion, useReducedMotion, type Transition } from "framer-motion"
import { useTranslations } from "next-intl"
import Image from "next/image"

import { IconBrandGithub, IconHeart, IconMessageReport, IconRocket, IconScan, IconShieldCheck, IconUsers } from "@/components/icons"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

import { SupportFlyingBirds, SupportGrowingBranch } from "./support-flying-birds"
import {
  SupportPageActions,
  SupportPageContentStack,
  SupportPageFooter,
  SupportPageHeader,
  SupportPageLayout,
  SupportPageValueBand,
  SupportPageValueBandItem,
  SupportPageValueBandList,
  SUPPORT_PAGE_TIER_OPTIONS_LOADING_SLOT,
  SUPPORT_TIER_CARD_CLASS,
} from "./support-page-layout"

const SUPPORT_SURFACE_TRANSITION_CLASS =
  "transition-[background-color,border-color,box-shadow,color,opacity] duration-300"
const SUPPORT_ACTION_TRANSITION_CLASS =
  "transition-[background-color,border-color,color,box-shadow] duration-200"

const TIERS = [
  { id: "starter", amount: "¥10" },
  { id: "steady", amount: "¥30" },
  { id: "growth", amount: "¥68" },
  { id: "guardian", amount: "¥128" },
  { id: "custom", amount: null },
] as const

type SupportTierId = (typeof TIERS)[number]["id"]

const SUPPORT_METHODS = ["wechat", "alipay", "contact"] as const

type SupportMethod = (typeof SUPPORT_METHODS)[number]

const SUPPORT_TREE_REVEAL = {
  default: { flowerScale: 0.5, progress: 0.12 },
  starter: { flowerScale: 0.8, progress: 0.34 },
  steady: { flowerScale: 0.95, progress: 0.55 },
  growth: { flowerScale: 1.12, progress: 0.76 },
  guardian: { flowerScale: 1.3, progress: 1 },
  custom: { flowerScale: 1.4, progress: 1 },
} as const

export interface SupportPageContentProps {
  onReady?: () => void
  deferInitialSkeleton?: boolean
}

export default function SupportPageContent({
  onReady,
  deferInitialSkeleton = false,
}: SupportPageContentProps = {}) {
  const t = useTranslations("settings.support")
  const prefersReducedMotion = useReducedMotion()
  const [showTiers, setShowTiers] = React.useState(false)
  const [activeMethod, setActiveMethod] = React.useState<SupportMethod>("wechat")
  const [selectedTier, setSelectedTier] = React.useState<SupportTierId | null>(null)
  const [hoveredTier, setHoveredTier] = React.useState<SupportTierId | null>(null)
  const [hasStableInitialFrame, setHasStableInitialFrame] = React.useState(false)
  const selectedTierData = TIERS.find((tier) => tier.id === selectedTier)
  const getTierAmountLabel = (tier: (typeof TIERS)[number]) => tier.amount ?? t("tiers.customAmount")
  const treeReveal = SUPPORT_TREE_REVEAL[hoveredTier ?? selectedTier ?? "default"]
  const transitionProps: Transition = prefersReducedMotion
    ? { duration: 0.25 }
    : { type: "spring", stiffness: 220, damping: 28 }

  React.useEffect(() => {
    let secondFrame: number | null = null
    const firstFrame = window.requestAnimationFrame(() => {
      secondFrame = window.requestAnimationFrame(() => {
        setHasStableInitialFrame(true)
        onReady?.()
      })
    })

    return () => {
      window.cancelAnimationFrame(firstFrame)
      if (secondFrame !== null) window.cancelAnimationFrame(secondFrame)
    }
  }, [onReady])

  if (deferInitialSkeleton && !hasStableInitialFrame) return null

  return (
    <SupportPageLayout
      decoration={
        <>
          <SupportFlyingBirds reducedMotion={prefersReducedMotion ?? false} />
          <SupportGrowingBranch
            flowerScale={treeReveal.flowerScale}
            reducedMotion={prefersReducedMotion ?? false}
            reveal={treeReveal.progress}
            trigger={showTiers ? 1 : 0}
          />
        </>
      }
      footer={
        <SupportPageFooter>
          <p>{t("footer.note")}</p>
          <div className="flex flex-wrap items-center justify-center gap-x-3 gap-y-1">
            <span>{t("footer.nonMonetary")}</span>
            <a href="https://github.com/yyhuni/xingrin" target="_blank" rel="noopener noreferrer" className="underline underline-offset-2 hover:text-foreground">
              {t("footer.links.github")}
            </a>
            <a href="https://github.com/yyhuni/xingrin/issues" target="_blank" rel="noopener noreferrer" className="underline underline-offset-2 hover:text-foreground">
              {t("footer.links.issues")}
            </a>
            <a href="https://github.com/yyhuni/xingrin/releases" target="_blank" rel="noopener noreferrer" className="underline underline-offset-2 hover:text-foreground">
              {t("footer.links.releases")}
            </a>
          </div>
        </SupportPageFooter>
      }
    >
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={transitionProps}
      >
        <SupportPageContentStack data-testid="support-value-first">
          <SupportPageHeader>
            <div className={cn("inline-flex items-center gap-2 text-success", textRole.bodyStrong)}>
              <IconUsers className="size-4" />
              {t("unlocked.badge")}
            </div>
            <h1 className={cn(textRole.pageTitleDisplay, "text-balance md:text-4xl md:leading-tight")}>
              {t("unlocked.title")}
            </h1>
            <p className={cn(textRole.bodyLarge, "max-w-[64ch] text-muted-foreground")}>
              {t("unlocked.description")}
            </p>
          </SupportPageHeader>

          <SupportPageValueBand aria-labelledby="support-impact-title">
            <h2 id="support-impact-title" className="sr-only">
              {t("unlocked.title")}
            </h2>
            <SupportPageValueBandList>
              {(["maintenance", "growth", "community"] as const).map((item, index) => (
                <SupportPageValueBandItem
                  key={item}
                  index={index}
                >
                  <div className="flex size-9 shrink-0 items-center justify-center rounded-md bg-primary/10 text-primary">
                    {item === "maintenance" ? <IconShieldCheck className="size-4" /> : item === "growth" ? <IconRocket className="size-4" /> : <IconUsers className="size-4" />}
                  </div>
                  <div className="space-y-1">
                    <h3 className={textRole.sectionTitle}>{t(`unlocked.impact.${item}.title`)}</h3>
                    <p className={textRole.bodySubtle}>{t(`unlocked.impact.${item}.description`)}</p>
                  </div>
                </SupportPageValueBandItem>
              ))}
            </SupportPageValueBandList>
          </SupportPageValueBand>

          <SupportPageActions>
            <Button
              aria-expanded={showTiers}
              aria-controls="support-tier-options"
              className="gap-2"
              onClick={() => setShowTiers(true)}
            >
              <IconUsers className="size-4" />
              {t("unlocked.primaryCta")}
            </Button>
            <Button
              variant="outline"
              className="gap-2"
              render={<a href="https://github.com/yyhuni/xingrin" target="_blank" rel="noopener noreferrer" />}
            >
              <IconBrandGithub className="size-4" />
              {t("hero.secondaryCta")}
            </Button>
          </SupportPageActions>

          <motion.section
            {...SUPPORT_PAGE_TIER_OPTIONS_LOADING_SLOT}
            id="support-tier-options"
            aria-hidden={!showTiers}
            aria-labelledby="support-tier-title"
            animate={{ opacity: showTiers ? 1 : 0 }}
            transition={{ duration: prefersReducedMotion ? 0 : 0.2 }}
            className={cn("scroll-mt-6", !showTiers && "invisible pointer-events-none")}
            data-testid="support-tier-options"
          >
            <div className="mb-6 max-w-2xl">
              <h2 id="support-tier-title" className={textRole.panelTitle}>
                {t("tiers.title")}
              </h2>
            </div>
            <div className="grid w-full grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-5">
              {TIERS.map((tier, index) => (
                <motion.button
                  key={tier.id}
                  type="button"
                  data-testid={`support-tier-${tier.id}`}
                  aria-pressed={selectedTier === tier.id}
                  initial={{ opacity: 0, y: prefersReducedMotion ? 0 : 12 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{
                    delay: prefersReducedMotion ? 0 : index * 0.08,
                    duration: prefersReducedMotion ? 0.2 : 0.55,
                  }}
                  className={cn(
                    SUPPORT_TIER_CARD_CLASS,
                    SUPPORT_SURFACE_TRANSITION_CLASS,
                    selectedTier === tier.id
                      ? "border-primary bg-primary/5 ring-1 ring-primary"
                      : "border-border bg-card hover:border-primary/50 hover:bg-muted/30",
                  )}
                  onBlur={() => setHoveredTier(null)}
                  onClick={() => setSelectedTier(tier.id)}
                  onFocus={() => setHoveredTier(tier.id)}
                  onMouseEnter={() => setHoveredTier(tier.id)}
                  onMouseLeave={() => setHoveredTier(null)}
                >
                  <div className="space-y-2">
                    <span className={cn(textRole.metricValueDisplay, "text-primary")}>
                      {getTierAmountLabel(tier)}
                    </span>
                    <h3 className={textRole.sectionTitle}>{t(`tiers.items.${tier.id}.title`)}</h3>
                    <p className={textRole.helperText}>{t(`tiers.items.${tier.id}.description`)}</p>
                  </div>
                  <span className={cn(textRole.badge, "mt-5 inline-flex items-center gap-1.5 text-primary")}>
                    <IconHeart className="size-3.5" />
                    {t("tiers.action")}
                  </span>
                </motion.button>
              ))}
            </div>
          </motion.section>

          <Dialog open={!!selectedTier} onOpenChange={(open) => !open && setSelectedTier(null)}>
            <DialogContent className="overflow-hidden rounded-[calc(var(--radius)*4)] border-0 bg-background/95 p-0 shadow-2xl backdrop-blur-xl sm:max-w-[420px]">
              <DialogHeader className="px-6 pt-8 text-center">
                <DialogTitle>{t("dialog.title")}</DialogTitle>
              </DialogHeader>
              <div className="relative flex flex-col items-center space-y-8 px-8 pb-8 pt-6">
                <div className="relative z-10 flex w-full gap-2 rounded-full border border-border/80 bg-muted/30 p-1.5 shadow-inner">
                  {SUPPORT_METHODS.map((method) => (
                    <button
                      key={method}
                      type="button"
                      data-testid={`payment-method-${method}`}
                      aria-pressed={activeMethod === method}
                      onClick={() => setActiveMethod(method)}
                      className={cn(
                        "flex h-10 flex-1 items-center justify-center gap-2 rounded-full py-2.5 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-background",
                        textRole.bodyStrong,
                        SUPPORT_ACTION_TRANSITION_CLASS,
                        activeMethod === method ? "bg-primary text-primary-foreground shadow" : "text-muted-foreground hover:text-foreground",
                      )}
                    >
                      {method === "wechat" ? <IconHeart className="size-4" /> : method === "alipay" ? <IconScan className="size-4" /> : <IconMessageReport className="size-4" />}
                      {t(`unlocked.${method === "wechat" ? "wechatPay" : method === "alipay" ? "alipay" : "contactAuthor"}`)}
                    </button>
                  ))}
                </div>

                <div className="relative mx-auto aspect-square w-full max-w-[240px]">
                  <AnimatePresence mode="wait">
                    <motion.div
                      key={activeMethod}
                      initial={prefersReducedMotion ? { opacity: 0 } : { opacity: 0, scale: 0.96 }}
                      animate={{ opacity: 1, scale: 1 }}
                      exit={{ opacity: 0 }}
                      transition={{ duration: 0.2 }}
                      className="absolute inset-0 flex items-center justify-center rounded-[calc(var(--radius)*3)] border border-border/20 bg-[var(--logo-background)] p-3 shadow-xl"
                    >
                      <Image
                        src={`/images/support/${activeMethod}-qr.jpg`}
                        alt={t(`cards.${activeMethod}.alt`)}
                        width={240}
                        height={240}
                        className="h-full w-full rounded-[calc(var(--radius)*2)] object-cover"
                      />
                    </motion.div>
                  </AnimatePresence>
                </div>

                <div className="z-10 w-full space-y-1 px-2 text-center">
                  <p data-testid="support-dialog-title" className={textRole.bodyStrong}>
                    {t(`cards.${activeMethod}.title`)}{activeMethod !== "contact" && selectedTierData ? ` · ${getTierAmountLabel(selectedTierData)}` : ""}
                  </p>
                  <p data-testid="support-dialog-amount" className="text-[13px] text-muted-foreground">
                    {activeMethod === "contact"
                      ? t("cards.contact.description")
                      : selectedTierData && t(`cards.${activeMethod}.description`, { amount: getTierAmountLabel(selectedTierData) })}
                  </p>
                </div>

              </div>
            </DialogContent>
          </Dialog>

        </SupportPageContentStack>
      </motion.div>
    </SupportPageLayout>
  )
}
