/**
 * Notification type definitions
 */

// Notification type enum (corresponds to backend NotificationCategory)
export type NotificationType = "vulnerability" | "scan" | "asset" | "system"

// Severity level (corresponds to backend NotificationLevel)
export type NotificationSeverity = "low" | "medium" | "high" | "critical"

// Backend notification level (consistent with backend)
export type BackendNotificationLevel = NotificationSeverity

// Backend notification data format
export interface BackendNotification {
  id: number
  category?: NotificationType
  title: string
  message: string
  level: BackendNotificationLevel
  created_at?: string
  createdAt?: string
  read_at?: string | null
  readAt?: string | null
  is_read?: boolean
  isRead?: boolean
}

// Notification interface
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

// Get notifications list request parameters
export interface GetNotificationsRequest {
  page?: number
  pageSize?: number
  type?: NotificationType
  unread?: boolean
}

// Get notifications list response
export interface GetNotificationsResponse {
  results: BackendNotification[]
  total: number
  page: number
  pageSize: number      // Backend returns camelCase format
  totalPages: number    // Backend returns camelCase format
  // Compatibility fields (backward compatible)
  page_size?: number
  total_pages?: number
}
