"use client"

import dynamic from "next/dynamic"
import { SidebarTrigger } from "@/components/ui/sidebar"
import { LanguageSwitcher } from "@/components/language-switcher"
import { Link } from "@/i18n/navigation"
import { useTranslations } from "next-intl"

const HeaderIconPlaceholder = () => (
  <div className="h-8 w-8" aria-hidden="true" />
)

const HeaderButtonPlaceholder = () => (
  <div className="h-8 w-24" aria-hidden="true" />
)

const QuickScanDialog = dynamic(
  () => import("@/components/scan/quick-scan-dialog").then((mod) => mod.QuickScanDialog),
  {
    ssr: false,
    loading: () => <HeaderButtonPlaceholder />,
  }
)

const NotificationDrawer = dynamic(
  () => import("@/components/notifications/notification-drawer").then((mod) => mod.NotificationDrawer),
  {
    ssr: false,
    loading: () => <HeaderIconPlaceholder />,
  }
)

/**
 * Unified top bar component
 * Contains logo, sidebar triggers, shortcut action buttons
 * Across the entire width of the page, the logo is on the far left
 */
export function UnifiedHeader() {
  const t = useTranslations("navigation")
  const logoSrc = "/images/icon-64.png"

  return (
    <header
      data-slot="unified-header"
      className="flex h-(--header-height) shrink-0 items-center border-b bg-background"
    >
      {/* Logo area - fixed width, consistent with sidebar width */}
      <div className="flex h-full w-(--sidebar-width) shrink-0 items-center px-4">
        <Link href="/" className="flex items-center gap-3">
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img src={logoSrc} alt="LunaFox Logo" className="size-7" width={28} height={28} />
          <div className="flex flex-col gap-0.5 leading-none">
            <div className="flex items-center gap-2">
              <span className="text-base font-semibold tracking-tight">
                {t("appName")}
              </span>
              <span className="rounded-sm border border-border px-1.5 py-0.5 text-[9px] uppercase tracking-[0.2em] text-muted-foreground">
                {process.env.NEXT_PUBLIC_IMAGE_TAG || "dev"}
              </span>
            </div>
            <div className="flex items-center gap-2 text-[10px] uppercase tracking-[0.24em] text-muted-foreground">
              <span className="h-1.5 w-1.5 rounded-full bg-[var(--success)]" />
              <span>Security Console</span>
            </div>
          </div>
        </Link>
      </div>

      {/* Right content area */}
      <div className="flex flex-1 items-center gap-2 px-4">
        {/* Sidebar trigger - mobile only */}
        <SidebarTrigger className="-ml-1 md:hidden" />

        {/* Right button area */}
        <div className="ml-auto flex items-center gap-2">
          <QuickScanDialog />
          <NotificationDrawer />
          <LanguageSwitcher />
        </div>
      </div>
    </header>
  )
}
