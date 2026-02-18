import type { User, MeResponse, LoginResponse, LogoutResponse } from '@/types/auth.types'

export const mockUser: User = {
  id: 1,
  username: 'admin',
  email: 'admin@example.com',
}

export const mockMeResponse: MeResponse = {
  authenticated: true,
  user: mockUser,
}

export const mockLoginResponse: LoginResponse = {
  accessToken: 'mock-access-token',
  refreshToken: 'mock-refresh-token',
  expiresIn: 900,
  user: mockUser,
}

export const mockLogoutResponse: LogoutResponse = {
  message: 'Logout successful',
}
