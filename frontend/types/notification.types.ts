/**
 * Notification type definitions
 */

export type NotificationType = "vulnerability" | "scan" | "asset" | "system"

export type NotificationSeverity = "low" | "medium" | "high" | "critical"

export type BackendNotificationLevel = NotificationSeverity

export interface BackendNotification {
  id: number
  category?: NotificationType
  title: string
  message: string
  level: BackendNotificationLevel
  createdAt?: string
  readAt?: string | null
  isRead?: boolean
}

export interface Notification {
  id: number
  type: NotificationType
  title: string
  description: string
  detail?: string
  time: string
  unread: boolean
  severity?: NotificationSeverity
  createdAt?: string
}

export interface GetNotificationsRequest {
  page?: number
  pageSize?: number
  type?: NotificationType
  unread?: boolean
}

export interface GetNotificationsResponse {
  results: BackendNotification[]
  total: number
  page: number
  pageSize: number
  totalPages: number
}
