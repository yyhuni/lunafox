/**
 * Durable inbox transport types. Display strings are immutable Server-rendered
 * snapshots; the frontend must never reconstruct them from a notification kind.
 */

export const NOTIFICATION_KINDS = [
  "scan-succeeded",
  "scan-failed",
  "vulnerability-observed",
  "agent-offline",
  "nuclei-poc-sync-succeeded",
  "nuclei-poc-sync-failed",
] as const

export type NotificationKind = (typeof NOTIFICATION_KINDS)[number]

export const EXTERNALLY_DELIVERABLE_NOTIFICATION_KINDS = [
  "scan-succeeded",
  "scan-failed",
  "vulnerability-observed",
  "agent-offline",
] as const

export type ExternalNotificationKind = (typeof EXTERNALLY_DELIVERABLE_NOTIFICATION_KINDS)[number]

export type NotificationCategory = "scan" | "vulnerability" | "system"

export type NotificationPriority = "normal" | "high" | "critical"

export interface NotificationInboxItem {
  name: string
  kind: NotificationKind
  category: NotificationCategory
  priority: NotificationPriority
  subject: string
  title: string
  message: string
  occurredAt: string
  createdAt: string
  readAt?: string | null
}

export interface ListNotificationsRequest {
  pageSize?: number
  pageToken?: string
}

export interface ListNotificationsResponse {
  results: NotificationInboxItem[]
  nextPageToken: string
  totalSize: number
}

export interface NotificationUnreadCount {
  unreadCount: number
}
