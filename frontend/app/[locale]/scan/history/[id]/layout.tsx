"use client"

import React from "react"
import { usePathname, useParams } from "next/navigation"
import Link from "next/link"
import { LayoutDashboard, Package, FolderSearch, Image as ImageIcon, ShieldAlert } from "@/components/icons"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { useScan } from "@/hooks/use-scans"
import { useTranslations } from "next-intl"

export default function ScanHistoryLayout({
  children,
}: {
  children: React.ReactNode
}) {
  const { id } = useParams<{ id: string }>()
  const pathname = usePathname()
  const { data: scanData, isLoading } = useScan(parseInt(id))
  const t = useTranslations("scan.history")

  // Get primary navigation active tab
  const getPrimaryTab = () => {
    if (pathname.includes("/overview")) return "overview"
    if (pathname.includes("/directories")) return "directories"
    if (pathname.includes("/screenshots")) return "screenshots"
    if (pathname.includes("/vulnerabilities")) return "vulnerabilities"
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

  const basePath = `/scan/history/${id}`
  const primaryPaths = {
    overview: `${basePath}/overview/`,
    assets: `${basePath}/websites/`, // Default to websites when clicking assets
    directories: `${basePath}/directories/`,
    screenshots: `${basePath}/screenshots/`,
    vulnerabilities: `${basePath}/vulnerabilities/`,
  }

  const secondaryPaths = {
    websites: `${basePath}/websites/`,
    subdomain: `${basePath}/subdomain/`,
    "ip-addresses": `${basePath}/ip-addresses/`,
    endpoints: `${basePath}/endpoints/`,
  }

  // Get counts for each tab from scan data
  const stats = scanData?.cachedStats
  const counts = {
    subdomain: stats?.subdomainsCount || 0,
    endpoints: stats?.endpointsCount || 0,
    websites: stats?.websitesCount || 0,
    directories: stats?.directoriesCount || 0,
    screenshots: stats?.screenshotsCount || 0,
    vulnerabilities: stats?.vulnsTotal || 0,
    "ip-addresses": stats?.ipsCount || 0,
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

  return (
    <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6 h-full">
      {/* Header: Page label + Scan info */}
      <div className="flex items-center gap-2 text-sm px-4 lg:px-6">
        <span className="text-muted-foreground">{t("breadcrumb.scanHistory")}</span>
        <span className="text-muted-foreground">/</span>
        <span className="font-medium">
          {scanData?.target?.name || t("taskId", { id })}
        </span>
      </div>

      {/* Primary navigation */}
      <div className="px-4 lg:px-6">
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
          </TabsList>
        </Tabs>
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
