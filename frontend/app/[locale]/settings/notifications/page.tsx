"use client"

import dynamic from "next/dynamic"

const NotificationSettingsPageContent = dynamic(
  () => import("@/components/settings/notifications/notification-settings-page-content"),
  {
    ssr: false,
    loading: () => (
      <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6 px-4 lg:px-6">
        <div className="h-9 w-64 animate-pulse rounded bg-muted" />
        <div className="h-32 w-full animate-pulse rounded bg-muted" />
      </div>
    ),
  }
)

export default function NotificationSettingsPage() {
  return <NotificationSettingsPageContent />
}
