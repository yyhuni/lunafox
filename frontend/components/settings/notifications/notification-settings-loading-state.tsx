"use client"

import { useTranslations } from "next-intl"

import {
  IconAlertTriangle,
  IconBrandDiscord,
  IconBrandSlack,
  IconEye,
  IconPlayerPlay,
  semanticIcons,
} from "@/components/icons"
import { PageHeader } from "@/components/common/page-header"
import { ActionSkeleton } from "@/components/shared/loading/action-skeleton"
import {
  getLoadingOwnerAttributes,
  getLoadingStructureSlotAttributes,
} from "@/components/shared/loading/loading-owner"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Checkbox } from "@/components/ui/checkbox"
import { Collapsible, CollapsibleContent } from "@/components/ui/collapsible"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Skeleton } from "@/components/ui/skeleton"
import { Switch } from "@/components/ui/switch"
import {
  getNotificationProviderIconClass,
  getNotificationProviderIconSurfaceClass,
} from "@/lib/notification-provider-config"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import { EXTERNALLY_DELIVERABLE_NOTIFICATION_KINDS } from "@/types/notification.types"
import {
  NOTIFICATION_DESTINATION_PROVIDERS,
  type NotificationDestination,
  type NotificationDestinationProvider,
} from "@/types/notification-settings.types"

import {
  NOTIFICATION_CHANNEL_EDITOR_CARD_CLASS,
  NOTIFICATION_CHANNEL_DETAIL_CONTENT_CLASS,
  NOTIFICATION_CHANNEL_DETAIL_HEADER_CLASS,
  NOTIFICATION_CHANNEL_DETAIL_HEADER_LAYOUT_CLASS,
  NOTIFICATION_CHANNEL_DETAIL_TITLE_CLASS,
  NOTIFICATION_CHANNEL_DETAIL_TITLE_ROW_CLASS,
  NOTIFICATION_CREDENTIAL_FIELD_CLASS,
  NOTIFICATION_CREDENTIAL_INPUT_CLASS,
  NOTIFICATION_CREDENTIAL_INPUT_SHELL_CLASS,
  NOTIFICATION_CREDENTIAL_REVEAL_ACTION_CLASS,
  NOTIFICATION_CHANNEL_ICON_CLASS,
  NOTIFICATION_CHANNELS_STACK_CLASS,
  NOTIFICATION_SETTINGS_CHANNEL_WORKBENCH_CLASS,
  NOTIFICATION_SETTINGS_CONTENT_SHELL_CLASS,
  NOTIFICATION_SETTINGS_PAGE_SHELL_CLASS,
  NOTIFICATION_SETTINGS_ACTION_BAR_CLASS,
  NOTIFICATION_SUBSCRIPTION_ITEM_CLASS,
  NOTIFICATION_SUBSCRIPTIONS_LIST_CLASS,
  NOTIFICATION_SUBSCRIPTIONS_SECTION_CLASS,
} from "./notification-settings-layout"

export interface NotificationSettingsPageLoadingStateProps {
  pageTitle: string
  pageDescription: string
  destinations?: readonly Pick<NotificationDestination, "provider" | "enabled" | "requiresWebhookUpdate">[]
  supportedKindCount?: number
  owner?: string
  className?: string
}

const DESTINATION_LOADING_ICONS: Record<NotificationDestinationProvider, typeof IconBrandDiscord> = {
  discord: IconBrandDiscord,
  wecom: IconBrandSlack,
  feishu: semanticIcons.concept.notification,
}

const DEFAULT_LOADING_DESTINATIONS: readonly Pick<NotificationDestination, "provider" | "enabled" | "requiresWebhookUpdate">[] = [
  { provider: "discord", enabled: true, requiresWebhookUpdate: false },
  { provider: "wecom", enabled: false, requiresWebhookUpdate: false },
  { provider: "feishu", enabled: false, requiresWebhookUpdate: false },
]

