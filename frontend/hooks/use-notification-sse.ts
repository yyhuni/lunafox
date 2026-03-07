/**
 * WebSocket real-time notification Hook
 */

import { useCallback, useEffect, useState, useRef } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import type { BackendNotification, Notification, BackendNotificationLevel, NotificationSeverity } from '@/types/notification.types'
import { getBackendBaseUrl } from '@/lib/env'
import { useToastMessages } from '@/lib/toast-helpers'

const severityMap: Record<BackendNotificationLevel, NotificationSeverity> = {
  critical: 'critical',
  high: 'high',
  medium: 'medium',
  low: 'low',
}

interface NotificationTimeLabels {
  justNow: string
  minutesAgo: (count: number) => string
  hoursAgo: (count: number) => string
}

export interface NotificationTransformOptions {
  locale?: string
  timeLabels?: NotificationTimeLabels
}

const isZhLocale = (locale?: string) => locale?.toLowerCase().startsWith('zh') ?? false

const getFallbackTimeLabels = (locale?: string): NotificationTimeLabels => {
  if (isZhLocale(locale)) {
    return {
      justNow: '刚刚',
      minutesAgo: (count) => `${count} 分钟前`,
      hoursAgo: (count) => `${count} 小时前`,
    }
  }
  return {
    justNow: 'Just now',
    minutesAgo: (count) => `${count} minutes ago`,
    hoursAgo: (count) => `${count} hours ago`,
  }
}

const scanPattern = /(扫描|任务|\bscan\b|\btask\b)/i
const vulnerabilityPattern = /(漏洞|\bvulnerability\b|\bcve(?:-\d{4}-\d+)?\b)/i
const assetPattern = /(资产|\basset\b|\bsubdomain\b|\bdomain\b|\bip(?:\s+address)?\b|\bendpoint\b|\bwebsite\b)/i

const inferNotificationType = (message: string, category?: string) => {
  // Prioritize using the category returned by the backend
  if (category === 'scan' || category === 'vulnerability' || category === 'asset' || category === 'system') {
    return category
  }

  const normalizedMessage = message ?? ''

  // Fallback: inference from message content
  if (scanPattern.test(normalizedMessage)) {
    return 'scan' as const
  }
  if (vulnerabilityPattern.test(normalizedMessage)) {
    return 'vulnerability' as const
  }
  if (assetPattern.test(normalizedMessage)) {
    return 'asset' as const
  }
  return 'system' as const
}

const formatTimeAgo = (date: Date, options?: NotificationTransformOptions): string => {
  const now = new Date()
  const diffMs = Math.max(0, now.getTime() - date.getTime())
  const diffMins = Math.floor(diffMs / (1000 * 60))
  const diffHours = Math.floor(diffMs / (1000 * 60 * 60))
  const labels = options?.timeLabels ?? getFallbackTimeLabels(options?.locale)

  if (diffMins < 1) return labels.justNow
  if (diffMins < 60) return labels.minutesAgo(diffMins)
  if (diffHours < 24) return labels.hoursAgo(diffHours)
  return date.toLocaleDateString(options?.locale)
}

const isBackendNotification = (value: unknown): value is BackendNotification => {
  if (!value || typeof value !== 'object') return false
  const record = value as Record<string, unknown>
  return typeof record.id === 'number' && typeof record.title === 'string' && typeof record.message === 'string'
}

export const transformBackendNotification = (
  backendNotification: BackendNotification,
  options?: NotificationTransformOptions
): Notification => {
  const createdDate = backendNotification.createdAt ? new Date(backendNotification.createdAt) : new Date()
  const isRead = backendNotification.isRead
  const notification: Notification = {
    id: backendNotification.id,
    type: inferNotificationType(backendNotification.message, backendNotification.category),
    title: backendNotification.title,
    description: backendNotification.message,
    time: formatTimeAgo(createdDate, options),
    unread: isRead === true ? false : true,
    severity: severityMap[backendNotification.level] ?? undefined,
    createdAt: createdDate.toISOString(),
  }
  return notification
}

