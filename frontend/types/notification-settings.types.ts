import type { ExternalNotificationKind } from "@/types/notification.types"

export type NotificationDestinationProvider = "discord" | "wecom" | "feishu"

export const NOTIFICATION_DESTINATION_PROVIDERS = ["discord", "wecom", "feishu"] as const satisfies readonly NotificationDestinationProvider[]

export interface NotificationLocale {
  locale: "zh" | "en"
}

export interface NotificationDestination {
  provider: NotificationDestinationProvider
  credential: string
  enabled: boolean
  subscriptions: ExternalNotificationKind[]
  requiresWebhookUpdate: boolean
}

export interface NotificationDestinationListResponse {
  results: NotificationDestination[]
  supportedKinds: ExternalNotificationKind[]
}

export type UpdateNotificationDestinationRequest = Pick<
  NotificationDestination,
  "credential" | "enabled" | "subscriptions"
>

export type NotificationDestinationTestResult =
  | "delivered"
  | "invalid_credential"
  | "connectivity_failure"
  | "provider_rejected"
  | "internal_failure"
  // The real Server never returns this value. It models the explicit mock
  // transport response without making mock mode a UI environment branch.
  | "unavailable"

export interface TestNotificationDestinationRequest {
  credential: string
}

export interface TestNotificationDestinationResponse {
  result: NotificationDestinationTestResult
}
