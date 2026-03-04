"use client" // Mark as client component, can use browser APIs and interactive features

// Import React library
import React from "react"
import dynamic from "next/dynamic"
// Import various icons from Tabler Icons library
import {
  IconDashboard, // Dashboard icon
  IconListDetails, // List details icon
  IconSettings, // Settings icon
  IconUsers, // Users icon
  IconChevronRight, // Right arrow icon
  IconRadar, // Radar scan icon
  IconTool, // Tool icon
  IconServer, // Server icon
  IconDatabase, // Database icon
  IconTerminal2, // Terminal icon
  IconBug, // Vulnerability icon
  IconSearch, // Search icon
  IconKey, // API Key icon
  IconBan, // Blacklist icon
  IconInfoCircle, // About icon
} from "@/components/icons"
// Import internationalization hook
import { useTranslations } from 'next-intl'
// Import internationalization navigation components
import { Link, usePathname } from '@/i18n/navigation'

// Import custom navigation components
import { NavSystem } from "@/components/nav-system"
import { NavUser } from "@/components/nav-user"
import { AboutDialog } from "@/components/about-dialog"
// Import sidebar UI components
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
  SidebarTrigger,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
} from "@/components/ui/sidebar"
// Import collapsible component
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"

const AppSidebarBackgroundTasks = dynamic(
  () => import("@/components/app-sidebar-background-tasks").then((mod) => mod.AppSidebarBackgroundTasks),
  { ssr: false }
)

/**
 * Application sidebar component
 * Displays the main navigation menu of the application, including user info, main menu, documents and secondary menu
 * Supports expand and collapse functionality for submenus
 * @param props - All properties of the Sidebar component
 */
