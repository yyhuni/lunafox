"use client";
import dynamic from "next/dynamic";
import { DropdownMenuRadioGroup, DropdownMenuRadioItem } from "@/components/ui/dropdown-menu";
import { Separator } from "@/components/ui/separator";
import { SidebarTrigger } from "@/components/ui/sidebar";
import { LanguageSwitcher } from "@/components/language-switcher";
import { GithubStarButton } from "@/components/github-star-button";
import { ServerResourcePopover } from "@/components/server-resource-popover";
import { McpAccessPopover } from "@/components/mcp-access-popover";
import { HeaderIconActionMenu } from "@/components/shared/dropdown-menu-owners";
import { useTranslations } from "next-intl";
import { IconCircleHalf2, IconMoon, IconSun } from "@/components/icons";
import { useColorTheme } from "@/hooks/use-color-theme";
import type { ThemeModeId } from "@/lib/color-themes";
const HeaderIconPlaceholder = () => (<div className="h-8 w-8" aria-hidden="true"/>);
const HeaderActionSeparator = () => (<Separator orientation="vertical" className="mx-2 h-4! w-px self-center bg-muted-foreground/35"/>);
const QuickScanHeaderTrigger = dynamic(() => import("@/components/scan/quick-scan-header-trigger").then((mod) => mod.QuickScanHeaderTrigger), {
    ssr: false,
    loading: () => <HeaderIconPlaceholder />,
});
const NotificationDrawer = dynamic(() => import("@/components/notifications/notification-drawer").then((mod) => mod.NotificationDrawer), {
    ssr: false,
    loading: () => <HeaderIconPlaceholder />,
});
/**
 * Unified top bar component
 * Contains sidebar trigger and shortcut action buttons.
 * The product brand block lives in the sidebar header.
 */
export function UnifiedHeader({ warmup = false }: {
    warmup?: boolean;
}) {
    if (warmup) {
        return <UnifiedHeaderWarmup />;
    }
    return <UnifiedHeaderInteractive />;
}
function UnifiedHeaderWarmup() {
    return (<header data-slot="unified-header" className="bg-card border-b border-border flex h-(--header-height) items-center shrink-0">
      <div className="flex flex-1 gap-0.5 items-center md:gap-1 md:px-3 min-w-0 px-2">
        <HeaderIconPlaceholder />
        <div className="flex gap-0.5 items-center md:gap-1 ml-auto shrink-0">
          {Array.from({ length: 7 }).map((_, index) => (<HeaderIconPlaceholder key={index}/>))}
        </div>
      </div>
    </header>);
}
function UnifiedHeaderInteractive() {
    const t = useTranslations("navigation");
    const { mode, setMode } = useColorTheme();
    const themeModeOptions = [
        { id: "system" as const, label: t("themeModeSystem"), Icon: IconCircleHalf2 },
        { id: "light" as const, label: t("themeModeLight"), Icon: IconSun },
        { id: "dark" as const, label: t("themeModeDark"), Icon: IconMoon },
    ];
    const activeThemeMode = themeModeOptions.find((option) => option.id === mode) ?? themeModeOptions[0];
    const themeMenuLabel = t("themeMenu");
    const ActiveThemeModeIcon = activeThemeMode.Icon;
    const handleThemeModeChange = (value: string) => {
        setMode(value as ThemeModeId);
    };
    return (<header data-slot="unified-header" className="bg-card border-b border-border flex h-(--header-height) items-center shrink-0">
      <div className="flex flex-1 gap-0.5 items-center md:gap-1 md:px-3 min-w-0 px-2">
        <SidebarTrigger className="size-8"/>

        {/* Right button area */}
        <div className="flex gap-0.5 items-center md:gap-1 ml-auto shrink-0">
          <QuickScanHeaderTrigger />
          <ServerResourcePopover />
          <NotificationDrawer />
          <McpAccessPopover />
          <LanguageSwitcher />
          <HeaderIconActionMenu
            ariaLabel={themeMenuLabel}
            icon={<ActiveThemeModeIcon className="h-4 w-4" aria-hidden="true" />}
          >
            <DropdownMenuRadioGroup value={mode} onValueChange={handleThemeModeChange}>
              {themeModeOptions.map(({ id, label, Icon }) => (
                <DropdownMenuRadioItem key={id} value={id}>
                  <Icon className="h-4 w-4" aria-hidden="true" />
                  <span>{label}</span>
                </DropdownMenuRadioItem>
              ))}
            </DropdownMenuRadioGroup>
          </HeaderIconActionMenu>
          <HeaderActionSeparator />
          <GithubStarButton />
        </div>
      </div>
    </header>);
}
