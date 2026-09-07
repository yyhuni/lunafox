"use client";
import React from "react";
import { usePathname, useParams } from "next/navigation";
import Link from "next/link";
import { LayoutOverview, Package, semanticIcons } from "@/components/icons";
import { DetailShellReadyProvider } from "@/components/shared/loading/detail-shell-ready-context";
import { Tabs, TabsCountBadge, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { DirectoriesViewRouteFallback } from "@/components/directories/directories-view-sections";
import { EndpointsDetailViewRouteFallback } from "@/components/endpoints/endpoints-detail-view-sections";
import { HiddenReadinessRouteBoundary } from "@/components/shared/loading/hidden-readiness-route-boundary";
import { IPAddressesViewRouteFallback } from "@/components/ip-addresses/ip-addresses-view-sections";
import { ScanOverviewLoadingState } from "@/components/scan/history/scan-overview-loading-state";
import { ScreenshotsGalleryRouteFallback } from "@/components/screenshots/screenshots-gallery-sections";
import { SubdomainsDetailViewRouteFallback } from "@/components/subdomains/subdomains-detail-view-sections";
import { useScan } from "@/hooks/use-scans";
import { useTranslations } from "next-intl";
import { VulnerabilitiesDetailViewRouteFallback } from "@/components/vulnerabilities/vulnerabilities-detail-view-sections";
import { WebSitesViewRouteFallback } from "@/components/websites/websites-view-sections";
import {
    ScanHistoryDetailShellContent,
    SCAN_HISTORY_DETAIL_SHELL_HANDOFF_CLASS,
    ScanHistoryDetailShellHeader,
    ScanHistoryDetailShellLayout,
    ScanHistoryDetailShellLoadingState,
    ScanHistoryDetailShellPrimaryTabs,
    ScanHistoryDetailShellSecondaryTabs,
} from "./scan-history-detail-shell-layout";
const VulnerabilityIcon = semanticIcons.concept.vulnerability;
const DirectoryIcon = semanticIcons.concept.directory;
const ScreenshotIcon = semanticIcons.concept.screenshot;
const ENDPOINTS_DETAIL_FALLBACK_ROW_COUNT = 4;
const DETAIL_FALLBACK_TABLE_ROW_COUNT = 6;
const INITIAL_WEBSITE_FALLBACK_ROW_COUNT = 2;
const INITIAL_DIRECTORY_FALLBACK_ROW_COUNT = 1;
const SCAN_HISTORY_PRIMARY_TAB_ORDER = ["overview", "assets", "directories", "screenshots", "vulnerabilities"] as const;
const SCAN_HISTORY_SECONDARY_TAB_ORDER = ["websites", "subdomains", "ip-addresses", "endpoints"] as const;
type ScanHistoryPrimaryTab = (typeof SCAN_HISTORY_PRIMARY_TAB_ORDER)[number];
export default function ScanHistoryLayout({ children, }: {
    children: React.ReactNode;
}) {
    const { id } = useParams<{
        id: string;
    }>();
    const pathname = usePathname();
    const { data: scanData, isLoading } = useScan(parseInt(id));
    const t = useTranslations("scan.history");
    // Get primary navigation active tab
    const getPrimaryTab = (): ScanHistoryPrimaryTab => {
        if (pathname.includes("/overview"))
            return "overview";
        if (pathname.includes("/directories"))
            return "directories";
        if (pathname.includes("/screenshots"))
            return "screenshots";
        if (pathname.includes("/vulnerabilities"))
            return "vulnerabilities";
        // All asset pages fall under "assets"
        if (pathname.includes("/websites") ||
            pathname.includes("/subdomains") ||
            pathname.includes("/ip-addresses") ||
            pathname.includes("/endpoints")) {
            return "assets";
        }
        return "overview";
    };
    const getSecondaryTab = () => {
        if (pathname.includes("/websites"))
            return "websites";
        if (pathname.includes("/subdomains"))
            return "subdomains";
        if (pathname.includes("/ip-addresses"))
            return "ip-addresses";
        if (pathname.includes("/endpoints"))
            return "endpoints";
        return "websites";
    };
    // Check if we should show secondary navigation
    const primaryTab = getPrimaryTab();
    const secondaryTab = getSecondaryTab();
    const activePrimaryTabIndex = SCAN_HISTORY_PRIMARY_TAB_ORDER.indexOf(primaryTab);
    const showSecondaryNav = primaryTab === "assets";
    const isEndpointRoute = secondaryTab === "endpoints";
    const basePath = `/scan/history/${id}`;
    const primaryPaths = {
        overview: `${basePath}/overview/`,
        assets: `${basePath}/websites/`, // Default to websites when clicking assets
        directories: `${basePath}/directories/`,
        screenshots: `${basePath}/screenshots/`,
        vulnerabilities: `${basePath}/vulnerabilities/`,
    };
    const primaryTabLabels = SCAN_HISTORY_PRIMARY_TAB_ORDER.map((tab) => {
        if (tab === "overview")
            return t("tabs.overview");
        if (tab === "assets")
            return t("tabs.assets");
        if (tab === "directories")
            return t("tabs.directories");
        if (tab === "screenshots")
            return t("tabs.screenshots");
        return t("tabs.vulnerabilities");
    });
    const secondaryTabLabels = SCAN_HISTORY_SECONDARY_TAB_ORDER.map((tab) => {
        if (tab === "ip-addresses")
            return t("tabs.ips");
        if (tab === "endpoints")
            return t("tabs.urls");
        return t(`tabs.${tab}`);
    });
    const secondaryPaths = {
        websites: `${basePath}/websites/`,
        subdomains: `${basePath}/subdomains/`,
        "ip-addresses": `${basePath}/ip-addresses/`,
        endpoints: `${basePath}/endpoints/`,
    };
    // Get counts for each tab from scan data
    const stats = scanData?.cachedStats;
    const counts = {
        subdomains: stats?.subdomainsCount || 0,
        endpoints: stats?.endpointsCount || 0,
        websites: stats?.websitesCount || 0,
        directories: stats?.directoriesCount || 0,
        screenshots: stats?.screenshotsCount || 0,
        vulnerabilities: stats?.vulnsTotal || 0,
        "ip-addresses": stats?.ipsCount || 0,
    };
    // Calculate total assets count
    const totalAssets = counts.websites + counts.subdomains + counts["ip-addresses"] + counts.endpoints;
    const renderChildLoadingState = () => {
        if (primaryTab === "overview") {
            return (<div className="flex min-h-0 min-w-0 flex-1 flex-col px-4 lg:px-6">
          <ScanOverviewLoadingState />
        </div>);
        }
        if (primaryTab === "directories") {
            return <DirectoriesViewRouteFallback rowCount={INITIAL_DIRECTORY_FALLBACK_ROW_COUNT} totalSize={0} />;
        }
        if (primaryTab === "screenshots") {
            return <ScreenshotsGalleryRouteFallback />;
        }
        if (primaryTab === "vulnerabilities") {
            return <VulnerabilitiesDetailViewRouteFallback rowCount={DETAIL_FALLBACK_TABLE_ROW_COUNT} />;
        }
        if (primaryTab === "assets") {
            if (secondaryTab === "websites") {
                return <WebSitesViewRouteFallback rowCount={INITIAL_WEBSITE_FALLBACK_ROW_COUNT} totalSize={0} />;
            }
            if (secondaryTab === "subdomains") {
                return <SubdomainsDetailViewRouteFallback rowCount={DETAIL_FALLBACK_TABLE_ROW_COUNT} />;
            }
            if (secondaryTab === "ip-addresses") {
                return <IPAddressesViewRouteFallback rowCount={DETAIL_FALLBACK_TABLE_ROW_COUNT} />;
            }
            if (isEndpointRoute) {
                return <EndpointsDetailViewRouteFallback rowCount={ENDPOINTS_DETAIL_FALLBACK_ROW_COUNT} />;
            }
        }
        return <WebSitesViewRouteFallback rowCount={INITIAL_WEBSITE_FALLBACK_ROW_COUNT} totalSize={0} />;
    };
    const renderShellLoadingState = () => (<ScanHistoryDetailShellLoadingState primaryTabLabels={primaryTabLabels} activePrimaryTabIndex={activePrimaryTabIndex} secondaryTabLabels={secondaryTabLabels} showSecondaryNav={showSecondaryNav}>
      {renderChildLoadingState()}
    </ScanHistoryDetailShellLoadingState>);
    const resolvedLayout = (<ScanHistoryDetailShellLayout>
      {/* Header: Page label + Scan info */}
      <ScanHistoryDetailShellHeader>
        <Link href="/scan/history/" className="rounded-sm text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
          {t("breadcrumb.scanHistory")}
        </Link>
        <span className="text-muted-foreground">/</span>
        <span className="font-medium">
          {scanData?.target?.displayName || scanData?.target?.name || t("taskId", { id })}
        </span>
      </ScanHistoryDetailShellHeader>

      {/* Primary navigation */}
      <ScanHistoryDetailShellPrimaryTabs>
        <Tabs value={getPrimaryTab()} className="w-max min-w-full">
          <TabsList className="max-w-none min-w-max">
            <TabsTrigger value="overview" render={<Link href={primaryPaths.overview} className="flex items-center gap-1.5"/>}>
                <LayoutOverview className="h-4 w-4"/>
                {t("tabs.overview")}
              </TabsTrigger>
            <TabsTrigger value="assets" render={<Link href={primaryPaths.assets} className="flex items-center gap-1.5"/>}>
                <Package className="h-4 w-4"/>
                {t("tabs.assets")}
                {totalAssets > 0 && (<TabsCountBadge>
                    {totalAssets}
                  </TabsCountBadge>)}
              </TabsTrigger>
            <TabsTrigger value="directories" render={<Link href={primaryPaths.directories} className="flex items-center gap-1.5"/>}>
                <DirectoryIcon className="h-4 w-4"/>
                {t("tabs.directories")}
                {counts.directories > 0 && (<TabsCountBadge>
                    {counts.directories}
                  </TabsCountBadge>)}
              </TabsTrigger>
            <TabsTrigger value="screenshots" render={<Link href={primaryPaths.screenshots} className="flex items-center gap-1.5"/>}>
                <ScreenshotIcon className="h-4 w-4"/>
                {t("tabs.screenshots")}
                {counts.screenshots > 0 && (<TabsCountBadge>
                    {counts.screenshots}
                  </TabsCountBadge>)}
              </TabsTrigger>
            <TabsTrigger value="vulnerabilities" render={<Link href={primaryPaths.vulnerabilities} className="flex items-center gap-1.5"/>}>
                <VulnerabilityIcon className="h-4 w-4"/>
                {t("tabs.vulnerabilities")}
                {counts.vulnerabilities > 0 && (<TabsCountBadge>
                    {counts.vulnerabilities}
                  </TabsCountBadge>)}
              </TabsTrigger>
          </TabsList>
        </Tabs>
      </ScanHistoryDetailShellPrimaryTabs>

      {/* Secondary navigation (only for assets) */}
      {showSecondaryNav && (<ScanHistoryDetailShellSecondaryTabs>
          <Tabs value={getSecondaryTab()} className="w-full">
            <TabsList variant="content" className="min-w-max">
              <TabsTrigger value="websites" variant="content" render={<Link href={secondaryPaths.websites} className="flex items-center gap-0.5"/>}>
                  {t("tabs.websites")}
                  {counts.websites > 0 && (<TabsCountBadge>
                      {counts.websites}
                    </TabsCountBadge>)}
                </TabsTrigger>
              <TabsTrigger value="subdomains" variant="content" render={<Link href={secondaryPaths.subdomains} className="flex items-center gap-0.5"/>}>
                  {t("tabs.subdomains")}
                  {counts.subdomains > 0 && (<TabsCountBadge>
                      {counts.subdomains}
                    </TabsCountBadge>)}
                </TabsTrigger>
              <TabsTrigger value="ip-addresses" variant="content" render={<Link href={secondaryPaths["ip-addresses"]} className="flex items-center gap-0.5"/>}>
                  {t("tabs.ips")}
                  {counts["ip-addresses"] > 0 && (<TabsCountBadge>
                      {counts["ip-addresses"]}
                    </TabsCountBadge>)}
                </TabsTrigger>
              <TabsTrigger value="endpoints" variant="content" render={<Link href={secondaryPaths.endpoints} className="flex items-center gap-0.5"/>}>
                  {t("tabs.urls")}
                  {counts.endpoints > 0 && (<TabsCountBadge>
                      {counts.endpoints}
                    </TabsCountBadge>)}
                </TabsTrigger>
            </TabsList>
          </Tabs>
        </ScanHistoryDetailShellSecondaryTabs>)}

      {/* Sub-page content */}
      <ScanHistoryDetailShellContent>{children}</ScanHistoryDetailShellContent>
    </ScanHistoryDetailShellLayout>);
    return (<HiddenReadinessRouteBoundary key={`scan-history-detail-shell-${id}`} owner="scan-history-detail-shell" layer="workspace" intent="data" transitionMode="replace" skeleton={renderShellLoadingState()} className={SCAN_HISTORY_DETAIL_SHELL_HANDOFF_CLASS} skeletonClassName={SCAN_HISTORY_DETAIL_SHELL_HANDOFF_CLASS} contentClassName={SCAN_HISTORY_DETAIL_SHELL_HANDOFF_CLASS}>
        {({ onReady, deferInitialSkeleton }) => {
        if (isLoading) {
            return null;
        }
        return (<DetailShellReadyProvider value={{ onReady, deferInitialSkeleton }}>
            {resolvedLayout}
          </DetailShellReadyProvider>);
    }}
      </HiddenReadinessRouteBoundary>);
}
