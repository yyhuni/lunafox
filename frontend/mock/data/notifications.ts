import type { BackendNotification, GetNotificationsResponse } from '@/types/notification.types'

export const mockNotifications: BackendNotification[] = [
  {
    id: 1,
    category: 'vulnerability',
    title: 'Critical Vulnerability Found',
    message: 'SQL Injection detected in retailmax.com/product endpoint',
    level: 'critical',
    createdAt: '2024-12-29T10:30:00Z',
    isRead: false,
  },
  {
    id: 2,
    category: 'scan',
    title: 'Scan Completed',
    message: 'Scan for acme.com completed successfully with 23 vulnerabilities found',
    level: 'medium',
    createdAt: '2024-12-29T09:00:00Z',
    isRead: false,
  },
  {
    id: 3,
    category: 'vulnerability',
    title: 'High Severity Vulnerability',
    message: 'XSS vulnerability found in acme.com/search',
    level: 'high',
    createdAt: '2024-12-28T16:45:00Z',
    isRead: true,
  },
  {
    id: 4,
    category: 'scan',
    title: 'Scan Failed',
    message: 'Scan for globalfinance.com failed: Connection timeout',
    level: 'high',
    createdAt: '2024-12-28T14:20:00Z',
    isRead: true,
  },
  {
    id: 5,
    category: 'asset',
    title: 'New Subdomains Discovered',
    message: '15 new subdomains discovered for techstart.io',
    level: 'low',
    createdAt: '2024-12-27T11:00:00Z',
    isRead: true,
  },
  {
    id: 6,
    category: 'system',
    title: 'Worker Offline',
    message: 'Worker node worker-03 is now offline',
    level: 'medium',
    createdAt: '2024-12-27T08:30:00Z',
    isRead: true,
  },
  {
    id: 7,
    category: 'scan',
    title: 'Scheduled Scan Started',
    message: 'Scheduled scan for Acme Corporation started',
    level: 'low',
    createdAt: '2024-12-26T06:00:00Z',
    isRead: true,
  },
  {
    id: 8,
    category: 'system',
    title: 'System Update Available',
    message: 'A new version of the scanner is available',
    level: 'low',
    createdAt: '2024-12-25T10:00:00Z',
    isRead: true,
  },
]

export function getMockNotifications(params?: {
  page?: number
  pageSize?: number
  unread?: boolean
}): GetNotificationsResponse {
  const page = params?.page || 1
  const pageSize = params?.pageSize || 10

  let filtered = mockNotifications

  if (params?.unread) {
    filtered = filtered.filter(n => !n.isRead)
  }

  const total = filtered.length
  const totalPages = Math.ceil(total / pageSize)
  const start = (page - 1) * pageSize
  const results = filtered.slice(start, start + pageSize)

  return {
    results,
    total,
    page,
    pageSize,
    totalPages,
  }
}

export function getMockUnreadCount(): { count: number } {
  return {
    count: mockNotifications.filter(n => !n.isRead).length,
  }
}
