"use client"

import React from "react"
import { usePathname } from "next/navigation"
import Link from "next/link"
import { HelpCircle } from "@/components/icons"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { PageHeader } from "@/components/common/page-header"
import { useFingerprintStats } from "@/hooks/use-fingerprints"
import { useTranslations } from "next-intl"

/**
 * Fingerprint management layout
 * Provides tab navigation to switch between different fingerprint libraries
 */
export default function FingerprintsLayout({
  children,
}: {
  children: React.ReactNode
}) {
  const pathname = usePathname()
  const { data: stats, isLoading } = useFingerprintStats()
  const t = useTranslations("tools.fingerprints")

  // Get currently active tab
  const getActiveTab = () => {
    if (pathname.includes("/ehole")) return "ehole"
    if (pathname.includes("/goby")) return "goby"
    if (pathname.includes("/wappalyzer")) return "wappalyzer"
    if (pathname.includes("/fingers")) return "fingers"
    if (pathname.includes("/fingerprinthub")) return "fingerprinthub"
    if (pathname.includes("/arl")) return "arl"
    return "ehole"
  }

  // Tab path mapping
  const basePath = "/tools/fingerprints"
  const tabPaths = {
    ehole: `${basePath}/ehole/`,
    goby: `${basePath}/goby/`,
    wappalyzer: `${basePath}/wappalyzer/`,
    fingers: `${basePath}/fingers/`,
    fingerprinthub: `${basePath}/fingerprinthub/`,
    arl: `${basePath}/arl/`,
  }

  // Fingerprint library counts
  const counts = {
    ehole: stats?.ehole || 0,
    goby: stats?.goby || 0,
    wappalyzer: stats?.wappalyzer || 0,
    fingers: stats?.fingers || 0,
    fingerprinthub: stats?.fingerprinthub || 0,
    arl: stats?.arl || 0,
  }

  if (isLoading) {
    return (
      <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
        <div className="flex items-center justify-between px-4 lg:px-6">
          <div className="space-y-2">
            <Skeleton className="h-7 w-32" />
            <Skeleton className="h-4 w-48" />
          </div>
        </div>
        <div className="px-4 lg:px-6">
          <Skeleton className="h-10 w-96" />
        </div>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
      <PageHeader
        code="FPR-01"
        title={t("title")}
        description={t("pageDescription")}
      />

      {/* Tabs navigation */}
      <div className="flex items-center justify-between px-4 lg:px-6">
        <div className="flex items-center gap-3">
          <Tabs value={getActiveTab()} className="w-full">
            <TabsList>
              <TabsTrigger value="ehole" asChild>
                <Link href={tabPaths.ehole} className="flex items-center gap-0.5">
                  EHole
                  {counts.ehole > 0 && (
                    <Badge variant="secondary" className="ml-1.5 h-5 min-w-5 rounded-full px-1.5 text-xs">
                      {counts.ehole}
                    </Badge>
                  )}
                </Link>
              </TabsTrigger>
              <TabsTrigger value="goby" asChild>
                <Link href={tabPaths.goby} className="flex items-center gap-0.5">
                  Goby
                  {counts.goby > 0 && (
                    <Badge variant="secondary" className="ml-1.5 h-5 min-w-5 rounded-full px-1.5 text-xs">
                      {counts.goby}
                    </Badge>
                  )}
                </Link>
              </TabsTrigger>
              <TabsTrigger value="wappalyzer" asChild>
                <Link href={tabPaths.wappalyzer} className="flex items-center gap-0.5">
                  Wappalyzer
                  {counts.wappalyzer > 0 && (
                    <Badge variant="secondary" className="ml-1.5 h-5 min-w-5 rounded-full px-1.5 text-xs">
                      {counts.wappalyzer}
                    </Badge>
                  )}
                </Link>
              </TabsTrigger>
              <TabsTrigger value="fingers" asChild>
                <Link href={tabPaths.fingers} className="flex items-center gap-0.5">
                  Fingers
                  {counts.fingers > 0 && (
                    <Badge variant="secondary" className="ml-1.5 h-5 min-w-5 rounded-full px-1.5 text-xs">
                      {counts.fingers}
                    </Badge>
                  )}
                </Link>
              </TabsTrigger>
              <TabsTrigger value="fingerprinthub" asChild>
                <Link href={tabPaths.fingerprinthub} className="flex items-center gap-0.5">
                  FingerPrintHub
                  {counts.fingerprinthub > 0 && (
                    <Badge variant="secondary" className="ml-1.5 h-5 min-w-5 rounded-full px-1.5 text-xs">
                      {counts.fingerprinthub}
                    </Badge>
                  )}
                </Link>
              </TabsTrigger>
              <TabsTrigger value="arl" asChild>
                <Link href={tabPaths.arl} className="flex items-center gap-0.5">
                  ARL
                  {counts.arl > 0 && (
                    <Badge variant="secondary" className="ml-1.5 h-5 min-w-5 rounded-full px-1.5 text-xs">
                      {counts.arl}
                    </Badge>
                  )}
                </Link>
              </TabsTrigger>
            </TabsList>
          </Tabs>

          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <HelpCircle className="h-4 w-4 text-muted-foreground cursor-help" />
              </TooltipTrigger>
              <TooltipContent side="right" className="max-w-sm whitespace-pre-line">
                {t("helpText")}
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </div>
      </div>

      {/* Sub-page content */}
      {children}
    </div>
  )
}
