"use client"; // Mark as client component, can use browser APIs and interactive features
// Import React library
import React from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { IconChevronRight, // Right arrow icon
semanticIcons, type Icon, } from "@/components/icons";
// Import internationalization hook
import { useTranslations } from 'next-intl';
// Import custom navigation components
import { NavUser, NavUserSkeleton } from "@/components/nav-user";
import { AboutDialog } from "@/components/about-dialog";
import { LunaFoxMark } from "@/components/brand/lunafox-mark";
// Import sidebar UI components
import { Sidebar, SidebarContent, SidebarFooter, SidebarHeader, SidebarMenu, SidebarMenuBadge, SidebarMenuButton, SidebarMenuItem, SidebarMenuSub, SidebarMenuSubButton, SidebarMenuSubItem, SidebarGroup, SidebarGroupLabel, SidebarGroupContent, SidebarNavigationPendingIndicator, useSidebar, } from "@/components/ui/sidebar";
// Import collapsible component
import { Collapsible, CollapsibleContent, CollapsibleTrigger, } from "@/components/ui/collapsible";
import { textRole } from "@/lib/typography";
import { shellOverlaySideOffsets } from "@/lib/ui/overlay-styles";
import { useLoginVisualDiscoverability } from "@/hooks/use-login-visual";
import { cn } from "@/lib/utils";
const AboutIcon = semanticIcons.navigation.about;
const SupportAuthorIcon = semanticIcons.navigation.support;
// Keep the framework's automatic route prefetch policy.
const sidebarLinkPrefetch = null;
type AppSidebarNavItem = {
    title: string;
    icon: Icon;
    url?: string;
    badge?: string;
    items?: Array<{
        title: string;
        url: string;
    }>;
};
type AppSidebarNavGroup = {
    label?: string;
    items: AppSidebarNavItem[];
};
type AppSidebarCollapsedMenuState = {
    key: string;
    title: string;
    icon: Icon;
    items: NonNullable<AppSidebarNavItem["items"]>;
    top: number;
    left: number;
};
function isSidebarNavigationItemActive(item: AppSidebarNavItem, current: string, normalize: (path: string) => string) {
    const ownUrl = item.url ? normalize(item.url) : null;
    if (ownUrl && (ownUrl === "/" ? current === "/" : current === ownUrl || current.startsWith(ownUrl + "/")))
        return true;
    return item.items?.some((subItem) => {
        const subUrl = normalize(subItem.url);
        return current === subUrl || current.startsWith(subUrl + "/");
    }) ?? false;
}
function AppSidebarCollapsibleNavItem({ collapsedMenuKey, current, isCollapsedDesktop, item, normalize, onCollapsedMenuClose, onCollapsedMenuOpen, }: {
    collapsedMenuKey?: string;
    current: string;
    isCollapsedDesktop: boolean;
    item: AppSidebarNavItem;
    normalize: (path: string) => string;
    onCollapsedMenuClose: () => void;
    onCollapsedMenuOpen: (menu: AppSidebarCollapsedMenuState) => void;
}) {
    const isActive = isSidebarNavigationItemActive(item, current, normalize);
    const [isOpen, setIsOpen] = React.useState(isActive);
    const shouldRenderSubItems = isOpen || isActive;
    const ItemIcon = item.icon;
    const itemKey = item.url ?? item.title;
    const isCollapsedMenuOpen = collapsedMenuKey === itemKey;
    const openCollapsedMenu = React.useCallback((event: React.MouseEvent | React.FocusEvent) => {
        const rect = event.currentTarget.getBoundingClientRect();
        onCollapsedMenuOpen({
            key: itemKey,
            title: item.title,
            icon: item.icon,
            items: item.items!,
            top: rect.top,
            left: rect.right + shellOverlaySideOffsets.sidebarDesktop,
        });
    }, [item.icon, item.items, item.title, itemKey, onCollapsedMenuOpen]);
    React.useEffect(() => {
        if (isActive)
            setIsOpen(true);
    }, [isActive]);
    if (isCollapsedDesktop) {
        return (<SidebarMenuItem>
        <SidebarMenuButton type="button" aria-label={item.title} aria-expanded={isCollapsedMenuOpen} onMouseEnter={openCollapsedMenu} onMouseLeave={onCollapsedMenuClose} onFocus={openCollapsedMenu} onBlur={onCollapsedMenuClose}>
          <ItemIcon />
          <span>{item.title}</span>
          <IconChevronRight className="ml-auto"/>
        </SidebarMenuButton>
      </SidebarMenuItem>);
    }
    return (<Collapsible open={isOpen} onOpenChange={setIsOpen} className="group/collapsible">
      <SidebarMenuItem>
        <CollapsibleTrigger render={<SidebarMenuButton/>}>
            <ItemIcon />
            <span>{item.title}</span>
            <IconChevronRight className="ml-auto"/>
          </CollapsibleTrigger>
        {shouldRenderSubItems ? (<CollapsibleContent>
            <SidebarMenuSub className="relative my-2 pl-6 before:absolute before:top-1 before:bottom-1 before:left-4 before:w-px before:rounded-full before:bg-sidebar-border/60 before:content-['']">
              {item.items!.map((subItem) => {
                const subUrl = normalize(subItem.url);
                const isSubActive = current === subUrl || current.startsWith(subUrl + "/");
                return (<SidebarMenuSubItem key={subItem.title}>
                    <SidebarMenuSubButton isActive={isSubActive} render={<Link href={subItem.url} prefetch={sidebarLinkPrefetch}/>}>
                        <SidebarNavigationPendingIndicator />
                        <span>{subItem.title}</span>
                      </SidebarMenuSubButton>
                  </SidebarMenuSubItem>);
            })}
            </SidebarMenuSub>
          </CollapsibleContent>) : null}
      </SidebarMenuItem>
    </Collapsible>);
}
function AppSidebarNavigationGroup({ collapsedMenuKey, current, group, isCollapsedDesktop, normalize, onCollapsedMenuClose, onCollapsedMenuOpen, }: {
    collapsedMenuKey?: string;
    current: string;
    group: AppSidebarNavGroup;
    isCollapsedDesktop: boolean;
    normalize: (path: string) => string;
    onCollapsedMenuClose: () => void;
    onCollapsedMenuOpen: (menu: AppSidebarCollapsedMenuState) => void;
}) {
    return (<SidebarGroup>
      {group.label ? <SidebarGroupLabel>{group.label}</SidebarGroupLabel> : null}
      <SidebarGroupContent>
        <SidebarMenu>
          {group.items.map((item) => {
            const isActive = isSidebarNavigationItemActive(item, current, normalize);
            const hasSubItems = item.items && item.items.length > 0;
            if (!hasSubItems) {
                if (!item.url)
                    throw new Error(`Sidebar navigation item "${item.title}" requires a route or submenu`);
                return (<SidebarMenuItem key={item.title}>
                  <SidebarMenuButton data-sidebar-has-badge={item.badge ? "true" : undefined} isActive={isActive} tooltip={item.title} render={<Link href={item.url} prefetch={sidebarLinkPrefetch}/>}>
                    <item.icon />
                    <SidebarNavigationPendingIndicator />
                    <span>{item.title}</span>
                  </SidebarMenuButton>
                  {item.badge ? <SidebarMenuBadge className="radius-badge border border-sidebar-border bg-sidebar">{item.badge}</SidebarMenuBadge> : null}
                </SidebarMenuItem>);
            }
            return (<AppSidebarCollapsibleNavItem collapsedMenuKey={collapsedMenuKey} key={item.title} current={current} isCollapsedDesktop={isCollapsedDesktop} item={item} normalize={normalize} onCollapsedMenuClose={onCollapsedMenuClose} onCollapsedMenuOpen={onCollapsedMenuOpen}/>);
          })}
        </SidebarMenu>
      </SidebarGroupContent>
    </SidebarGroup>);
}
function CollapsedSidebarSubmenu({ current, menu, normalize, onClose, onCloseImmediately, onKeepOpen, }: {
    current: string;
    menu: AppSidebarCollapsedMenuState;
    normalize: (path: string) => string;
    onClose: () => void;
    onCloseImmediately: () => void;
    onKeepOpen: () => void;
}) {
    const MenuIcon = menu.icon;
    return (<div className="fixed z-50 w-max min-w-40 max-w-56" data-sidebar-collapsed-submenu="true" onMouseEnter={onKeepOpen} onMouseLeave={onClose} onFocus={onKeepOpen} onBlur={onClose} style={{ transform: `translate3d(${menu.left}px, ${menu.top}px, 0)` }}>
      <div className="radius-overlay border border-border bg-popover p-1 text-popover-foreground shadow-md">
        <div className={cn("flex max-w-full items-center gap-2 truncate px-2 py-1 text-sidebar-foreground/70", textRole.helperText)}>
          <MenuIcon className="size-4 shrink-0"/>
          <span className="truncate">{menu.title}</span>
        </div>
        <div className="flex min-w-0 flex-col gap-1">
          {menu.items.map((subItem) => {
            const subUrl = normalize(subItem.url);
            const isSubActive = current === subUrl || current.startsWith(subUrl + "/");
            return (<Link key={subItem.title} href={subItem.url} prefetch={sidebarLinkPrefetch} onClick={onCloseImmediately} className={cn("radius-control flex min-w-0 items-center px-2 py-1.5 text-sidebar-foreground/65 outline-none hover:bg-sidebar-accent hover:text-sidebar-accent-foreground focus-visible:bg-sidebar-accent focus-visible:text-sidebar-accent-foreground focus-visible:ring-2 focus-visible:ring-ring", textRole.navLabel, isSubActive && "bg-sidebar-accent text-sidebar-accent-foreground font-medium")}>
                <span className="truncate">{subItem.title}</span>
              </Link>);
        })}
        </div>
      </div>
    </div>);
}
/**
 * Application sidebar component
 * Displays the main navigation menu of the application, including user info, main menu, documents and secondary menu
 * Supports expand and collapse functionality for submenus
 * @param props - All properties of the Sidebar component
 */
