"use client";
import React from "react";
import { usePathname, useParams } from "next/navigation";
import Link from "next/link";
import { LayoutOverview, Package, Settings, HelpCircle, semanticIcons } from "@/components/icons";
import { AppErrorState } from "@/components/shared/feedback/app-error-state";
import { DetailShellReadyProvider, useDetailShellReadySignal } from "@/components/shared/loading/detail-shell-ready-context";
import { DirectoriesViewRouteFallback } from "@/components/directories/directories-view-sections";
import { EndpointsDetailViewRouteFallback } from "@/components/endpoints/endpoints-detail-view-sections";
import { HiddenReadinessRouteBoundary } from "@/components/shared/loading/hidden-readiness-route-boundary";
import { IPAddressesViewRouteFallback } from "@/components/ip-addresses/ip-addresses-view-sections";
import { ScreenshotsGalleryRouteFallback } from "@/components/screenshots/screenshots-gallery-sections";
import { SubdomainsDetailViewRouteFallback } from "@/components/subdomains/subdomains-detail-view-sections";
import { TargetSettingsRouteFallback } from "@/components/target/target-settings-sections";
import { TargetOverviewLoadingState } from "@/components/target/target-overview-sections";
import { Tabs, TabsCountBadge, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger, } from "@/components/ui/tooltip";
import { VulnerabilitiesDetailViewRouteFallback } from "@/components/vulnerabilities/vulnerabilities-detail-view-sections";
import { WebsiteRelationEvidenceLoadingState } from "@/components/websites/website-relation-evidence-view";
import { WebsiteRelationDetailLoadingState } from "@/components/websites/website-relation-detail-view";
import { useTarget } from "@/hooks/use-targets";
import { useTranslations } from "next-intl";
import { createAppError } from "@/lib/errors/app-error";
import { normalizeError } from "@/lib/errors/normalize-error";
import type { TargetDetail } from "@/types/target.types";
import {
    TargetDetailShellContent,
    TARGET_DETAIL_SHELL_HANDOFF_CLASS,
    TargetDetailShellHeader,
    TargetDetailShellLayout,
    TargetDetailShellLoadingState,
    TargetDetailShellPrimaryTabs,
    TargetDetailShellSecondaryTabs,
} from "./target-detail-shell-layout";
const VulnerabilityIcon = semanticIcons.concept.vulnerability;
const DirectoryIcon = semanticIcons.concept.directory;
const ScreenshotIcon = semanticIcons.concept.screenshot;
const ENDPOINTS_DETAIL_FALLBACK_ROW_COUNT = 4;
const DETAIL_FALLBACK_TABLE_ROW_COUNT = 6;
const DETAIL_FALLBACK_MAX_ROWS = 10;
const DETAIL_FALLBACK_MIN_ROWS = 1;
const INITIAL_DETAIL_FALLBACK_ROW_COUNT = 2;
const INITIAL_SCREENSHOT_FALLBACK_ITEM_COUNT = 3;
const INITIAL_DIRECTORY_FALLBACK_ROW_COUNT = 1;
const TARGET_PRIMARY_TAB_ORDER = ["overview", "assets", "directories", "screenshots", "vulnerabilities", "settings"] as const;
const TARGET_SECONDARY_TAB_ORDER = ["websites", "subdomains", "ip-addresses", "endpoints"] as const;
const TARGET_SETTINGS_SECONDARY_TAB_ORDER = ["blacklist", "scheduled-scans"] as const;
type TargetPrimaryTab = (typeof TARGET_PRIMARY_TAB_ORDER)[number];
type TargetAssetSecondaryTab = (typeof TARGET_SECONDARY_TAB_ORDER)[number];
type TargetSettingsSecondaryTab = (typeof TARGET_SETTINGS_SECONDARY_TAB_ORDER)[number];
function getDetailFallbackRowCount(total: number | undefined) {
    if (total === undefined || total <= 0) {
        return DETAIL_FALLBACK_MIN_ROWS;
    }
    return Math.min(Math.max(Math.floor(total), DETAIL_FALLBACK_MIN_ROWS), DETAIL_FALLBACK_MAX_ROWS);
}
function DetailMetadataReadySignal() {
    useDetailShellReadySignal(true);
    return null;
}
/**
 * Target detail layout
 * Two-level navigation: Overview / Assets / Vulnerabilities
 * Assets has secondary navigation for different asset types
 */
