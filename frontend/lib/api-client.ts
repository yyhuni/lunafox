/**
 * API client configuration file
 * 
 * Core functionality:
 * 1. Unified HTTP request wrapper
 * 2. Unified error handling
 * 3. Request/response logging
 * 4. JWT token management with auto-refresh
 * 
 * Naming convention explanation:
 * - Frontend (TypeScript/React): camelCase
 *   Example: pageSize, createdAt, organizationId
 * 
 * - Backend (Go): Uses JSON tags for camelCase output
 *   Example: pageSize, createdAt, organizationId
 * 
 * - API JSON format: camelCase
 *   Example: pageSize, createdAt, organizationId
 */

import axios, { AxiosRequestConfig, AxiosError, InternalAxiosRequestConfig } from 'axios';
import { USE_MOCK } from '@/mock/config';
import {
  AUTH_PRIMARY_TOKEN_FIELD,
  AUTH_PRIMARY_TOKEN_KEY,
  AUTH_RENEWAL_TOKEN_FIELD,
  AUTH_RENEWAL_TOKEN_KEY,
} from '@/lib/auth-storage-keys.mjs';
import {
  buildLoginRedirectPathForLocation,
} from '@/lib/auth-runtime.mjs';

// Axios requests and fetch-SSE streams share one renewal promise. This keeps
// an expired access token from creating concurrent /sessions:renew calls.
let activePrimaryTokenRenewal: Promise<string> | null = null;

// Keep a small buffer so normal requests do not race the Server's exp boundary.
export const TOKEN_REFRESH_LEAD_MS = 60_000;

export function primaryTokenExpirationMs(token: string): number | null {
  const payload = token.split(".")[1];
  if (!payload || typeof atob !== "function") {
    return null;
  }

  try {
    const base64 = payload.replace(/-/g, "+").replace(/_/g, "/");
    const padded = base64.padEnd(Math.ceil(base64.length / 4) * 4, "=");
    const bytes = Uint8Array.from(atob(padded), (character) => character.charCodeAt(0));
    const data = JSON.parse(new TextDecoder().decode(bytes)) as { exp?: unknown };
    if (typeof data.exp !== "number" || !Number.isFinite(data.exp)) {
      return null;
    }
    const expirationMs = data.exp * 1_000;
    return Number.isFinite(expirationMs) ? expirationMs : null;
  } catch {
    return null;
  }
}

export function primaryTokenRefreshDelayMs(
  token: string,
  now = Date.now(),
): number | null {
  const expirationMs = primaryTokenExpirationMs(token);
  if (expirationMs === null) {
    return null;
  }
  return Math.max(0, expirationMs - now - TOKEN_REFRESH_LEAD_MS);
}