export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const t = useTranslations('navigation')
  const pathname = usePathname()
  const normalize = (p: string) => (p !== "/" && p.endsWith("/") ? p.slice(0, -1) : p)
  const current = normalize(pathname)

  const handleNavClick = React.useCallback(() => {
    if (typeof window !== "undefined") {
      window.dispatchEvent(new Event("lunafox:route-progress-start"))
    }
  }, [])
  const showDevTools = process.env.NODE_ENV === "development"

  const user = React.useMemo(
    () => ({
      name: "admin",
      email: "admin@admin.com",
      avatar: "/images/icon-64.png",
    }),
    []
  )

  const navMain = React.useMemo(
    () => [
      {
        title: t('dashboard'),
        url: "/dashboard/",
        icon: IconDashboard,
      },
      {
        title: t('search'),
        url: "/search/",
        icon: IconSearch,
      },
      {
        title: t('organization'),
        url: "/organization/",
        icon: IconUsers,
      },
      {
        title: t('target'),
        url: "/target/",
        icon: IconListDetails,
      },
      {
        title: t('vulnerabilities'),
        url: "/vulnerabilities/",
        icon: IconBug,
      },
      {
        title: t('scan'),
        url: "/scan/",
        icon: IconRadar,
        items: [
          {
            title: t('scanHistory'),
            url: "/scan/history/",
          },
          {
            title: t('scheduledScan'),
            url: "/scan/scheduled/",
          },
          {
            title: t('scanEngine'),
            url: "/scan/workflow/",
          },
        ],
      },
      {
        title: t('tools'),
        url: "/tools/",
        icon: IconTool,
        items: [
          {
            title: t('wordlists'),
            url: "/tools/wordlists/",
          },
          {
            title: t('fingerprints'),
            url: "/tools/fingerprints/",
          },
          {
            title: t('nucleiTemplates'),
            url: "/tools/nuclei/",
          },
        ],
      },
    ],
    [t]
  )

  const documents = React.useMemo(
    () => [
      {
        name: t('workers'),
        url: "/settings/workers/",
        icon: IconServer,
      },
      {
        name: t('systemLogs'),
        url: "/settings/system-logs/",
        icon: IconTerminal2,
      },
      {
        name: t('databaseHealth'),
        url: "/settings/database-health/",
        icon: IconDatabase,
      },
      {
        name: t('notifications'),
        url: "/settings/notifications/",
        icon: IconSettings,
      },
      {
        name: t('apiKeys'),
        url: "/settings/api-keys/",
        icon: IconKey,
      },
      {
        name: t('globalBlacklist'),
        url: "/settings/blacklist/",
        icon: IconBan,
      },
    ],
    [t]
  )

  const devToolGroups = React.useMemo(() => [
    {
      title: "组件演示",
      icon: IconTool,
      items: [
        { title: "组件总览", url: "/tools/component-gallery/" },
        { title: "组件 Demo 索引", url: "/tools/component-demos/" },
        { title: "Agent Node Designs", url: "/prototypes/agent-node-designs/" },
        { title: "Vuln Visualization", url: "/prototypes/vuln-designs/" },
        { title: "Vuln Audit Mode (Left-Right)", url: "/prototypes/vuln-audit/" },
        { title: "Vuln Audit Mode (Top-Bottom)", url: "/prototypes/vuln-audit-vertical/" },
        { title: "Vuln Audit Mode (Drawer)", url: "/prototypes/vuln-audit-drawer/" },
        { title: "Button Demo", url: "/tools/button-demo/" },
        { title: "Badge Demo", url: "/tools/badge-demo/" },
        { title: "Badge Variants", url: "/prototypes/badge-variants/" },
        { title: "Status Progress Variants", url: "/prototypes/status-progress-variants/" },
        { title: "Asset Card Variants", url: "/prototypes/asset-card-variants/" },
      ],
    },
    {
      title: "布局与导航",
      icon: IconListDetails,
      items: [
        { title: "Demo A: Unified Card", url: "/prototypes/header-demo-a/" },
        { title: "Demo B: Ghost Header", url: "/prototypes/header-demo-b/" },
        { title: "Demo C: Docked UI", url: "/prototypes/header-demo-c/" },
        { title: "Sidebar Variants", url: "/prototypes/sidebar-variants/" },
        { title: "Nudge & Care Demo", url: "/tools/nudges/" },
        { title: "Nudge Design Variants", url: "/tools/nudges/design-variants/" },
      ],
    },
    {
      title: "仪表盘",
      icon: IconDashboard,
      items: [
        { title: "Dashboard Demo", url: "/prototypes/dashboard-demo/" },
        { title: "Dashboard Demo Dark", url: "/prototypes/dashboard-demo-dark/" },
        { title: "Dashboard Rework", url: "/prototypes/dashboard-rework/" },
        { title: "GitHub Buttons", url: "/tools/github-buttons/" },
        { title: "Asset Pulse Designs", url: "/prototypes/asset-pulse-designs/" },
        { title: "Asset Pulse Light", url: "/prototypes/asset-pulse-light-designs/" },
        { title: "Advanced Asset Pulse", url: "/prototypes/advanced-asset-pulse/" },
        { title: "Vuln Chart Designs", url: "/prototypes/vuln-designs/" },
      ],
    },
    {
      title: "流程与风格",
      icon: IconRadar,
      items: [
        { title: "Scan Workflow Designs", url: "/scan/workflow/demo/" },
        { title: "Scan Dialogs", url: "/prototypes/scan-dialogs/" },
        { title: "Scan Actions Demo", url: "/prototypes/scan-history-actions/" },
        { title: "Workers Page Layout", url: "/prototypes/workers-page/" },
        { title: "Arknights UI Demo", url: "/prototypes/arknights-ui/" },
      ],
    },
  ], [])

  return (
    // collapsible="icon" means the sidebar can be collapsed to icon-only mode
    <Sidebar collapsible="icon" {...props}>
      <AppSidebarBackgroundTasks />
      {/* Sidebar main content area - Logo has been moved to the top bar */}
      <SidebarContent>
        {/* Main navigation menu */}
        <SidebarGroup>
          <div className="flex h-8 items-center justify-between gap-2 px-2 group-data-[collapsible=icon]:justify-center group-data-[collapsible=icon]:px-0 group-data-[collapsible=icon]:gap-0">
            <span className="text-sidebar-foreground/70 ring-sidebar-ring text-xs font-medium transition-[max-width,opacity] duration-200 ease-linear whitespace-nowrap overflow-hidden max-w-full group-data-[collapsible=icon]:max-w-0 group-data-[collapsible=icon]:opacity-0">
              {t('mainFeatures')}
            </span>
            <SidebarTrigger className="size-8 text-muted-foreground hover:text-foreground [&>svg]:size-4" />
          </div>
          <SidebarGroupContent>
            <SidebarMenu>
              {navMain.map((item) => {
                const navUrl = normalize(item.url)
                const isActive = navUrl === "/" ? current === "/" : current === navUrl || current.startsWith(navUrl + "/")
                const hasSubItems = item.items && item.items.length > 0

                if (!hasSubItems) {
                  return (
                    <SidebarMenuItem key={item.title}>
                      <SidebarMenuButton asChild isActive={isActive}>
                        <Link href={item.url} onClick={handleNavClick}>
                          <item.icon />
                          <span>{item.title}</span>
                        </Link>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  )
                }

                return (
                  <Collapsible
                    key={item.title}
                    defaultOpen={isActive}
                    className="group/collapsible"
                  >
                    <SidebarMenuItem>
                      <CollapsibleTrigger asChild>
                        <SidebarMenuButton isActive={isActive}>
                          <item.icon />
                          <span>{item.title}</span>
                          <IconChevronRight className="ml-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90" />
                        </SidebarMenuButton>
                      </CollapsibleTrigger>
                      <CollapsibleContent>
                        <SidebarMenuSub>
                          {item.items?.map((subItem) => {
                            const subUrl = normalize(subItem.url)
                            const isSubActive = current === subUrl || current.startsWith(subUrl + "/")
                            return (
                              <SidebarMenuSubItem key={subItem.title}>
                                <SidebarMenuSubButton
                                  asChild
                                  isActive={isSubActive}
                                >
                                  <Link href={subItem.url} onClick={handleNavClick}>
                                    <span>{subItem.title}</span>
                                  </Link>
                                </SidebarMenuSubButton>
                              </SidebarMenuSubItem>
                            )
                          })}
                        </SidebarMenuSub>
                      </CollapsibleContent>
                    </SidebarMenuItem>
                  </Collapsible>
                )
              })}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        {/* System settings navigation menu */}
        <NavSystem items={documents} />
        {showDevTools && (
          <SidebarGroup>
            <SidebarGroupLabel>{t('devTools')}</SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                {devToolGroups.map((group) => {
                  const isGroupActive = group.items.some((item) => {
                    const devUrl = normalize(item.url)
                    return current === devUrl || current.startsWith(devUrl + "/")
                  })
                  return (
                    <Collapsible
                      key={group.title}
                      defaultOpen={isGroupActive}
                      className="group/collapsible"
                    >
                      <SidebarMenuItem>
                        <CollapsibleTrigger asChild>
                          <SidebarMenuButton isActive={isGroupActive}>
                            <group.icon />
                            <span>{group.title}</span>
                            <IconChevronRight className="ml-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90" />
                          </SidebarMenuButton>
                        </CollapsibleTrigger>
                        <CollapsibleContent>
                          <SidebarMenuSub>
                            {group.items.map((item) => {
                              const devUrl = normalize(item.url)
                              const isActive = current === devUrl || current.startsWith(devUrl + "/")
                              return (
                                <SidebarMenuSubItem key={item.title}>
                                  <SidebarMenuSubButton asChild isActive={isActive}>
                                    <Link href={item.url} onClick={handleNavClick}>
                                      <span>{item.title}</span>
                                    </Link>
                                  </SidebarMenuSubButton>
                                </SidebarMenuSubItem>
                              )
                            })}
                          </SidebarMenuSub>
                        </CollapsibleContent>
                      </SidebarMenuItem>
                    </Collapsible>
                  )
                })}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        )}
        {/* About system button */}
        <SidebarGroup className="mt-auto">
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuItem>
                <AboutDialog>
                  <SidebarMenuButton>
                    <IconInfoCircle />
                    <span>{t('about')}</span>
                  </SidebarMenuButton>
                </AboutDialog>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>

      {/* Sidebar footer */}
      <SidebarFooter>
        <NavUser user={user} />
      </SidebarFooter>
    </Sidebar>
  )
}