export default function TargetLayout({ children, }: {
    children: React.ReactNode;
}) {
    const { id } = useParams<{
        id: string;
    }>();
    const pathname = usePathname();
    const t = useTranslations("pages.targetDetail");
    // Use React Query to get target data
    const { data: target, isLoading, error } = useTarget(Number(id));
    const targetDisplayName = target?.name;
    const normalizedError = error ? normalizeError(error) : null;
    const isWebsiteDetailRoute = /^\/targets\/[^/]+\/websites\/[^/]+/.test(pathname);
    // Get primary navigation active tab
    const getPrimaryTab = (): TargetPrimaryTab => {
        // Website detail sections belong to Assets; their local section name must not activate a target tab.
        if (isWebsiteDetailRoute)
            return "assets";
        if (pathname.includes("/overview"))
            return "overview";
        if (pathname.includes("/directories"))
            return "directories";
        if (pathname.includes("/screenshots"))
            return "screenshots";
        if (pathname.includes("/vulnerabilities"))
            return "vulnerabilities";
        if (pathname.includes("/settings"))
            return "settings";
        // All asset pages fall under "assets"
        if (pathname.includes("/websites") ||
            pathname.includes("/subdomains") ||
            pathname.includes("/ip-addresses") ||
            pathname.includes("/endpoints")) {
            return "assets";
        }
        return "overview";
    };
    // Get the secondary navigation active tab for the current primary workspace.
    const getSecondaryTab = (): TargetAssetSecondaryTab | TargetSettingsSecondaryTab => {
        if (pathname.includes("/settings/scheduled-scans"))
            return "scheduled-scans";
        if (pathname.includes("/settings"))
            return "blacklist";
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
    // Settings uses the same secondary rail as Assets, with a separate route set.
    const primaryTab = getPrimaryTab();
    const secondaryTab = getSecondaryTab();
    const showSecondaryNav = primaryTab === "settings" || (primaryTab === "assets" && !isWebsiteDetailRoute);
    // Tab path mapping
    const basePath = `/targets/${id}`;
    const primaryPaths = {
        overview: `${basePath}/overview/`,
        assets: `${basePath}/websites/`, // Default to websites when clicking assets
        directories: `${basePath}/directories/`,
        screenshots: `${basePath}/screenshots/`,
        vulnerabilities: `${basePath}/vulnerabilities/`,
        settings: `${basePath}/settings/`,
    };
    const secondaryPaths = {
        websites: `${basePath}/websites/`,
        subdomains: `${basePath}/subdomains/`,
        "ip-addresses": `${basePath}/ip-addresses/`,
        endpoints: `${basePath}/endpoints/`,
    };
    const settingsSecondaryPaths = {
        blacklist: `${basePath}/settings/`,
        "scheduled-scans": `${basePath}/settings/scheduled-scans/`,
    };
    const breadcrumbPrimaryLabels = {
        overview: t("tabs.overview"),
        assets: t("tabs.assets"),
        directories: t("tabs.directories"),
        screenshots: t("tabs.screenshots"),
        vulnerabilities: t("tabs.vulnerabilities"),
        settings: t("tabs.settings"),
    };
    const activePrimaryTabIndex = TARGET_PRIMARY_TAB_ORDER.indexOf(primaryTab);
    const primaryTabLabels = TARGET_PRIMARY_TAB_ORDER.map((tab) => breadcrumbPrimaryLabels[tab]);
    const secondaryTabLabels = primaryTab === "settings"
        ? TARGET_SETTINGS_SECONDARY_TAB_ORDER.map((tab) => tab === "blacklist" ? t("tabs.blacklist") : t("tabs.scheduledScans"))
        : TARGET_SECONDARY_TAB_ORDER.map((tab) => {
            if (tab === "ip-addresses")
                return t("tabs.ips");
            if (tab === "endpoints")
                return t("tabs.urls");
            return t(`tabs.${tab}`);
        });
    const activeSecondaryTabIndex = primaryTab === "settings"
        ? TARGET_SETTINGS_SECONDARY_TAB_ORDER.indexOf(secondaryTab as TargetSettingsSecondaryTab)
        : TARGET_SECONDARY_TAB_ORDER.indexOf(secondaryTab as TargetAssetSecondaryTab);
    const breadcrumbItems = [
        { label: t("breadcrumb.targetDetail"), href: "/targets/" },
        { label: targetDisplayName ?? t("breadcrumb.targetDetail"), href: primaryPaths.overview },
        { label: breadcrumbPrimaryLabels[primaryTab] },
    ];
    // Get counts for each tab from target data
    const targetSummary = (target as TargetDetail | undefined)?.summary;
    const counts = {
        subdomains: targetSummary?.subdomains || 0,
        endpoints: targetSummary?.endpoints || 0,
        websites: targetSummary?.websites || 0,
        directories: targetSummary?.directories || 0,
        vulnerabilities: targetSummary?.vulnerabilities?.total || 0,
        "ip-addresses": targetSummary?.ips || 0,
        screenshots: targetSummary?.screenshots || 0,
    };
    const ipAddressesFallbackRowCount = getDetailFallbackRowCount(targetSummary?.ips);
    // Calculate total assets count
    const totalAssets = counts.websites + counts.subdomains + counts["ip-addresses"] + counts.endpoints;
    const isEndpointRoute = primaryTab === "assets" && secondaryTab === "endpoints";
    const renderChildLoadingState = () => {
        if (primaryTab === "overview") {
            return (<div className="px-4 lg:px-6">
          <TargetOverviewLoadingState />
        </div>);
        }
        if (primaryTab === "directories") {
            return <DirectoriesViewRouteFallback rowCount={INITIAL_DIRECTORY_FALLBACK_ROW_COUNT} totalSize={0} showBulkAdd />;
        }
        if (primaryTab === "screenshots") {
            return <div className="px-4 lg:px-6"><ScreenshotsGalleryRouteFallback itemCount={INITIAL_SCREENSHOT_FALLBACK_ITEM_COUNT} showSelectionAction /></div>;
        }
        if (primaryTab === "vulnerabilities") {
            return <div className="px-4 lg:px-6"><VulnerabilitiesDetailViewRouteFallback rowCount={INITIAL_DETAIL_FALLBACK_ROW_COUNT} /></div>;
        }
        if (primaryTab === "settings") {
            const settingsSection = secondaryTab === "scheduled-scans" ? "scheduled-scans" : "blacklist";
            return <div className="flex min-h-0 flex-1 flex-col px-4 lg:px-6"><TargetSettingsRouteFallback rowCount={INITIAL_DETAIL_FALLBACK_ROW_COUNT} section={settingsSection} /></div>;
        }
        if (primaryTab === "assets") {
            if (secondaryTab === "websites") {
                return <div className="px-4 lg:px-6">{isWebsiteDetailRoute ? <WebsiteRelationDetailLoadingState /> : <WebsiteRelationEvidenceLoadingState />}</div>;
            }
            if (secondaryTab === "subdomains") {
                return <SubdomainsDetailViewRouteFallback rowCount={DETAIL_FALLBACK_TABLE_ROW_COUNT} />;
            }
            if (secondaryTab === "ip-addresses") {
                return <IPAddressesViewRouteFallback rowCount={ipAddressesFallbackRowCount} />;
            }
            if (isEndpointRoute) {
                return <EndpointsDetailViewRouteFallback rowCount={ENDPOINTS_DETAIL_FALLBACK_ROW_COUNT} />;
            }
        }
        return <div className="px-4 lg:px-6"><WebsiteRelationEvidenceLoadingState /></div>;
    };
    const renderShellLoadingState = () => (<TargetDetailShellLoadingState primaryTabLabels={primaryTabLabels} activePrimaryTabIndex={activePrimaryTabIndex} secondaryTabLabels={secondaryTabLabels} activeSecondaryTabIndex={activeSecondaryTabIndex} showSecondaryNav={showSecondaryNav}>
      {renderChildLoadingState()}
    </TargetDetailShellLoadingState>);
    let metadataErrorState: React.ReactNode = null;
    if (!isLoading && error) {
        const isMissingTarget = normalizedError?.kind === "resource-not-found";
        metadataErrorState = (<div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
        <AppErrorState error={normalizedError ?? createAppError("unexpected-error")} title={isMissingTarget ? t("notFound.title") : undefined} description={isMissingTarget ? t("notFound.message", { id }) : undefined} resourceLabel={t("breadcrumb.targetDetail")} actionHref="/targets/"/>
      </div>);
    }
    else if (!isLoading && !target) {
        metadataErrorState = (<div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
        <AppErrorState error={createAppError("resource-not-found", { retryable: false })} title={t("notFound.title")} description={t("notFound.message", { id })} resourceLabel={t("breadcrumb.targetDetail")} actionHref="/targets/"/>
      </div>);
    }
    const resolvedLayout = (<TargetDetailShellLayout>
      <TargetDetailShellHeader aria-label={t("breadcrumb.targetDetail")}>
        {breadcrumbItems.map((item, index) => (<React.Fragment key={`${item.label}-${item.href ?? "current"}`}>
          {index > 0 ? <span aria-hidden="true" className="shrink-0 text-muted-foreground">/</span> : null}
          {item.href ? (
            <Link href={item.href} className="min-w-0 shrink truncate rounded-sm text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
              {item.label}
            </Link>
          ) : <span aria-current="page" className="shrink-0 font-medium">{item.label}</span>}
        </React.Fragment>))}
      </TargetDetailShellHeader>

      {/* Primary navigation */}
      <TargetDetailShellPrimaryTabs>
        <div className="flex items-center gap-3">
          <Tabs value={getPrimaryTab()}>
            <TabsList>
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
              <TabsTrigger value="settings" render={<Link href={primaryPaths.settings} className="flex items-center gap-1.5"/>}>
                  <Settings className="h-4 w-4"/>
                  {t("tabs.settings")}
                </TabsTrigger>
            </TabsList>
          </Tabs>

          {getPrimaryTab() === "directories" && (<TooltipProvider>
              <Tooltip>
                <TooltipTrigger render={<HelpCircle className="h-4 w-4 text-muted-foreground cursor-help"/>}></TooltipTrigger>
                <TooltipContent side="right" className="max-w-sm">
                  {t("directoriesHelp")}
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>)}
        </div>
      </TargetDetailShellPrimaryTabs>

      {/* Secondary navigation for asset and settings workspaces */}
      {showSecondaryNav && (<TargetDetailShellSecondaryTabs>
          <Tabs value={getSecondaryTab()} className="w-full">
            <TabsList variant="content">
              {primaryTab === "settings" ? (<>
                <TabsTrigger value="blacklist" variant="content" render={<Link href={settingsSecondaryPaths.blacklist} className="flex items-center gap-0.5"/>}>
                    {t("tabs.blacklist")}
                  </TabsTrigger>
                <TabsTrigger value="scheduled-scans" variant="content" render={<Link href={settingsSecondaryPaths["scheduled-scans"]} className="flex items-center gap-0.5"/>}>
                    {t("tabs.scheduledScans")}
                  </TabsTrigger>
              </>) : (<>
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
              </>)}
            </TabsList>
          </Tabs>
        </TargetDetailShellSecondaryTabs>)}

      {/* Sub-page content */}
      <TargetDetailShellContent>{children}</TargetDetailShellContent>
    </TargetDetailShellLayout>);
    return (<HiddenReadinessRouteBoundary key={`target-detail-shell-${id}`} owner="target-detail-shell" layer="workspace" intent="data" transitionMode="replace" skeleton={renderShellLoadingState()} className={TARGET_DETAIL_SHELL_HANDOFF_CLASS} skeletonClassName={TARGET_DETAIL_SHELL_HANDOFF_CLASS} contentClassName={TARGET_DETAIL_SHELL_HANDOFF_CLASS}>
        {({ onReady, deferInitialSkeleton }) => {
        if (isLoading) {
            return null;
        }
        return (<DetailShellReadyProvider value={{ onReady, deferInitialSkeleton }}>
            {metadataErrorState ? <DetailMetadataReadySignal /> : null}
            {metadataErrorState ?? resolvedLayout}
          </DetailShellReadyProvider>);
    }}
      </HiddenReadinessRouteBoundary>);
}
