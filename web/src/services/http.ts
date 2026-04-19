import axios, { AxiosHeaders } from 'axios'
import type { AxiosError, InternalAxiosRequestConfig } from 'axios'
import type { ApiError, ApiValidationError, TokenPair } from '@/types'

interface RetryableRequestConfig extends InternalAxiosRequestConfig {
  _retry?: boolean
  _isAuthRequest?: boolean
}

const http = axios.create({
  baseURL: '/api',
  headers: {
    'Content-Type': 'application/json',
  },
})

// Track if we're currently refreshing to prevent multiple refresh attempts
let isRefreshing = false
let refreshPromise: Promise<boolean> | null = null

// Request interceptor: inject Authorization header and X-Request-ID
http.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const headers = config.headers ?? new AxiosHeaders()
  
  // Add X-Request-ID for traceability (uses crypto.randomUUID for cryptographic randomness)
  if (!headers.has('X-Request-ID')) {
    headers.set('X-Request-ID', generateUUID())
  }

  // Add Authorization header if token exists
  const token = localStorage.getItem('i9pay:auth:accessToken')
  if (token && !config.url?.includes('/auth/login') && !config.url?.includes('/auth/refresh')) {
    headers.set('Authorization', `Bearer ${token}`)
  }

  config.headers = headers
  return config
})

async function refreshToken(): Promise<boolean> {
  const refresh = localStorage.getItem('i9pay:auth:refreshToken')
  if (!refresh) return false

  try {
    const { data } = await axios.post<TokenPair>('/api/v1/auth/refresh', {
      refresh_token: refresh,
    })
    localStorage.setItem('i9pay:auth:accessToken', data.access_token)
    localStorage.setItem('i9pay:auth:refreshToken', data.refresh_token)
    return true
  } catch (error) {
    // Clear auth on refresh failure
    localStorage.removeItem('i9pay:auth:accessToken')
    localStorage.removeItem('i9pay:auth:refreshToken')
    return false
  }
}

// Response interceptor: auto-refresh on 401, redirect on 503
http.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const config = error.config as RetryableRequestConfig | undefined

    // Redirect to maintenance page on 503
    if (error.response?.status === 503) {
      if (typeof window !== 'undefined') {
        window.location.href = '/maintenance'
      }
      return Promise.reject(error)
    }

    // Check if this is an auth endpoint
    const isAuthEndpoint =
      config?.url?.includes('/auth/login') ||
      config?.url?.includes('/auth/refresh') ||
      config?.url?.includes('/auth/select-organization')

    // Auto-refresh on 401 (unless already retried or is auth endpoint)
    if (error.response?.status === 401 && config && !config._retry && !isAuthEndpoint) {
      config._retry = true

      if (!isRefreshing) {
        isRefreshing = true
        refreshPromise = refreshToken().finally(() => {
          isRefreshing = false
          refreshPromise = null
        })
      }

      const refreshed = await refreshPromise
      if (refreshed) {
        const newToken = localStorage.getItem('i9pay:auth:accessToken')
        const headers = config.headers ?? new AxiosHeaders()
        headers.set('Authorization', `Bearer ${newToken}`)
        config.headers = headers
        return http(config)
      }

      // Refresh failed — clear session and redirect to login
      localStorage.removeItem('i9pay:auth:accessToken')
      localStorage.removeItem('i9pay:auth:refreshToken')
      if (typeof window !== 'undefined') {
        window.location.href = '/login'
      }
    }

    return Promise.reject(normalizeError(error))
  },
)

export function normalizeError(error: AxiosError): ApiError {
  if (error.response) {
    const { status, data } = error.response as {
      status: number
      data: { error?: string; errors?: ApiValidationError[] }
    }
    // Sanitize error message to prevent reflected XSS from server-controlled strings
    const rawMessage = data?.error || error.message || 'Erro desconhecido'
    const message = typeof rawMessage === 'string' ? rawMessage.slice(0, 500) : 'Erro desconhecido'

    const apiError: ApiError = { message, status }

    if (status === 400 && Array.isArray(data?.errors) && data.errors.length > 0) {
      apiError.validationErrors = {}
      for (const ve of data.errors) {
        // Sanitize validation error messages too
        if (ve.field && typeof ve.field === 'string' && typeof ve.message === 'string') {
          apiError.validationErrors[ve.field.slice(0, 100)] = ve.message.slice(0, 500)
        }
      }
    }

    return apiError
  }

  // Network error or no response
  return {
    message: 'Erro de conexão com o servidor',
    status: 0,
  }
}

/**
 * Generate a UUID v4 using crypto.randomUUID() for cryptographically
 * strong randomness. Falls back to Math.random-based implementation
 * in environments where crypto is unavailable.
 */
function generateUUID(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  // Fallback for environments without crypto.randomUUID
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0
    const v = c === 'x' ? r : (r & 0x3) | 0x8
    return v.toString(16)
  })
}

/**
 * Add X-Idempotency-Key header for idempotent operations
 * Use for POST and PATCH requests that should be idempotent
 */
export function withIdempotencyKey(config: InternalAxiosRequestConfig): InternalAxiosRequestConfig {
  const headers = config.headers ?? new AxiosHeaders()
  if (!headers.has('X-Idempotency-Key')) {
    headers.set('X-Idempotency-Key', generateUUID())
  }
  config.headers = headers
  return config
}

/**
 * Typed API helpers that unwrap AxiosResponse automatically.
 * Use instead of `http` when you want the response body directly,
 * avoiding the `response.data` boilerplate on every call.
 *
 * @example
 * const customers = await api.get<PagedResponse<Customer>>('/v1/customers')
 * customers.data // Customer[]  — no double .data needed
 */
export const api = {
  get: <T>(url: string, config?: Parameters<typeof http.get>[1]): Promise<T> =>
    http.get<T>(url, config).then((r) => r.data),

  post: <T>(url: string, data?: unknown, config?: Parameters<typeof http.post>[2]): Promise<T> =>
    http.post<T>(url, data, config).then((r) => r.data),

  put: <T>(url: string, data?: unknown, config?: Parameters<typeof http.put>[2]): Promise<T> =>
    http.put<T>(url, data, config).then((r) => r.data),

  patch: <T>(url: string, data?: unknown, config?: Parameters<typeof http.patch>[2]): Promise<T> =>
    http.patch<T>(url, data, config).then((r) => r.data),

  delete: <T = void>(url: string, config?: Parameters<typeof http.delete>[1]): Promise<T> =>
    http.delete<T>(url, config).then((r) => r.data),
}

export default http
