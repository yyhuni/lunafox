"use client"

import React from "react"
import { usePathname, useParams } from "next/navigation"
import Link from "next/link"
import { Target, LayoutDashboard, Package, FolderSearch, Image as ImageIcon, ShieldAlert, Settings, HelpCircle } from "@/components/icons"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Badge } from "@/components/ui/badge"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { useTarget } from "@/hooks/use-targets"
import { useTranslations } from "next-intl"
import type { TargetDetail } from "@/types/target.types"

/**
 * Target detail layout
 * Two-level navigation: Overview / Assets / Vulnerabilities
 * Assets has secondary navigation for different asset types
 */
export default function TargetLayout({
  children,
}: {
  children: React.ReactNode
}) {
  const { id } = useParams<{ id: string }>()
  const pathname = usePathname()
  const t = useTranslations("pages.targetDetail")

  // Use React Query to get target data
  const {
    data: target,
    isLoading,
    error
  } = useTarget(Number(id))

  // Get primary navigation active tab
  const getPrimaryTab = () => {
    if (pathname.includes("/overview")) return "overview"
    if (pathname.includes("/directories")) return "directories"
    if (pathname.includes("/screenshots")) return "screenshots"
    if (pathname.includes("/vulnerabilities")) return "vulnerabilities"
    if (pathname.includes("/settings")) return "settings"
    // All asset pages fall under "assets"
    if (
      pathname.includes("/websites") ||
      pathname.includes("/subdomain") ||
      pathname.includes("/ip-addresses") ||
      pathname.includes("/endpoints")
    ) {
      return "assets"
    }
    return "overview"
  }

  // Get secondary navigation active tab (for assets)
  const getSecondaryTab = () => {
    if (pathname.includes("/websites")) return "websites"
    if (pathname.includes("/subdomain")) return "subdomain"
    if (pathname.includes("/ip-addresses")) return "ip-addresses"
    if (pathname.includes("/endpoints")) return "endpoints"
    return "websites"
  }

  // Check if we should show secondary navigation
  const showSecondaryNav = getPrimaryTab() === "assets"

  // Tab path mapping
  const basePath = `/target/${id}`
  const primaryPaths = {
    overview: `${basePath}/overview/`,
    assets: `${basePath}/websites/`, // Default to websites when clicking assets
    directories: `${basePath}/directories/`,
    screenshots: `${basePath}/screenshots/`,
    vulnerabilities: `${basePath}/vulnerabilities/`,
    settings: `${basePath}/settings/`,
  }

  const secondaryPaths = {
    websites: `${basePath}/websites/`,
    subdomain: `${basePath}/subdomain/`,
    "ip-addresses": `${basePath}/ip-addresses/`,
    endpoints: `${basePath}/endpoints/`,
  }

  // Get counts for each tab from target data
  const targetSummary = (target as TargetDetail | undefined)?.summary
  const counts = {
    subdomain: targetSummary?.subdomains || 0,
    endpoints: targetSummary?.endpoints || 0,
    websites: targetSummary?.websites || 0,
    directories: targetSummary?.directories || 0,
    vulnerabilities: targetSummary?.vulnerabilities?.total || 0,
    "ip-addresses": targetSummary?.ips || 0,
    screenshots: targetSummary?.screenshots || 0,
  }

  // Calculate total assets count
  const totalAssets = counts.websites + counts.subdomain + counts["ip-addresses"] + counts.endpoints

  // Loading state
  if (isLoading) {
    return (
      <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
        {/* Header skeleton */}
        <div className="flex items-center gap-2 px-4 lg:px-6">
          <Skeleton className="h-4 w-16" />
          <span className="text-muted-foreground">/</span>
          <Skeleton className="h-4 w-32" />
        </div>
        {/* Tabs skeleton */}
        <div className="flex gap-1 px-4 lg:px-6">
          <Skeleton className="h-9 w-20" />
          <Skeleton className="h-9 w-20" />
          <Skeleton className="h-9 w-24" />
        </div>
      </div>
    )
  }

  // Error state
  if (error) {
    return (
      <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
        <div className="flex items-center justify-center py-12">
          <div className="text-center">
            <Target className="mx-auto text-destructive mb-4" />
            <h3 className="text-lg font-semibold mb-2">{t("error.title")}</h3>
            <p className="text-muted-foreground">
              {error.message || t("error.message")}
            </p>
          </div>
        </div>
      </div>
    )
  }

  if (!target) {
    return (
      <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
        <div className="flex items-center justify-center py-12">
          <div className="text-center">
            <Target className="mx-auto text-muted-foreground mb-4" />
            <h3 className="text-lg font-semibold mb-2">{t("notFound.title")}</h3>
            <p className="text-muted-foreground">
              {t("notFound.message", { id })}
            </p>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
      {/* Header: Page label + Target name */}
      <div className="flex items-center gap-2 text-sm px-4 lg:px-6">
        <span className="text-muted-foreground">{t("breadcrumb.targetDetail")}</span>
        <span className="text-muted-foreground">/</span>
        <span className="font-medium">{target.name}</span>
      </div>

      {/* Primary navigation */}
      <div className="flex items-center justify-between px-4 lg:px-6">
        <div className="flex items-center gap-3">
          <Tabs value={getPrimaryTab()}>
            <TabsList>
              <TabsTrigger value="overview" asChild>
                <Link href={primaryPaths.overview} className="flex items-center gap-1.5">
                  <LayoutDashboard className="h-4 w-4" />
                  {t("tabs.overview")}
                </Link>
              </TabsTrigger>
              <TabsTrigger value="assets" asChild>
                <Link href={primaryPaths.assets} className="flex items-center gap-1.5">
                  <Package className="h-4 w-4" />
                  {t("tabs.assets")}
                  {totalAssets > 0 && (
                    <Badge variant="secondary" className="ml-1.5 h-5 min-w-5 rounded-full px-1.5 text-xs">
                      {totalAssets}
                    </Badge>
                  )}
                </Link>
              </TabsTrigger>
              <TabsTrigger value="directories" asChild>
                <Link href={primaryPaths.directories} className="flex items-center gap-1.5">
                  <FolderSearch className="h-4 w-4" />
                  {t("tabs.directories")}
                  {counts.directories > 0 && (
                    <Badge variant="secondary" className="ml-1.5 h-5 min-w-5 rounded-full px-1.5 text-xs">
                      {counts.directories}
                    </Badge>
                  )}
                </Link>
              </TabsTrigger>
              <TabsTrigger value="screenshots" asChild>
                <Link href={primaryPaths.screenshots} className="flex items-center gap-1.5">
                  <ImageIcon className="h-4 w-4" />
                  {t("tabs.screenshots")}
                  {counts.screenshots > 0 && (
                    <Badge variant="secondary" className="ml-1.5 h-5 min-w-5 rounded-full px-1.5 text-xs">
                      {counts.screenshots}
                    </Badge>
                  )}
                </Link>
              </TabsTrigger>
              <TabsTrigger value="vulnerabilities" asChild>
                <Link href={primaryPaths.vulnerabilities} className="flex items-center gap-1.5">
                  <ShieldAlert className="h-4 w-4" />
                  {t("tabs.vulnerabilities")}
                  {counts.vulnerabilities > 0 && (
                    <Badge variant="secondary" className="ml-1.5 h-5 min-w-5 rounded-full px-1.5 text-xs">
                      {counts.vulnerabilities}
                    </Badge>
                  )}
                </Link>
              </TabsTrigger>
              <TabsTrigger value="settings" asChild>
                <Link href={primaryPaths.settings} className="flex items-center gap-1.5">
                  <Settings className="h-4 w-4" />
                  {t("tabs.settings")}
                </Link>
              </TabsTrigger>
            </TabsList>
          </Tabs>

          {getPrimaryTab() === "directories" && (
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <HelpCircle className="h-4 w-4 text-muted-foreground cursor-help" />
                </TooltipTrigger>
                <TooltipContent side="right" className="max-w-sm">
                  {t("directoriesHelp")}
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          )}
        </div>
      </div>

      {/* Secondary navigation (only for assets) */}
      {showSecondaryNav && (
        <div className="flex items-center px-4 lg:px-6">
          <Tabs value={getSecondaryTab()} className="w-full">
            <TabsList variant="minimal-tab">
              <TabsTrigger value="websites" variant="minimal-tab" asChild>
                <Link href={secondaryPaths.websites} className="flex items-center gap-0.5">
                  {t("tabs.websites")}
                  {counts.websites > 0 && (
                    <Badge variant="secondary" className="ml-1.5 h-5 min-w-5 rounded-full px-1.5 text-xs">
                      {counts.websites}
                    </Badge>
                  )}
                </Link>
              </TabsTrigger>
              <TabsTrigger value="subdomain" variant="minimal-tab" asChild>
                <Link href={secondaryPaths.subdomain} className="flex items-center gap-0.5">
                  {t("tabs.subdomains")}
                  {counts.subdomain > 0 && (
                    <Badge variant="secondary" className="ml-1.5 h-5 min-w-5 rounded-full px-1.5 text-xs">
                      {counts.subdomain}
                    </Badge>
                  )}
                </Link>
              </TabsTrigger>
              <TabsTrigger value="ip-addresses" variant="minimal-tab" asChild>
                <Link href={secondaryPaths["ip-addresses"]} className="flex items-center gap-0.5">
                  {t("tabs.ips")}
                  {counts["ip-addresses"] > 0 && (
                    <Badge variant="secondary" className="ml-1.5 h-5 min-w-5 rounded-full px-1.5 text-xs">
                      {counts["ip-addresses"]}
                    </Badge>
                  )}
                </Link>
              </TabsTrigger>
              <TabsTrigger value="endpoints" variant="minimal-tab" asChild>
                <Link href={secondaryPaths.endpoints} className="flex items-center gap-0.5">
                  {t("tabs.urls")}
                  {counts.endpoints > 0 && (
                    <Badge variant="secondary" className="ml-1.5 h-5 min-w-5 rounded-full px-1.5 text-xs">
                      {counts.endpoints}
                    </Badge>
                  )}
                </Link>
              </TabsTrigger>
            </TabsList>
          </Tabs>
        </div>
      )}

      {/* Sub-page content */}
      {children}
    </div>
  )
}
