/**
 * Authentication service
 */
import { api, tokenManager } from '@/lib/api-client'
import type { 
  LoginRequest, 
  LoginResponse, 
  MeResponse, 
  LogoutResponse,
  ChangePasswordRequest,
  ChangePasswordResponse
} from '@/types/auth.types'
import { USE_MOCK, mockDelay, mockLoginResponse, mockLogoutResponse, mockMeResponse } from '@/mock'

/**
 * User login
 * Stores JWT tokens on successful login
 */
export async function login(data: LoginRequest): Promise<LoginResponse> {
  if (USE_MOCK) {
    await mockDelay()
    // Keep mock auth behavior aligned with real mode so refresh can restore auth state.
    tokenManager.setTokens(mockLoginResponse.accessToken, mockLoginResponse.refreshToken)
    return mockLoginResponse
  }
  const res = await api.post<LoginResponse>('/auth/login/', data)
  
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
  if (USE_MOCK) {
    await mockDelay()
    tokenManager.clearTokens()
    return mockLogoutResponse
  }
  
  // Clear tokens first (even if API call fails)
  tokenManager.clearTokens()
  
  // Optionally notify backend (for token blacklisting if implemented)
  try {
    const res = await api.post<LogoutResponse>('/auth/logout/')
    return res.data
  } catch {
    // Logout is successful even if API call fails (tokens are cleared)
    return { message: 'Logged out successfully' }
  }
}

/**
 * Get current user information
 * Returns authenticated: false if no token
 */
export async function getMe(): Promise<MeResponse> {
  if (USE_MOCK) {
    await mockDelay()
    if (!tokenManager.hasTokens()) {
      return { authenticated: false, user: null }
    }
    return mockMeResponse
  }
  
  // If no token, return unauthenticated
  if (!tokenManager.hasTokens()) {
    return { authenticated: false, user: null }
  }
  
  try {
    const res = await api.get<{ id: number; username: string; email: string }>('/auth/me/')
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
  if (USE_MOCK) {
    await mockDelay()
    return { message: 'Password changed successfully' }
  }
  const res = await api.put<ChangePasswordResponse>('/users/me/password/', data)
  return res.data
}
