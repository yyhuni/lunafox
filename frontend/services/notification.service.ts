import {
  api,
  ensureMockInterceptionReady,
  ensurePrimaryTokenFresh,
  redirectToLoginAfterSessionFailure,
  renewPrimaryToken,
  tokenManager,
} from "@/lib/api-client"
import { buildBackendUrl } from "@/lib/env"
import type {
  ListNotificationsRequest,
  ListNotificationsResponse,
  NotificationUnreadCount,
} from "@/types/notification.types"

const inboxPath = "/users/current/notifications"

type NotificationStreamErrorBody = {
  error?: {
    details?: Array<{ reason?: unknown } | unknown>
  }
}

export class NotificationService {
  static getStreamToken(): string | null {
    return tokenManager.getPrimaryToken()
  }

  static async getFreshStreamToken(): Promise<string | null> {
    const freshToken = await ensurePrimaryTokenFresh()
    // Read after the await so an SSE connection cannot reuse the token that
    // was captured before a concurrent shared renewal completed.
    return freshToken ?? tokenManager.getPrimaryToken()
  }

  static renewStreamToken(): Promise<string> {
    return renewPrimaryToken()
  }

  static handleStreamSessionFailure(): void {
    redirectToLoginAfterSessionFailure()
  }

  static async list(
    params: ListNotificationsRequest = {}
  ): Promise<ListNotificationsResponse> {
    const response = await api.get<ListNotificationsResponse>(inboxPath, {
      params,
    })
    return response.data
  }

  static async markAllRead(): Promise<void> {
    await api.post<void>(`${inboxPath}:markAllRead`)
  }

  static async getUnreadCount(): Promise<NotificationUnreadCount> {
    const response = await api.get<NotificationUnreadCount>(`${inboxPath}:unreadCount`)
    return response.data
  }

  static async openStream(token: string, signal: AbortSignal): Promise<Response> {
    if (!token.trim()) {
      throw new Error("Notification stream token is required")
    }
    await ensureMockInterceptionReady()
    return fetch(buildBackendUrl("/v1/users/current/notifications:stream"), {
      method: "GET",
      headers: {
        Accept: "text/event-stream",
        Authorization: `Bearer ${token}`,
      },
      cache: "no-store",
      signal,
    })
  }

  static async streamErrorReason(response: Response): Promise<string | null> {
    try {
      const body = await response.clone().json() as NotificationStreamErrorBody
      for (const detail of body.error?.details ?? []) {
        if (
          typeof detail === "object" &&
          detail !== null &&
          typeof (detail as { reason?: unknown }).reason === "string"
        ) {
          return (detail as { reason: string }).reason
        }
      }
    } catch {
      return null
    }
    return null
  }
}
