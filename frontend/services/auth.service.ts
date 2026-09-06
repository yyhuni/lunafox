/**
 * Authentication service
 */
import {
  api,
  ensurePrimaryTokenFresh,
  primaryTokenExpirationMs,
  primaryTokenRefreshDelayMs,
  tokenManager,
} from '@/lib/api-client'
import type { 
  LoginRequest, 
  LoginResponse, 
  MeResponse, 
  LogoutResponse,
  ChangePasswordRequest,
  ChangePasswordResponse
} from '@/types/auth.types'

export function hasAuthTokens(): boolean {
  return tokenManager.hasTokens()
}

export function ensureFreshAuthToken(): Promise<string | null> {
  return ensurePrimaryTokenFresh()
}

export function authTokenExpirationMs(token: string): number | null {
  return primaryTokenExpirationMs(token)
}

export function authTokenRefreshDelayMs(token: string, now = Date.now()): number | null {
  return primaryTokenRefreshDelayMs(token, now)
}

/**
 * User login
 * Stores JWT tokens on successful login
 */
export async function login(data: LoginRequest): Promise<LoginResponse> {
  const res = await api.post<LoginResponse>('/sessions', data)
  
  // Store JWT tokens
  if (res.data.accessToken && res.data.refreshToken) {
    tokenManager.setTokens(res.data.accessToken, res.data.refreshToken)
  }
  
  return res.data
}

/**
 * User logout
 * Clears JWT tokens
 */
export async function logout(): Promise<LogoutResponse> {
  // Clear tokens first (even if API call fails)
  tokenManager.clearTokens()

  return { message: 'Logged out successfully' }
}

/**
 * Get current user information
 * Returns authenticated: false if no token
 */
export async function getMe(): Promise<MeResponse> {
  // If no token, return unauthenticated
  if (!tokenManager.hasTokens()) {
    return { authenticated: false, user: null }
  }
  
  try {
    const res = await api.get<{ id: number; username: string; email: string }>('/users/current')
    return {
      authenticated: true,
      user: res.data
    }
  } catch {
    // Token invalid or expired
    tokenManager.clearTokens()
    return { authenticated: false, user: null }
  }
}

/**
 * Change password
 */
export async function changePassword(data: ChangePasswordRequest): Promise<ChangePasswordResponse> {
  const res = await api.post<ChangePasswordResponse>('/users/me:changePassword', data)
  return res.data
}
