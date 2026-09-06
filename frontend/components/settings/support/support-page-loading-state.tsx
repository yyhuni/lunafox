"use client"

import * as React from "react"
import { useTranslations } from "next-intl"

import { IconBrandGithub, IconUsers } from "@/components/icons"
import { getLoadingOwnerAttributes } from "@/components/shared/loading/loading-owner"
import { ActionSkeleton } from "@/components/shared/loading/action-skeleton"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
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

const SUPPORT_IMPACT_ITEMS = ["maintenance", "growth", "community"] as const

function LoadingText({
  children,
  className,
}: {
  children: React.ReactNode
  className?: string
}) {
  return (
    <span
      aria-hidden="true"
      className={cn(
        "loading-skeleton inline text-transparent [box-decoration-break:clone] [-webkit-box-decoration-break:clone] [&::after]:hidden",
        className,
      )}
    >
      {children}
    </span>
  )
}

function SupportLoadingActionShell({
  children,
  icon,
  variant = "outline",
}: {
  children: React.ReactNode
  icon: React.ReactElement<{ className?: string }>
  variant?: "default" | "outline"
}) {
  return (
    <Button
      aria-hidden="true"
      disabled
      variant={variant}
      className="relative border-transparent bg-transparent text-transparent shadow-none disabled:opacity-100"
    >
      {React.cloneElement(icon, { className: cn(icon.props.className, "invisible") })}
      <span className="invisible">{children}</span>
      <ActionSkeleton className="pointer-events-none absolute inset-0 h-full w-full" />
    </Button>
  )
}

export interface SupportPageLoadingStateProps {
  owner?: string
  className?: string
}

export function SupportPageLoadingState({
  owner,
  className,
}: SupportPageLoadingStateProps) {
  const t = useTranslations("settings.support")

  if (owner !== undefined && !owner.trim()) {
    throw new Error("SupportPageLoadingState requires a non-empty owner.")
  }

  return (
    <SupportPageLayout
      {...(owner ? getLoadingOwnerAttributes({ owner, layer: "route", intent: "route" }) : {})}
      className={className}
      footer={
        <SupportPageFooter>
          <p><LoadingText>{t("footer.note")}</LoadingText></p>
          <div className="flex flex-wrap items-center justify-center gap-x-3 gap-y-1">
            <LoadingText>{t("footer.nonMonetary")}</LoadingText>
            <LoadingText>{t("footer.links.github")}</LoadingText>
            <LoadingText>{t("footer.links.issues")}</LoadingText>
            <LoadingText>{t("footer.links.releases")}</LoadingText>
          </div>
        </SupportPageFooter>
      }
    >
      <div>
        <SupportPageContentStack aria-hidden="true">
          <SupportPageHeader>
            <div className={cn("inline-flex items-center gap-2 text-success", textRole.bodyStrong)}>
              <IconUsers className="invisible size-4" />
              <LoadingText>{t("unlocked.badge")}</LoadingText>
            </div>
            <h1 className={cn(textRole.pageTitleDisplay, "text-balance md:text-4xl md:leading-tight")}>
              <LoadingText>{t("unlocked.title")}</LoadingText>
            </h1>
            <p className={cn(textRole.bodyLarge, "max-w-[64ch] text-muted-foreground")}>
              <LoadingText>{t("unlocked.description")}</LoadingText>
            </p>
          </SupportPageHeader>

          <SupportPageValueBand>
            <SupportPageValueBandList>
              {SUPPORT_IMPACT_ITEMS.map((id, index) => (
                <SupportPageValueBandItem key={id} index={index}>
                  <Skeleton className="size-9 shrink-0 rounded-md" />
                  <div className="space-y-1">
                    <h3 className={textRole.sectionTitle}>
                      <LoadingText>{t(`unlocked.impact.${id}.title`)}</LoadingText>
                    </h3>
                    <p className={textRole.bodySubtle}>
                      <LoadingText>{t(`unlocked.impact.${id}.description`)}</LoadingText>
                    </p>
                  </div>
                </SupportPageValueBandItem>
              ))}
            </SupportPageValueBandList>
          </SupportPageValueBand>

          <SupportPageActions>
            <SupportLoadingActionShell icon={<IconUsers className="size-4" />} variant="default">
              {t("unlocked.primaryCta")}
            </SupportLoadingActionShell>
            <SupportLoadingActionShell icon={<IconBrandGithub className="size-4" />}>
              {t("hero.secondaryCta")}
            </SupportLoadingActionShell>
          </SupportPageActions>

          <section
            {...SUPPORT_PAGE_TIER_OPTIONS_LOADING_SLOT}
            aria-hidden="true"
            className="invisible"
          >
            <div className="mb-6 max-w-2xl">
              <h2 className={textRole.panelTitle}>
                <LoadingText>{t("tiers.title")}</LoadingText>
              </h2>
            </div>
            <div className="grid w-full grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-5">
              {Array.from({ length: 5 }, (_, index) => (
                <div key={index} className={SUPPORT_TIER_CARD_CLASS} />
              ))}
            </div>
          </section>
        </SupportPageContentStack>
      </div>
    </SupportPageLayout>
  )
}
