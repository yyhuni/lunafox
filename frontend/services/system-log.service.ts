import type { AxiosError } from "axios"

import { api } from "@/lib/api-client"
import type { SystemLogItem, SystemLogsResponse } from "@/types/system-log.types"

const BASE_URL = "/admin/system"

const DEFAULT_LOG_LIMIT = 200
const MAX_LOG_LIMIT = 500

export interface FetchServerLogsParams {
  limit?: number
  cursor?: string
  direction?: "newer" | "older"
  signal?: AbortSignal
}

export class SystemLogQueryError extends Error {
  code: string
  status?: number

  constructor(code: string, message: string, status?: number) {
    super(message)
    this.name = "SystemLogQueryError"
    this.code = code
    this.status = status
  }
}

export const systemLogService = {
  async fetchServerLogs(params: FetchServerLogsParams = {}): Promise<SystemLogsResponse> {
    const limit = requireLogLimit(params.limit)
    const cursor = params.cursor?.trim() ?? ""
    const direction = params.direction?.trim() ?? ""
    const queryParams: Record<string, string> = {
      pageSize: String(limit),
    }
    if (cursor) {
      queryParams.pageToken = cursor
    }
    if (direction) {
      queryParams.direction = direction
    }

    let payload: {
      results?: unknown[]
      nextPageToken?: string
      previousPageToken?: string
      hasOlder?: unknown
      hasNewer?: unknown
      caughtUp?: unknown
      gap?: unknown
      gapReason?: unknown
    }
    try {
      const response = await api.get<{
        results?: unknown[]
        nextPageToken?: string
        previousPageToken?: string
        hasOlder?: unknown
        hasNewer?: unknown
        caughtUp?: unknown
        gap?: unknown
        gapReason?: unknown
      }>(`${BASE_URL}/logEntries`, { params: queryParams, signal: params.signal })
      payload = response.data ?? {}
    } catch (error) {
      if (isCanceledRequestError(error)) {
        throw error
      }
      throw toSystemLogQueryError(error)
    }

    const logs = Array.isArray(payload.results)
      ? payload.results.map((item) => normalizeSystemLogItem(item)).filter((item): item is SystemLogItem => item !== null)
      : []

    return {
      logs,
      nextCursor: asString(payload.nextPageToken),
      previousCursor: asString(payload.previousPageToken),
      hasOlder: asBoolean(payload.hasOlder),
      hasNewer: asBoolean(payload.hasNewer),
      caughtUp: payload.caughtUp === true,
      gap: asBoolean(payload.gap),
      gapReason: asString(payload.gapReason),
    }
  },
}

function isCanceledRequestError(error: unknown): boolean {
  if (error instanceof DOMException) {
    return error.name === "AbortError"
  }
  if (!(error instanceof Error)) {
    return false
  }
  const maybeCanceled = error as Error & { code?: unknown }
  return error.name === "AbortError" || error.name === "CanceledError" || maybeCanceled.code === "ERR_CANCELED"
}

function toSystemLogQueryError(error: unknown): SystemLogQueryError {
  const axiosError = error as AxiosError<{
    error?: {
      code?: string
      message?: string
    }
  }>

  const status = axiosError?.response?.status
  let code = typeof status === "number" ? `http_${status}` : "network_error"
  let message = axiosError?.message || "Request failed"

  const body = axiosError?.response?.data
  if (body?.error?.code) {
    code = body.error.code
  }
  if (body?.error?.message) {
    message = body.error.message
  }

  return new SystemLogQueryError(code, message, status)
}

function normalizeSystemLogItem(raw: unknown): SystemLogItem | null {
  if (!raw || typeof raw !== "object") {
    return null
  }
  const item = raw as Partial<SystemLogItem>
  const id = asString(item.id)
  if (!id) {
    return null
  }
  return {
    id,
    ts: asString(item.ts),
    tsNs: asString(item.tsNs),
    stream: asString(item.stream) || "stdout",
    line: asString(item.line),
    truncated: Boolean(item.truncated),
  }
}

function asString(value: unknown): string {
  return typeof value === "string" ? value : ""
}

function asBoolean(value: unknown): boolean {
  return value === true
}

function requireLogLimit(value: number | undefined): number {
  const limit = value ?? DEFAULT_LOG_LIMIT
  if (!Number.isInteger(limit) || limit < 1 || limit > MAX_LOG_LIMIT) {
    throw new SystemLogQueryError("bad_request", `pageSize must be between 1 and ${MAX_LOG_LIMIT}`)
  }
  return limit
}
