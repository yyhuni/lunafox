import type { NotificationDestinationProvider } from "@/types/notification-settings.types"

export type NotificationProviderBrand = NotificationDestinationProvider

const NOTIFICATION_PROVIDER_ICON_SURFACE_CLASSNAMES: Record<NotificationProviderBrand, string> = {
  discord: "bg-[var(--brand-discord)]/10",
  wecom: "bg-[var(--brand-wecom)]/10",
  feishu: "bg-[var(--brand-feishu)]/10",
}

const NOTIFICATION_PROVIDER_ICON_CLASSNAMES: Record<NotificationProviderBrand, string> = {
  discord: "text-[var(--brand-discord)]",
  wecom: "text-[var(--brand-wecom)]",
  feishu: "text-[var(--brand-feishu)]",
}

export function getNotificationProviderIconSurfaceClass(provider: NotificationProviderBrand): string {
  return NOTIFICATION_PROVIDER_ICON_SURFACE_CLASSNAMES[provider]
}

export function getNotificationProviderIconClass(provider: NotificationProviderBrand): string {
  return NOTIFICATION_PROVIDER_ICON_CLASSNAMES[provider]
}
