import type {
  ListNotificationsRequest,
  ListNotificationsResponse,
  NotificationInboxItem,
  NotificationUnreadCount,
} from "@/types/notification.types"

const PAGE_TOKEN_PREFIX = "mock-notification-page"

const INITIAL_NOTIFICATIONS: readonly NotificationInboxItem[] = [
  {
    name: "users/current/notifications/1",
    kind: "vulnerability-observed",
    category: "vulnerability",
    priority: "critical",
    subject: "scans/42/vulnerabilities/1",
    title: "Critical vulnerability observed",
    message: "A critical vulnerability was observed during the RetailMax scan.",
    occurredAt: "2026-08-07T10:30:00.000Z",
    createdAt: "2026-08-07T10:30:05.000Z",
    readAt: null,
  },
  {
    name: "users/current/notifications/2",
    kind: "scan-succeeded",
    category: "scan",
    priority: "normal",
    subject: "scans/41",
    title: "Scan completed",
    message: "The Acme perimeter scan completed successfully.",
    occurredAt: "2026-08-07T09:00:00.000Z",
    createdAt: "2026-08-07T09:00:02.000Z",
    readAt: null,
  },
  {
    name: "users/current/notifications/3",
    kind: "scan-failed",
    category: "scan",
    priority: "high",
    subject: "scans/40",
    title: "Scan failed",
    message: "The Global Finance scan could not establish a connection before its deadline.",
    occurredAt: "2026-08-06T16:45:00.000Z",
    createdAt: "2026-08-06T16:45:04.000Z",
    readAt: "2026-08-06T17:10:00.000Z",
  },
  {
    name: "users/current/notifications/4",
    kind: "agent-offline",
    category: "system",
    priority: "high",
    subject: "agents/agent-03",
    title: "Agent is offline",
    message: "Agent agent-03 transitioned from online to offline.",
    occurredAt: "2026-08-05T08:30:00.000Z",
    createdAt: "2026-08-05T08:30:03.000Z",
    readAt: "2026-08-05T09:00:00.000Z",
  },
  {
    name: "users/current/notifications/5",
    kind: "nuclei-poc-sync-succeeded",
    category: "system",
    priority: "normal",
    subject: "nucleiPocSyncTasks/6dd1f0fd-5b34-4cbb-9c4f-3b9c6c0f6a12",
    title: "Nuclei POC sync completed",
    message: "Imported 184 Nuclei POCs from commit 0123456789ab.",
    occurredAt: "2026-08-04T12:00:00.000Z",
    createdAt: "2026-08-04T12:00:04.000Z",
    readAt: null,
  },
  {
    name: "users/current/notifications/6",
    kind: "nuclei-poc-sync-failed",
    category: "system",
    priority: "high",
    subject: "nucleiPocSyncTasks/2f7d3a99-90a5-42d1-a0d9-b7db9b40e1cd",
    title: "Nuclei POC sync failed",
    message: "Sync failed (TEMPLATE_INVALID): One or more Nuclei templates failed validation.",
    occurredAt: "2026-08-03T11:20:00.000Z",
    createdAt: "2026-08-03T11:20:03.000Z",
    readAt: "2026-08-03T12:00:00.000Z",
  },
] as const

export const mockNotifications: NotificationInboxItem[] = INITIAL_NOTIFICATIONS.map((item) => ({ ...item }))

function cloneNotification(notification: NotificationInboxItem): NotificationInboxItem {
  return { ...notification }
}

function pageFromToken(pageToken: string | undefined): number {
  if (!pageToken) {
    return 1
  }
  const match = new RegExp(`^${PAGE_TOKEN_PREFIX}-(\\d+)$`).exec(pageToken)
  return match ? Number.parseInt(match[1], 10) : 1
}

export function getMockNotifications(
  params: ListNotificationsRequest = {}
): ListNotificationsResponse {
  const pageSize = Math.max(1, Math.min(params.pageSize ?? 50, 100))
  const page = pageFromToken(params.pageToken)
  const start = (page - 1) * pageSize
  const nextPage = start + pageSize < mockNotifications.length ? page + 1 : undefined

  return {
    results: mockNotifications.slice(start, start + pageSize).map(cloneNotification),
    nextPageToken: nextPage ? `${PAGE_TOKEN_PREFIX}-${nextPage}` : "",
    totalSize: mockNotifications.length,
  }
}

export function getMockNotification(name: string): NotificationInboxItem | undefined {
  const notification = mockNotifications.find((item) => item.name === name)
  return notification ? cloneNotification(notification) : undefined
}

export function markMockNotificationRead(name: string): NotificationInboxItem | undefined {
  const notification = mockNotifications.find((item) => item.name === name)
  if (!notification) {
    return undefined
  }
  if (!notification.readAt) {
    notification.readAt = new Date().toISOString()
  }
  return cloneNotification(notification)
}

export function markAllMockNotificationsRead(): void {
  const readAt = new Date().toISOString()
  for (const notification of mockNotifications) {
    if (!notification.readAt) {
      notification.readAt = readAt
    }
  }
}

export function getMockUnreadCount(): NotificationUnreadCount {
  return {
    unreadCount: mockNotifications.filter((notification) => !notification.readAt).length,
  }
}

export function resetMockNotifications(): void {
  mockNotifications.splice(0, mockNotifications.length, ...INITIAL_NOTIFICATIONS.map((item) => ({ ...item })))
}
