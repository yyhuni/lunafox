"use client"

import { NotificationDrawerLayout } from "./notification-drawer-sections"
import { useNotificationDrawerState } from "./notification-drawer-state"

/**
 * Notification drawer component
 * A side panel that slides out from the right, displaying detailed notification information
 */
export function NotificationDrawer() {
  const state = useNotificationDrawerState()
  return <NotificationDrawerLayout state={state} />
}