export function isSessionAuthRequest(url?: string): boolean {
  const pathname = (url ?? "").split(/[?#]/, 1)[0];
  return /(?:^|\/)sessions(?::renew)?(?:\/|$)/.test(pathname);
}

// Cache for localStorage reads to avoid expensive I/O operations
const storageCache = new Map<string, string | null>();

const getLoginRedirectPath = (): string => {
  if (typeof window === 'undefined') {
    return '/login/';
  }
  return buildLoginRedirectPathForLocation(
    window.location.pathname,
    window.location.search,
    window.location.hash
  );
};

export const ensureMockInterceptionReady = async (): Promise<void> => {
  if (!USE_MOCK || typeof window === 'undefined') {
    return;
  }

  const { startMockWorker } = await import("@/mock/browser");
  await startMockWorker();
};

/**
 * Token management utilities with caching and error handling
 */
export const tokenManager = {
  getPrimaryToken: (): string | null => {
    if (typeof window === 'undefined') return null;

    // Check cache first
    if (storageCache.has(AUTH_PRIMARY_TOKEN_KEY)) {
      return storageCache.get(AUTH_PRIMARY_TOKEN_KEY)!;
    }

    // Read from localStorage with error handling
    try {
      const token = localStorage.getItem(AUTH_PRIMARY_TOKEN_KEY);
      storageCache.set(AUTH_PRIMARY_TOKEN_KEY, token);
      return token;
    } catch {
      // localStorage throws in incognito mode or when disabled
      return null;
    }
  },

  getRenewalToken: (): string | null => {
    if (typeof window === 'undefined') return null;

    // Check cache first
    if (storageCache.has(AUTH_RENEWAL_TOKEN_KEY)) {
      return storageCache.get(AUTH_RENEWAL_TOKEN_KEY)!;
    }

    // Read from localStorage with error handling
    try {
      const token = localStorage.getItem(AUTH_RENEWAL_TOKEN_KEY);
      storageCache.set(AUTH_RENEWAL_TOKEN_KEY, token);
      return token;
    } catch {
      // localStorage throws in incognito mode or when disabled
      return null;
    }
  },

  setTokens: (primaryToken: string, renewalToken: string): void => {
    if (typeof window === 'undefined') return;
    try {
      localStorage.setItem(AUTH_PRIMARY_TOKEN_KEY, primaryToken);
      localStorage.setItem(AUTH_RENEWAL_TOKEN_KEY, renewalToken);
      // Update cache
      storageCache.set(AUTH_PRIMARY_TOKEN_KEY, primaryToken);
      storageCache.set(AUTH_RENEWAL_TOKEN_KEY, renewalToken);
    } catch {
      // localStorage throws when quota exceeded or disabled
    }
  },

  setPrimaryToken: (primaryToken: string): void => {
    if (typeof window === 'undefined') return;
    try {
      localStorage.setItem(AUTH_PRIMARY_TOKEN_KEY, primaryToken);
      // Update cache
      storageCache.set(AUTH_PRIMARY_TOKEN_KEY, primaryToken);
    } catch {
      // localStorage throws when quota exceeded or disabled
    }
  },

  clearTokens: (): void => {
    if (typeof window === 'undefined') return;
    try {
      localStorage.removeItem(AUTH_PRIMARY_TOKEN_KEY);
      localStorage.removeItem(AUTH_RENEWAL_TOKEN_KEY);
      // Clear cache
      storageCache.clear();
    } catch {
      // localStorage throws when disabled
    }
  },

  hasTokens: (): boolean => {
    return !!tokenManager.getPrimaryToken();
  }
};

/**
 * End the local session through the same redirect behavior used by protected
 * Axios requests. Fetch-SSE callers use this for non-renewable 401 responses.
 */
export function redirectToLoginAfterSessionFailure(): void {
  tokenManager.clearTokens();
  if (typeof window !== 'undefined') {
    window.location.href = getLoginRedirectPath();
  }
}

/**
 * Renew the primary access token once for all active HTTP and SSE consumers.
 */
export function renewPrimaryToken(): Promise<string> {
  if (activePrimaryTokenRenewal) {
    return activePrimaryTokenRenewal;
  }

  const renewalToken = tokenManager.getRenewalToken();
  if (!renewalToken) {
    const error = new Error('Renewal token is required');
    redirectToLoginAfterSessionFailure();
    return Promise.reject(error);
  }

  const renewal = (async () => {
    await ensureMockInterceptionReady();
    try {
      const response = await axios.post('/v1/sessions:renew', {
        [AUTH_RENEWAL_TOKEN_FIELD]: renewalToken,
      });
      const nextPrimaryToken = response.data?.[AUTH_PRIMARY_TOKEN_FIELD];
      if (typeof nextPrimaryToken !== 'string' || !nextPrimaryToken.trim()) {
        throw new Error('Session renewal did not return an access token');
      }
      tokenManager.setPrimaryToken(nextPrimaryToken);
      return nextPrimaryToken;
    } catch (error) {
      redirectToLoginAfterSessionFailure();
      throw error;
    }
  })();

  activePrimaryTokenRenewal = renewal;
  void renewal.then(
    () => {
      if (activePrimaryTokenRenewal === renewal) {
        activePrimaryTokenRenewal = null;
      }
    },
    () => {
      if (activePrimaryTokenRenewal === renewal) {
        activePrimaryTokenRenewal = null;
      }
    }
  );
  return renewal;
}

/**
 * Return the current access token, renewing it when it reaches the lead window.
 * The caller must read the returned value rather than reusing a token captured
 * before the await, because HTTP and SSE checks can complete concurrently.
 */
export async function ensurePrimaryTokenFresh(): Promise<string | null> {
  const token = tokenManager.getPrimaryToken();
  if (!token) {
    return null;
  }

  const expirationMs = primaryTokenExpirationMs(token);
  if (expirationMs === null || expirationMs - Date.now() > TOKEN_REFRESH_LEAD_MS) {
    return token;
  }

  return renewPrimaryToken();
}

/**
 * Create axios instance
 * Configure base URL, timeout and default headers
 */
export const apiClient = axios.create({
  baseURL: '/v1',  // Backend API base path
  timeout: 30000,      // 30 second timeout
  headers: {
    'Content-Type': 'application/json',
  },
});

/**
 * Request interceptor: Handle preparation work before request
 * 
 * Workflow:
 * 1. Add Authorization header with JWT token
 * 2. (Removed) Log request for debugging
 */
apiClient.interceptors.request.use(
  async (config) => {
    await ensureMockInterceptionReady();

    // Auth endpoints must not recursively trigger a session renewal.
    const token = isSessionAuthRequest(config.url)
      ? tokenManager.getPrimaryToken()
      : await ensurePrimaryTokenFresh();
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }

    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

/**
 * Response interceptor: Handle response data and auto-refresh token
 * 
 * Workflow:
 * 1. On 401 error, try to refresh token and retry the request
 * 2. Return response data
 */
apiClient.interceptors.response.use(
  (response) => {
    return response;
  },
  async (error: AxiosError) => {
    const originalRequest = error.config as InternalAxiosRequestConfig & { _retry?: boolean };

    // Handle 401 Unauthorized with auto-refresh
    if (error.response?.status === 401 && originalRequest && !originalRequest._retry) {
      const url = originalRequest.url || '';
      
      if (isSessionAuthRequest(url)) {
        return Promise.reject(error);
      }

      originalRequest._retry = true;

      try {
        const nextPrimaryToken = await renewPrimaryToken();
        originalRequest.headers.Authorization = `Bearer ${nextPrimaryToken}`;
        return apiClient(originalRequest);
      } catch (refreshError) {
        return Promise.reject(refreshError);
      }
    }

    return Promise.reject(error);
  }
);

// Export default axios instance (generally not used directly)
export default apiClient;

/**
 * Export common HTTP methods
 * 
 * Usage examples:
 * 
 * 1. GET request:
 *    api.get('/organizations', {
 *      params: { pageSize: 10, sortBy: 'name' }
 *    })
 * 
 * 2. POST request:
 *    api.post('/organizations/create', {
 *      organizationName: 'test',
 *      createdAt: '2024-01-01'
 *    })
 * 
 * 3. Response data (already camelCase):
 *    const response = await api.get('/organizations')
 *    response.data.pageSize  // [OK] Use camelCase directly
 *    response.data.createdAt // [OK] Use camelCase directly
 * 
 * Type parameters:
 * - T: Response data type (optional)
 * - config: axios configuration object (optional)
 */
export const api = {
  /**
   * GET request
   * @param url - Request path (relative to baseURL)
   * @param config - axios config, recommend using params for query parameters
   * @returns Promise<AxiosResponse<T>>
   */
  get: <T = unknown>(url: string, config?: AxiosRequestConfig) => apiClient.get<T>(url, config),

  /**
   * POST request
   * @param url - Request path (relative to baseURL)
   * @param data - Request body data, use API contract field names directly
   * @param config - axios config (optional)
   * @returns Promise<AxiosResponse<T>>
   */
  post: <T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig) => apiClient.post<T>(url, data, config),

  /**
   * PUT request
   * @param url - Request path (relative to baseURL)
   * @param data - Request body data, use API contract field names directly
   * @param config - axios config (optional)
   * @returns Promise<AxiosResponse<T>>
   */
  put: <T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig) => apiClient.put<T>(url, data, config),

  /**
   * PATCH request (partial update)
   * @param url - Request path (relative to baseURL)
   * @param data - Request body data, use API contract field names directly
   * @param config - axios config (optional)
   * @returns Promise<AxiosResponse<T>>
   */
  patch: <T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig) => apiClient.patch<T>(url, data, config),

  /**
   * DELETE request
   * @param url - Request path (relative to baseURL)
   * @param config - axios config (optional)
   * @returns Promise<AxiosResponse<T>>
   */
  delete: <T = unknown>(url: string, config?: AxiosRequestConfig) => apiClient.delete<T>(url, config),
};

