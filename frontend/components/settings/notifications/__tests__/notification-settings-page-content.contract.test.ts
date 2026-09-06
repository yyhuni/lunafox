import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/settings/notifications/notification-settings-page-content.tsx"), "utf8")
const loadingSource = readFileSync(path.resolve(process.cwd(), "components/settings/notifications/notification-settings-loading-state.tsx"), "utf8")
const layoutSource = readFileSync(path.resolve(process.cwd(), "components/settings/notifications/notification-settings-layout.ts"), "utf8")

describe("notification-settings-page-content contract", () => {
  it("preserves the route-boundary content markers", () => {
    expect(source).toContain("export interface NotificationSettingsPageContentProps")
    expect(source).toContain("pageTitle: string")
    expect(source).toContain("pageDescription: string")
    expect(source).toContain("onReady?: () => void")
    expect(source).toContain("deferInitialSkeleton?: boolean")
    expect(source).toContain("export default function NotificationSettingsPageContent")
    expect(source).toContain("ContentHandoff")
    expect(source).toContain("if (isInitialLoading && deferInitialSkeleton)")
    expect(source).toContain('owner="notification-settings-page-content"')
    expect(source).toContain('from "./notification-settings-loading-state"')
    expect(loadingSource).toContain("export function NotificationSettingsPageLoadingState")
    expect(source).toContain("<NotificationSettingsPageLoadingState")
    expect(source).not.toContain("notification-settings-page-skeleton")
    expect(source).not.toContain("NotificationSettingsPageSkeleton")
  })

  it("uses shared provider icon owners", () => {
    expect(source).toContain('from "@/components/icons"')
    expect(source).toContain('from "@/lib/notification-provider-config"')
    expect(source).toContain("getNotificationProviderIconSurfaceClass")
    expect(source).toContain("getNotificationProviderIconClass")
    expect(source).not.toContain("#5865F2")
    expect(source).not.toContain("#07C160")
  })

  it("derives the resolved page and loading state from one channel workbench layout contract", () => {
    const sharedLayoutConstants = [
      "NOTIFICATION_SETTINGS_CHANNEL_WORKBENCH_CLASS",
      "NOTIFICATION_CHANNELS_STACK_CLASS",
      "NOTIFICATION_CHANNEL_EDITOR_CARD_CLASS",
      "NOTIFICATION_CHANNEL_DETAIL_HEADER_CLASS",
      "NOTIFICATION_CHANNEL_DETAIL_HEADER_LAYOUT_CLASS",
      "NOTIFICATION_CHANNEL_DETAIL_TITLE_ROW_CLASS",
      "NOTIFICATION_CHANNEL_DETAIL_TITLE_CLASS",
      "NOTIFICATION_CHANNEL_DETAIL_CONTENT_CLASS",
      "NOTIFICATION_CHANNEL_ICON_CLASS",
      "NOTIFICATION_CREDENTIAL_FIELD_CLASS",
      "NOTIFICATION_SUBSCRIPTIONS_SECTION_CLASS",
      "NOTIFICATION_SUBSCRIPTIONS_LIST_CLASS",
      "NOTIFICATION_SUBSCRIPTION_ITEM_CLASS",
    ]

    expect(source).toContain('from "./notification-settings-layout"')
    for (const constant of sharedLayoutConstants) {
      expect(layoutSource).toContain(`export const ${constant}`)
      expect(source).toContain(constant)
    }

    for (const constant of [
      "NOTIFICATION_SETTINGS_CHANNEL_WORKBENCH_CLASS",
      "NOTIFICATION_CHANNELS_STACK_CLASS",
      "NOTIFICATION_CHANNEL_EDITOR_CARD_CLASS",
      "NOTIFICATION_CHANNEL_DETAIL_HEADER_CLASS",
      "NOTIFICATION_CHANNEL_DETAIL_HEADER_LAYOUT_CLASS",
      "NOTIFICATION_CHANNEL_DETAIL_TITLE_ROW_CLASS",
      "NOTIFICATION_CHANNEL_DETAIL_TITLE_CLASS",
      "NOTIFICATION_CHANNEL_ICON_CLASS",
    ]) {
      expect(loadingSource).toContain(constant)
    }
  })

  it("keeps the full channel stack in the page scroll region instead of shrinking editors for the save action", () => {
    expect(layoutSource).toContain('NOTIFICATION_SETTINGS_CONTENT_SHELL_CLASS =\n  "flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto px-4 [scrollbar-gutter:stable] lg:px-6"')
    expect(layoutSource).toContain('NOTIFICATION_SETTINGS_CHANNEL_WORKBENCH_CLASS = "flex w-full min-w-0"')
    expect(layoutSource).toContain('NOTIFICATION_CHANNELS_STACK_CLASS = "flex w-full min-w-0 flex-col gap-4"')
    expect(layoutSource).toContain('NOTIFICATION_CHANNEL_EDITOR_CARD_CLASS =\n  "flex min-w-0 shrink-0 flex-col gap-0 overflow-hidden py-0"')
    expect(layoutSource).toContain('NOTIFICATION_CHANNEL_DETAIL_CONTENT_CLASS = "space-y-5 py-6"')
    expect(layoutSource).toContain('NOTIFICATION_SETTINGS_ACTION_BAR_CLASS = "flex shrink-0 items-center justify-end gap-3"')
  })

  it("models the fixed Discord, WeCom, and Feishu destinations without inbox preference transport", () => {
    expect(source).toContain('NOTIFICATION_DESTINATION_PROVIDERS')
    expect(source).toContain('from "@/types/notification-settings.types"')
    expect(source).toContain("function NotificationDestinationEditor")
    expect(source).toContain("<Collapsible open={draft.enabled}")
    expect(source).toContain("<CollapsibleContent")
    expect(source).toContain("const dirtyProviders")
    expect(source).toContain("Promise.allSettled")
    expect(source).toContain("useTestNotificationDestination")
    expect(source).toContain('data-testid="notification-channel-editors"')
    expect(source).toContain('t("destination.saveAll")')
    expect(source).not.toContain('t("destination.save")')
    expect(source).toContain('data-testid="notification-channel-editors"')
    expect(source).toContain('data-testid={`notification-channel-editor-${provider}`}')
    expect(source).not.toContain("function NotificationChannelList")
    expect(source).not.toContain("selectedChannel")
    expect(source).not.toContain("notification-channel-list")
    expect(source).not.toContain("NotificationInboxEditor")
    expect(source).not.toContain("NotificationPreferences")
    expect(source).not.toContain("notificationPreferences")
    expect(source).not.toContain("useNotificationPreferences")
    expect(source).not.toContain("useUpdateNotificationPreferences")
    expect(source).not.toContain("PREFERENCE_ITEMS")
    expect(source).not.toContain("TabsContent")
    expect(source).not.toContain("TabsTrigger")
    expect(source).not.toContain("tabs.channels")
    expect(source).not.toContain("categories.")
    expect(loadingSource).toContain("NOTIFICATION_DESTINATION_PROVIDERS.map")
    expect(loadingSource).toContain("DESTINATION_LOADING_ICONS")
    expect(loadingSource).toContain("DEFAULT_LOADING_DESTINATIONS")
    expect(loadingSource).toContain("destinationByProvider")
    expect(loadingSource).toContain("<Collapsible open={isExpanded}")
    expect(loadingSource).toContain("supportedKindCount")
    expect(loadingSource).toContain('<div aria-hidden="true" className={textRole.bodySubtle}>')
    expect(loadingSource).toContain('<Skeleton className="inline-block h-5 w-32 align-middle" />')
    expect(loadingSource).toContain("feishu: semanticIcons.concept.notification")
    expect(loadingSource).not.toContain('provider: "inbox"')
  })

  it("fails visibly when the fixed external destination contract omits a provider", () => {
    expect(source).toContain("const hasAllDestinationProviders")
    expect(source).toContain("const destinationContractError")
    expect(source).toContain("Notification destinations response omitted a required provider.")
    expect(source).toContain("const settingsError")
  })

  it("pairs the route-visible header and channel workbench across loading and resolved content", () => {
    for (const slot of [
      "notification-settings-header",
      "notification-settings-channel-workbench",
    ]) {
      expect(source).toContain(`getLoadingStructureSlotAttributes("${slot}")`)
      expect(loadingSource).toContain(`getLoadingStructureSlotAttributes("${slot}")`)
    }

    expect(source).not.toContain("notification-settings-tab-rail")
    expect(loadingSource).not.toContain("notification-settings-tab-rail")
    expect(source).not.toContain("notification-settings-preference-section")
    expect(loadingSource).not.toContain("notification-settings-preference-section")
  })

  it("keeps the component-owned loading state free of resolved data dependencies", () => {
    expect(loadingSource).not.toContain("react-hook-form")
    expect(loadingSource).not.toContain("use-notification-settings")
  })

  it("keeps unsaved drafts memory-only and uses server-derived remediation", () => {
    expect(source).toContain('window.addEventListener("beforeunload", onBeforeUnload)')
    expect(source).toContain('window.removeEventListener("beforeunload", onBeforeUnload)')
    expect(source).not.toContain("localStorage")
    expect(source).not.toContain("sessionStorage")
    expect(source).toContain("baseline.requiresWebhookUpdate")
    expect(source).not.toContain("new URL(")
  })

  it("keeps resolved and loading channel headers on the balanced spacing path", () => {
    expect(layoutSource).toContain('NOTIFICATION_CHANNEL_DETAIL_HEADER_CLASS = "border-b pt-6"')
    expect(source).toContain('<CardHeader className={NOTIFICATION_CHANNEL_DETAIL_HEADER_CLASS}>')
    expect(loadingSource).toContain('<CardHeader className={NOTIFICATION_CHANNEL_DETAIL_HEADER_CLASS}>')
    expect(source).not.toContain('density="compact"')
    expect(loadingSource).not.toContain('density="compact"')
  })
})