export function useNotificationSSE(transformOptions?: NotificationTransformOptions) {
  const [isConnected, setIsConnected] = useState(false)
  const [notifications, setNotifications] = useState<Notification[]>([])
  const wsRef = useRef<WebSocket | null>(null)
  const queryClient = useQueryClient()
  const reconnectTimerRef = useRef<NodeJS.Timeout | null>(null)
  const heartbeatTimerRef = useRef<NodeJS.Timeout | null>(null)
  const isConnectingRef = useRef(false)
  const transformOptionsRef = useRef(transformOptions)
  const reconnectAttempts = useRef(0)
  const maxReconnectAttempts = 10
  const baseReconnectDelay = 1000 // 1 second
  const toastMessages = useToastMessages()

  useEffect(() => {
    transformOptionsRef.current = transformOptions
  }, [transformOptions])

  useEffect(() => {
    setNotifications((prev) => {
      let hasChanged = false
      const updated = prev.map((notification) => {
        if (!notification.createdAt) return notification
        const createdDate = new Date(notification.createdAt)
        if (Number.isNaN(createdDate.getTime())) return notification
        const formattedTime = formatTimeAgo(createdDate, transformOptions)
        if (formattedTime === notification.time) return notification
        hasChanged = true
        return {
          ...notification,
          time: formattedTime,
        }
      })
      return hasChanged ? updated : prev
    })
  }, [transformOptions])

  const markNotificationsAsRead = useCallback((ids?: number[]) => {
    setNotifications(prev => prev.map(notification => {
      if (!ids || ids.includes(notification.id)) {
        return { ...notification, unread: false }
      }
      return notification
    }))
  }, [])

  // Start heartbeat
  const startHeartbeat = useCallback(() => {
    // Clear old heartbeat timers
    if (heartbeatTimerRef.current) {
      clearInterval(heartbeatTimerRef.current)
    }

    // Send heartbeat every 30 seconds
    heartbeatTimerRef.current = setInterval(() => {
      if (wsRef.current?.readyState === WebSocket.OPEN) {
        wsRef.current.send(JSON.stringify({ type: 'ping' }))
      }
    }, 30000) // 30 seconds
  }, [])

  // stop heartbeat
  const stopHeartbeat = useCallback(() => {
    if (heartbeatTimerRef.current) {
      clearInterval(heartbeatTimerRef.current)
      heartbeatTimerRef.current = null
    }
  }, [])

  // Calculate reconnection delay (exponential backoff)
  const getReconnectDelay = useCallback(() => {
    const delay = Math.min(baseReconnectDelay * Math.pow(2, reconnectAttempts.current), 30000)
    return delay
  }, [])

  // Connect WebSockets
  const connect = useCallback(() => {
    // Prevent duplicate connections
    if (isConnectingRef.current) {
      return
    }

    // If already connected, skip
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return
    }

    isConnectingRef.current = true

    // close old connection
    if (wsRef.current && wsRef.current.readyState !== WebSocket.CLOSED) {
      wsRef.current.close()
    }

    // Clear reconnect timer
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current)
      reconnectTimerRef.current = null
    }

    try {
      // Construct WebSocket URL
      const backendUrl = getBackendBaseUrl()
      const wsProtocol = backendUrl.startsWith('https') ? 'wss' : 'ws'
      const wsHost = backendUrl.replace(/^https?:\/\//, '')
      const wsUrl = `${wsProtocol}://${wsHost}/ws/notifications/`


      const ws = new WebSocket(wsUrl)
      wsRef.current = ws

      ws.onopen = () => {
        setIsConnected(true)
        isConnectingRef.current = false
        reconnectAttempts.current = 0 // Reset reconnection count
        // Start heartbeat
        startHeartbeat()
      }

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data) as Record<string, unknown>

          const messageType = typeof data.type === 'string' ? data.type : undefined

          if (messageType === 'connected') {
            return
          }

          if (messageType === 'pong') {
            // heartbeat response
            return
          }

          if (messageType === 'error') {
            const errorMessage = typeof data.message === 'string' ? data.message : ''
            toastMessages.error('toast.notification.connection.error', { message: errorMessage })
            return
          }

          // Handle notification messages
          if (messageType === 'notification') {
            if (isBackendNotification(data)) {
              const notification = transformBackendNotification(data, transformOptionsRef.current)
              setNotifications(prev => {
                const updated = [notification, ...prev.slice(0, 49)]
                return updated
              })

              queryClient.invalidateQueries({ queryKey: ['notifications'] })
            }
            return
          }

          // Alternate processing: Check notification fields directly
          if (isBackendNotification(data)) {
            const notification = transformBackendNotification(data, transformOptionsRef.current)

            // Add to notification list
            setNotifications(prev => {
              const updated = [notification, ...prev.slice(0, 49)]
              return updated
            })

            // Refresh notification query
            queryClient.invalidateQueries({ queryKey: ['notifications'] })
          }
        } catch (error) {
          void error
        }
      }

      ws.onerror = () => {
        // WebSocket onerror receives Event objects, not Errors
        // The actual error message is usually not available, only the connection status is logged
        setIsConnected(false)
        isConnectingRef.current = false
      }

      ws.onclose = (event) => {
        setIsConnected(false)
        isConnectingRef.current = false
        // stop heartbeat
        stopHeartbeat()

        // Automatic reconnection (when shutting down abnormally)
        if (event.code !== 1000) { // 1000 = Normal shutdown
          if (reconnectAttempts.current < maxReconnectAttempts) {
            const delay = getReconnectDelay()
            reconnectAttempts.current++
            reconnectTimerRef.current = setTimeout(() => {
              connect()
            }, delay)
          }
        }
      }
    } catch {
      setIsConnected(false)
      isConnectingRef.current = false
    }
  }, [queryClient, startHeartbeat, stopHeartbeat, getReconnectDelay, toastMessages])

  // Disconnect
  const disconnect = useCallback(() => {
    // stop heartbeat
    stopHeartbeat()

    // Clear reconnect timer
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current)
      reconnectTimerRef.current = null
    }

    // Reset reconnection count
    reconnectAttempts.current = 0
    isConnectingRef.current = false

    if (wsRef.current) {
      wsRef.current.close(1000, 'User disconnect') // 1000 = Normal shutdown
      wsRef.current = null
    }
    setIsConnected(false)
  }, [stopHeartbeat])

  // Clear notifications
  const clearNotifications = () => {
    setNotifications([])
  }

  // The component is connected when it is mounted and disconnected when it is unmounted.
  // Note: Do not rely on connect/disconnect to avoid infinite loops
  useEffect(() => {
    connect()

    return () => {
      disconnect()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return {
    isConnected,
    notifications,
    connect,
    disconnect,
    clearNotifications,
    markNotificationsAsRead,
  }
}