/**
 * Error handling utility function
 * 
 * Function: Extract user-friendly error messages from error objects
 * 
 * Error priority:
 * 1. Request cancelled
 * 2. Request timeout
 * 3. Backend returned error message
 * 4. axios error message
 * 5. Unknown error
 * 
 * Usage example:
 * try {
 *   await api.get('/organizations')
 * } catch (error) {
 *   const message = getErrorMessage(error)
 *   toast.error(message)
 * }
 * 
 * @param error - Error object (can be any type)
 * @returns User-friendly error message string
 */
export const getErrorMessage = (error: unknown): string => {
  // Request was cancelled (user actively cancelled or component unmounted)
  if (axios.isCancel(error)) {
    return 'Request has been cancelled';
  }

  // Type guard: Check if it's an error object
  const err = error as {
    code?: string;
    response?: { data?: { 
      message?: string; 
      error?: string | { code?: string; message?: string; details?: Array<{ field?: string; message?: string }> }; 
      detail?: string 
    } };
    message?: string
  }

  // Request timeout (over 30 seconds)
  if (err.code === 'ECONNABORTED') {
    return 'Request timeout, please try again later';
  }

  // Backend returned error message (supports multiple formats)
  const errorData = err.response?.data?.error;
  if (errorData) {
    // New format: { error: { code, message, details } }
    if (typeof errorData === 'object') {
      // If has validation details, return first detail message
      if (errorData.details && errorData.details.length > 0) {
        const detail = errorData.details[0];
        return detail.message || errorData.message || 'Validation error';
      }
      return errorData.message || 'Unknown error';
    }
    // Old format: { error: "string" }
    return errorData;
  }
  if (err.response?.data?.message) {
    return err.response.data.message;
  }
  if (err.response?.data?.detail) {
    return err.response.data.detail;
  }

  // axios own error message
  if (err.message) {
    return err.message;
  }

  // Fallback error message
  return 'Unknown error occurred';
};
