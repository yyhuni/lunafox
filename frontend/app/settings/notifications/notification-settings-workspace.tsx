"use client"

import { useTranslations } from "next-intl"
import { HiddenReadinessRouteBoundary } from "@/components/shared/loading/hidden-readiness-route-boundary"
import NotificationSettingsPageContent from "@/components/settings/notifications/notification-settings-page-content"
import { NotificationSettingsPageLoadingState } from "@/components/settings/notifications/notification-settings-loading-state"
import { useNotificationDestinations } from "@/hooks/use-notification-settings"
import {
  NOTIFICATION_SETTINGS_WORKSPACE_HANDOFF_CLASS,
  NOTIFICATION_SETTINGS_WORKSPACE_STATE_CLASS,
} from "@/components/settings/notifications/notification-settings-layout"

export function NotificationSettingsWorkspace() {
  const t = useTranslations("settings.notifications")
  const destinationsQuery = useNotificationDestinations()
  const pageTitle = t("pageTitle")
  const pageDescription = t("pageDesc")

  return (
    <HiddenReadinessRouteBoundary
      owner="notification-settings-page-route"
      layer="workspace"
      intent="data"
      skeleton={(
        <NotificationSettingsPageLoadingState
          pageTitle={pageTitle}
          pageDescription={pageDescription}
          destinations={destinationsQuery.data?.results}
          supportedKindCount={destinationsQuery.data?.supportedKinds.length}
        />
      )}
      className={NOTIFICATION_SETTINGS_WORKSPACE_HANDOFF_CLASS}
      skeletonClassName={NOTIFICATION_SETTINGS_WORKSPACE_STATE_CLASS}
      contentClassName={NOTIFICATION_SETTINGS_WORKSPACE_STATE_CLASS}
    >
      {({ onReady, deferInitialSkeleton }) => (
        <NotificationSettingsPageContent
          pageTitle={pageTitle}
          pageDescription={pageDescription}
          onReady={onReady}
          deferInitialSkeleton={deferInitialSkeleton}
        />
      )}
    </HiddenReadinessRouteBoundary>
  )
}
