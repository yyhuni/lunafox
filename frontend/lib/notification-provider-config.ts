import type { NotificationDestinationProvider } from "@/types/notification-settings.types"

export type NotificationProviderBrand = NotificationDestinationProvider

const NOTIFICATION_PROVIDER_ICON_CLASSNAMES: Record<NotificationProviderBrand, string> = {
  discord: "text-[var(--brand-discord)]",
  wecom: "text-[var(--brand-wecom)]",
  feishu: "text-[var(--brand-feishu)]",
}

export function getNotificationProviderIconClass(provider: NotificationProviderBrand): string {
  return NOTIFICATION_PROVIDER_ICON_CLASSNAMES[provider]
}