function NotificationDestinationEditorLoadingState({
  provider,
  Icon,
  isExpanded,
  requiresWebhookUpdate,
  supportedKindCount,
}: {
  provider: NotificationDestinationProvider
  Icon: typeof IconBrandDiscord
  isExpanded: boolean
  requiresWebhookUpdate: boolean
  supportedKindCount: number
}) {
  const t = useTranslations("settings.notifications")
  const providerTitle = t(`${provider}.title`)

  return (
    <Card className={NOTIFICATION_CHANNEL_EDITOR_CARD_CLASS}>
      <Collapsible open={isExpanded} className="flex min-w-0 flex-1 flex-col">
        <CardHeader className={NOTIFICATION_CHANNEL_DETAIL_HEADER_CLASS}>
          <div className={NOTIFICATION_CHANNEL_DETAIL_HEADER_LAYOUT_CLASS}>
            <div className="min-w-0">
              <div className={NOTIFICATION_CHANNEL_DETAIL_TITLE_ROW_CLASS}>
                <span className={cn(NOTIFICATION_CHANNEL_ICON_CLASS, getNotificationProviderIconSurfaceClass(provider))}>
                  <Icon aria-hidden="true" className={cn("size-5", getNotificationProviderIconClass(provider))} />
                </span>
                <CardTitle className={NOTIFICATION_CHANNEL_DETAIL_TITLE_CLASS}>{providerTitle}</CardTitle>
                <Badge variant={isExpanded ? "success" : "outline"}>
                  {isExpanded ? t("status.enabled") : t("status.disabled")}
                </Badge>
              </div>
              <CardDescription>{t(`${provider}.description`)}</CardDescription>
            </div>
            <Switch checked={isExpanded} disabled aria-label={t("destination.enabledLabel", { provider: providerTitle })} />
          </div>
        </CardHeader>
        <CollapsibleContent className="min-h-0 flex-1">
          <CardContent className={NOTIFICATION_CHANNEL_DETAIL_CONTENT_CLASS}>
            {requiresWebhookUpdate ? (
              <Alert className="border-warning/20 bg-warning/10 text-foreground">
                <IconAlertTriangle aria-hidden="true" className="text-warning" />
                <AlertDescription>{t("remediation.required")}</AlertDescription>
              </Alert>
            ) : null}
            <div className={NOTIFICATION_CREDENTIAL_FIELD_CLASS}>
              <Label>{t("destination.credentialLabel")}</Label>
              <div className={NOTIFICATION_CREDENTIAL_INPUT_SHELL_CLASS}>
                <Input
                  className={NOTIFICATION_CREDENTIAL_INPUT_CLASS}
                  disabled
                  placeholder={t(`${provider}.credentialPlaceholder`)}
                  aria-label={t("destination.credentialLabel")}
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className={NOTIFICATION_CREDENTIAL_REVEAL_ACTION_CLASS}
                  aria-label={t("destination.showCredential")}
                  disabled
                >
                  <IconEye className="size-4" />
                </Button>
              </div>
              <p className={textRole.helperText}>{t("destination.credentialHelp")}</p>
            </div>
            <div className={NOTIFICATION_SUBSCRIPTIONS_SECTION_CLASS}>
              <Label>{t("subscriptions.title")}</Label>
              <p className={textRole.helperText}>{t("subscriptions.description")}</p>
              <div className={NOTIFICATION_SUBSCRIPTIONS_LIST_CLASS}>
                {Array.from({ length: supportedKindCount }, (_, index) => (
                  <div key={index} className={NOTIFICATION_SUBSCRIPTION_ITEM_CLASS}>
                    <Checkbox disabled />
                    <div aria-hidden="true" className={textRole.bodySubtle}>
                      <Skeleton className="inline-block h-5 w-32 align-middle" />
                    </div>
                  </div>
                ))}
              </div>
            </div>
            <div className="flex items-center gap-3">
              <Button type="button" variant="outline" disabled>
                <IconPlayerPlay className="size-4" />
                {t("test.action")}
              </Button>
            </div>
          </CardContent>
        </CollapsibleContent>
      </Collapsible>
    </Card>
  )
}

export function NotificationSettingsPageLoadingState({
  pageTitle,
  pageDescription,
  destinations,
  supportedKindCount = EXTERNALLY_DELIVERABLE_NOTIFICATION_KINDS.length,
  owner,
  className,
}: NotificationSettingsPageLoadingStateProps) {
  if (owner !== undefined && !owner.trim()) {
    throw new Error("NotificationSettingsPageLoadingState requires a non-empty owner.")
  }

  const destinationByProvider = new Map((destinations ?? DEFAULT_LOADING_DESTINATIONS).map((destination) => [
    destination.provider,
    destination,
  ]))

  return (
    <div
      {...(owner ? getLoadingOwnerAttributes({ owner, layer: "route", intent: "route" }) : {})}
      data-slot="notification-settings-page-loading-state"
      className={cn(NOTIFICATION_SETTINGS_PAGE_SHELL_CLASS, className)}
    >
      <header {...getLoadingStructureSlotAttributes("notification-settings-header")}>
        <PageHeader code="NTF-01" title={pageTitle} description={pageDescription} />
      </header>

      <div className={NOTIFICATION_SETTINGS_CONTENT_SHELL_CLASS}>
        <div
          {...getLoadingStructureSlotAttributes("notification-settings-channel-workbench")}
          className={NOTIFICATION_SETTINGS_CHANNEL_WORKBENCH_CLASS}
        >
          <div className={NOTIFICATION_CHANNELS_STACK_CLASS}>
            {NOTIFICATION_DESTINATION_PROVIDERS.map((provider) => {
              const destination = destinationByProvider.get(provider)
              return (
                <NotificationDestinationEditorLoadingState
                  key={provider}
                  provider={provider}
                  Icon={DESTINATION_LOADING_ICONS[provider]}
                  isExpanded={destination?.enabled ?? false}
                  requiresWebhookUpdate={destination?.requiresWebhookUpdate ?? false}
                  supportedKindCount={supportedKindCount}
                />
              )
            })}
            <div className={NOTIFICATION_SETTINGS_ACTION_BAR_CLASS}>
              <ActionSkeleton widthClassName="w-28" />
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