export function AppSidebar({ warmup = false, ...props }: React.ComponentProps<typeof Sidebar> & {
    warmup?: boolean;
}) {
    const { state, isMobile, setOpenMobile } = useSidebar();
    const t = useTranslations('navigation');
    const pathname = usePathname();
    const { unlocked: loginVisualUnlocked } = useLoginVisualDiscoverability();
    const normalize = (p: string) => (p !== "/" && p.endsWith("/") ? p.slice(0, -1) : p);
    const current = normalize(pathname);
    const [collapsedMenu, setCollapsedMenu] = React.useState<AppSidebarCollapsedMenuState | null>(null);
    const closeTimerRef = React.useRef<number | null>(null);
    const isCollapsedDesktop = state === "collapsed" && !isMobile;
    const clearCollapsedMenuCloseTimer = React.useCallback(() => {
        if (closeTimerRef.current) {
            window.clearTimeout(closeTimerRef.current);
            closeTimerRef.current = null;
        }
    }, []);
    const openCollapsedMenu = React.useCallback((menu: AppSidebarCollapsedMenuState) => {
        clearCollapsedMenuCloseTimer();
        setCollapsedMenu(menu);
    }, [clearCollapsedMenuCloseTimer]);
    const closeCollapsedMenu = React.useCallback(() => {
        clearCollapsedMenuCloseTimer();
        closeTimerRef.current = window.setTimeout(() => {
            setCollapsedMenu(null);
            closeTimerRef.current = null;
        }, 180);
    }, [clearCollapsedMenuCloseTimer]);
    const closeCollapsedMenuImmediately = React.useCallback(() => {
        clearCollapsedMenuCloseTimer();
        setCollapsedMenu(null);
    }, [clearCollapsedMenuCloseTimer]);
    const previousPathRef = React.useRef(current);
    React.useEffect(() => {
        if (previousPathRef.current !== current) {
            previousPathRef.current = current;
            closeCollapsedMenuImmediately();
            // Close only after router commit so cancelled mobile navigation can restore in place.
            if (isMobile)
                setOpenMobile(false);
        }
    }, [closeCollapsedMenuImmediately, current, isMobile, setOpenMobile]);
    React.useEffect(() => {
        if (!isCollapsedDesktop)
            setCollapsedMenu(null);
    }, [isCollapsedDesktop]);
    React.useEffect(() => clearCollapsedMenuCloseTimer, [clearCollapsedMenuCloseTimer]);
    const user = React.useMemo(() => ({
        name: "admin",
        email: "admin@admin.com",
    }), []);
    const navGroups = React.useMemo<AppSidebarNavGroup[]>(() => [
        {
            label: t('workspace'),
            items: [
                {
                    title: t('overview'),
                    url: "/overview/",
                    icon: semanticIcons.navigation.overview,
                },
                {
                    title: t('search'),
                    url: "/search/",
                    icon: semanticIcons.navigation.search,
                },
            ],
        },
        {
            label: t('assetsAndRisk'),
            items: [
                {
                    title: t('organization'),
                    url: "/organizations/",
                    icon: semanticIcons.concept.organization,
                },
                {
                    title: t('target'),
                    url: "/targets/",
                    icon: semanticIcons.concept.target,
                },
                {
                    title: t('vulnerabilities'),
                    url: "/vulnerabilities/",
                    icon: semanticIcons.concept.vulnerability,
                },
            ],
        },
        {
            label: t('scanning'),
            items: [
                {
                    title: t('scanHistory'),
                    url: "/scan/history/",
                    icon: semanticIcons.concept.scan,
                },
                {
                    title: t('scheduledScan'),
                    url: "/scan/scheduled/",
                    icon: semanticIcons.concept.scheduledScan,
                },
                {
                    title: t('scanConfiguration'),
                    url: "/scan/config/workflows/",
                    icon: semanticIcons.concept.workflow,
                    badge: t('beta'),
                },
            ],
        },
        {
            label: t('tools'),
            items: [
                {
                    title: t('wordlists'),
                    url: "/tools/wordlists/",
                    icon: semanticIcons.concept.wordlist,
                },
                {
                    title: t('fingerprints'),
                    url: "/tools/fingerprints/fingerprinthub/",
                    icon: semanticIcons.concept.fingerprint,
                },
                {
                    title: t('nucleiTemplates'),
                    url: "/tools/nuclei/",
                    icon: semanticIcons.concept.nucleiRepository,
                },
            ],
        },
        {
            label: t('system'),
            items: [
                {
                    title: t('nodesManagement'),
                    url: "/settings/agents/",
                    icon: semanticIcons.concept.agent,
                },
                {
                    title: t('systemLogs'),
                    url: "/settings/system-logs/",
                    icon: semanticIcons.concept.systemLog,
                },
                {
                    title: t('databaseHealth'),
                    url: "/settings/database-health/",
                    icon: semanticIcons.concept.database,
                },
                {
                    title: t('systemSettings'),
                    icon: semanticIcons.concept.apiKey,
                    items: [
                        {
                            title: t('notifications'),
                            url: "/settings/notifications/",
                        },
                        {
                            title: t('apiKeys'),
                            url: "/settings/api-keys/",
                        },
                        {
                            title: t('globalBlacklist'),
                            url: "/settings/blacklist/",
                        },
                        ...(loginVisualUnlocked ? [{
                            title: t('loginVisual'),
                            url: "/settings/login-visual/",
                        }] : []),
                    ],
                },
            ],
        },
    ], [loginVisualUnlocked, t]);
    return (
    // collapsible="icon" means the sidebar can be collapsed to icon-only mode
    <Sidebar collapsible="icon" {...props}>
      {isCollapsedDesktop && collapsedMenu ? (<CollapsedSidebarSubmenu current={current} menu={collapsedMenu} normalize={normalize} onClose={closeCollapsedMenu} onCloseImmediately={closeCollapsedMenuImmediately} onKeepOpen={clearCollapsedMenuCloseTimer}/>) : null}
      <SidebarHeader className="border-b-0 min-h-12 justify-center py-1">
        {/* Preserve the logo's vertical anchor while the shell width transitions. */}
        <Link href="/overview/" prefetch={sidebarLinkPrefetch} className="flex h-8 w-full gap-2 items-center min-w-0 overflow-hidden rounded-lg px-2.5 transition-colors duration-200 ease-linear hover:bg-sidebar-accent hover:text-sidebar-accent-foreground group-data-[collapsible=icon]:size-8 group-data-[collapsible=icon]:gap-0 group-data-[collapsible=icon]:p-0">
          <div className="flex size-8 shrink-0 items-center justify-center">
            <LunaFoxMark className="size-7 shrink-0" decorative={false}/>
          </div>
          <div className="flex gap-2 items-center min-w-0 overflow-hidden transition-opacity duration-200 ease-linear group-data-[collapsible=icon]:opacity-0">
            <span className="font-semibold text-base tracking-tight truncate">
              {t("appName")}
            </span>
            <span className={cn("border border-sidebar-border px-1 py-0 rounded-sm text-sidebar-foreground/60", textRole.badge)}>
              {t("communityEditionAbbreviation")}
            </span>
          </div>
        </Link>
      </SidebarHeader>

      {/* Sidebar main content area */}
      <SidebarContent>
        {navGroups.map((group, index) => (<AppSidebarNavigationGroup collapsedMenuKey={collapsedMenu?.key} current={current} group={group} isCollapsedDesktop={isCollapsedDesktop} key={group.label ?? `top-navigation-${index}`} normalize={normalize} onCollapsedMenuClose={closeCollapsedMenu} onCollapsedMenuOpen={openCollapsedMenu}/>))}
      </SidebarContent>

      {/* Sidebar footer */}
      <SidebarFooter>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              isActive={current === "/settings/support/"}
              tooltip={t('supportAuthor')}
              render={<Link href="/settings/support/" prefetch={sidebarLinkPrefetch}/>}
            >
              <SupportAuthorIcon />
              <SidebarNavigationPendingIndicator />
              <span>{t('supportAuthor')}</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
          <SidebarMenuItem>
            <AboutDialog>
              <SidebarMenuButton tooltip={t('about')}>
                <AboutIcon />
                <span>{t('about')}</span>
              </SidebarMenuButton>
            </AboutDialog>
          </SidebarMenuItem>
        </SidebarMenu>
        {warmup ? <NavUserSkeleton /> : <NavUser user={user}/>}
      </SidebarFooter>
    </Sidebar>);
}
