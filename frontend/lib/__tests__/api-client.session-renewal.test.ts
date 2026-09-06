import axios, { AxiosError, type AxiosResponse, type InternalAxiosRequestConfig } from "axios"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import {
  api,
  apiClient,
  ensurePrimaryTokenFresh,
  isSessionAuthRequest,
  primaryTokenExpirationMs,
  primaryTokenRefreshDelayMs,
  TOKEN_REFRESH_LEAD_MS,
  tokenManager,
} from "@/lib/api-client"
import { NotificationService } from "@/services/notification.service"

function tokenWithExpiration(expirationSeconds: number) {
  const payload = btoa(JSON.stringify({ exp: expirationSeconds }))
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "")
  return `header.${payload}.signature`
}

function successfulResponse<T>(
  config: InternalAxiosRequestConfig,
  data: T,
): AxiosResponse<T> {
  return {
    config,
    data,
    headers: {},
    status: 200,
    statusText: "OK",
  }
}

describe("api-client session renewal", () => {
  const originalAdapter = apiClient.defaults.adapter

  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date("2026-08-18T12:00:00.000Z"))
    vi.clearAllMocks()
    tokenManager.clearTokens()
  })

  afterEach(() => {
    apiClient.defaults.adapter = originalAdapter
    tokenManager.clearTokens()
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it("只接受数值 exp，并用统一提前窗口计算下一次检查", () => {
    const expiration = Math.floor((Date.now() + TOKEN_REFRESH_LEAD_MS + 30_000) / 1_000)
    const token = tokenWithExpiration(expiration)

    expect(primaryTokenExpirationMs(token)).toBe(expiration * 1_000)
    expect(primaryTokenExpirationMs("header.invalid.signature")).toBeNull()
    expect(primaryTokenRefreshDelayMs(token)).toBe(30_000)
  })

  it("HTTP freshness 与 SSE service 并发检查只发送一次续签请求", async () => {
    const expiringToken = tokenWithExpiration(Math.floor((Date.now() + 30_000) / 1_000))
    const renewedToken = tokenWithExpiration(Math.floor((Date.now() + 3_600_000) / 1_000))
    tokenManager.setTokens(expiringToken, "renewal-token")
    const renewRequest = vi.spyOn(axios, "post").mockResolvedValue({
      data: { accessToken: renewedToken },
    } as never)

    await expect(Promise.all([
      ensurePrimaryTokenFresh(),
      NotificationService.getFreshStreamToken(),
    ])).resolves.toEqual([renewedToken, renewedToken])

    expect(renewRequest).toHaveBeenCalledTimes(1)
    expect(tokenManager.getPrimaryToken()).toBe(renewedToken)
  })

  it("在发送受保护请求前续签，并把新 token 放入 Authorization", async () => {
    const expiringToken = tokenWithExpiration(Math.floor((Date.now() + 30_000) / 1_000))
    const renewedToken = tokenWithExpiration(Math.floor((Date.now() + 3_600_000) / 1_000))
    tokenManager.setTokens(expiringToken, "renewal-token")
    const renewRequest = vi.spyOn(axios, "post").mockResolvedValue({
      data: { accessToken: renewedToken },
    } as never)
    const adapter = vi.fn((config: InternalAxiosRequestConfig) =>
      Promise.resolve(successfulResponse(config, { ok: true }))
    )
    apiClient.defaults.adapter = adapter

    await api.get("/users/current")

    expect(renewRequest).toHaveBeenCalledTimes(1)
    expect(adapter).toHaveBeenCalledWith(expect.objectContaining({
      headers: expect.objectContaining({ Authorization: `Bearer ${renewedToken}` }),
    }))
  })

  it("跳过 session endpoint 的请求前续签，并保留一次 401 fallback", async () => {
    const nearExpiryToken = tokenWithExpiration(Math.floor((Date.now() + 30_000) / 1_000))
    const stillValidToken = tokenWithExpiration(Math.floor((Date.now() + 3_600_000) / 1_000))
    const renewedToken = tokenWithExpiration(Math.floor((Date.now() + 7_200_000) / 1_000))
    tokenManager.setTokens(nearExpiryToken, "renewal-token")
    const renewRequest = vi.spyOn(axios, "post").mockResolvedValue({
      data: { accessToken: renewedToken },
    } as never)
    const adapter = vi.fn((config: InternalAxiosRequestConfig) =>
      Promise.resolve(successfulResponse(config, { ok: true }))
    )
    apiClient.defaults.adapter = adapter

    await api.post("/sessions", { username: "operator" })

    expect(renewRequest).not.toHaveBeenCalled()
    expect(adapter).toHaveBeenCalledWith(expect.objectContaining({
      headers: expect.objectContaining({ Authorization: `Bearer ${nearExpiryToken}` }),
    }))

    tokenManager.setTokens(stillValidToken, "renewal-token")
    let protectedRequestCount = 0
    apiClient.defaults.adapter = vi.fn((config: InternalAxiosRequestConfig) => {
      protectedRequestCount += 1
      if (protectedRequestCount === 1) {
        return Promise.reject(new AxiosError(
          "Unauthorized",
          undefined,
          config,
          undefined,
          {
            config,
            data: { error: "expired" },
            headers: {},
            status: 401,
            statusText: "Unauthorized",
          }
        ))
      }
      return Promise.resolve(successfulResponse(config, { ok: true }))
    })

    await api.get("/users/current")

    expect(protectedRequestCount).toBe(2)
    expect(renewRequest).toHaveBeenCalledTimes(1)
  })

  it("只把 canonical session routes 识别为认证端点", () => {
    expect(isSessionAuthRequest("/sessions")).toBe(true)
    expect(isSessionAuthRequest("/v1/sessions:renew?source=timer")).toBe(true)
    expect(isSessionAuthRequest("sessions")).toBe(true)
    expect(isSessionAuthRequest("/users/current/notifications")).toBe(false)
  })
})
