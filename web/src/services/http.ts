import type { ApiError, ApiValidationError, TokenPair } from '@/types'
import type { AxiosError, InternalAxiosRequestConfig } from 'axios'
import axios, { AxiosHeaders } from 'axios'

interface RetryableRequestConfig extends InternalAxiosRequestConfig {
  _retry?: boolean
}

const http = axios.create({
  baseURL: '/api',
  headers: { 'Content-Type': 'application/json' },
  // Required so the browser sends HttpOnly auth cookies on every request,
  // including cross-origin requests to the Vite dev proxy.
  withCredentials: true,
})

// ---- Request interceptor ----
// Injects X-Request-ID for traceability and X-CSRF-Token for state-changing
// requests (Double Submit Cookie pattern). The Authorization header is NOT
// injected here — browser clients rely on the HttpOnly cookie; programmatic
// clients (API key) must set the header themselves via the `api` helpers or
// by passing a custom config.
http.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const headers = config.headers ?? new AxiosHeaders()

  if (!headers.has('X-Request-ID')) {
    headers.set('X-Request-ID', generateUUID())
  }

  // Echo the CSRF token cookie back in the header for state-changing requests.
  const method = (config.method ?? '').toUpperCase()
  if (['POST', 'PUT', 'PATCH', 'DELETE'].includes(method)) {
    const csrfToken = getCsrfTokenFromCookie()
    if (csrfToken) {
      headers.set('X-CSRF-Token', csrfToken)
    }
  }

  config.headers = headers
  return config
})

// ---- Response interceptor ----
// Handles 401 with a single silent refresh attempt, then redirects to /login
// on failure. Redirects to /maintenance on 503.
let isRefreshing = false
let refreshPromise: Promise<boolean> | null = null

http.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const config = error.config as RetryableRequestConfig | undefined

    if (error.response?.status === 503) {
      if (typeof window !== 'undefined') window.location.href = '/maintenance'
      return Promise.reject(error)
    }

    const isAuthEndpoint =
      config?.url?.includes('/auth/login') ||
      config?.url?.includes('/auth/refresh') ||
      config?.url?.includes('/auth/logout') ||
      config?.url?.includes('/auth/select-organization')

    if (error.response?.status === 401 && config && !config._retry && !isAuthEndpoint) {
      config._retry = true

      if (!isRefreshing) {
        isRefreshing = true
        refreshPromise = silentRefresh().finally(() => {
          isRefreshing = false
          refreshPromise = null
        })
      }

      const refreshed = await refreshPromise
      if (refreshed) {
        // Retry the original request — the new access token is in the cookie,
        // no need to update any header manually.
        return http(config)
      }

      if (typeof window !== 'undefined') window.location.href = '/login'
    }

    return Promise.reject(normalizeError(error))
  },
)

// Attempts a silent token refresh using the HttpOnly refresh cookie.
// Returns true on success, false on any failure.
async function silentRefresh(): Promise<boolean> {
  try {
    await http.post<TokenPair>('/v1/auth/refresh', {})
    return true
  } catch {
    return false
  }
}

export function normalizeError(error: AxiosError): ApiError {
  if (error.response) {
    const { status, data } = error.response as {
      status: number
      data: { error?: string; errors?: ApiValidationError[] }
    }
    const rawMessage = data?.error || error.message || 'Erro desconhecido'
    const message = typeof rawMessage === 'string' ? rawMessage.slice(0, 500) : 'Erro desconhecido'

    const apiError: ApiError = { message, status }

    if (status === 400 && Array.isArray(data?.errors) && data.errors.length > 0) {
      apiError.validationErrors = {}
      for (const ve of data.errors) {
        if (ve.field && typeof ve.field === 'string' && typeof ve.message === 'string') {
          apiError.validationErrors[ve.field.slice(0, 100)] = ve.message.slice(0, 500)
        }
      }
    }

    return apiError
  }

  return { message: 'Erro de conexão com o servidor', status: 0 }
}

/**
 * Read the non-HttpOnly "csrf_token" cookie set by the server on login.
 * JavaScript can read it to echo back in X-CSRF-Token (Double Submit Cookie).
 */
function getCsrfTokenFromCookie(): string | null {
  const match = document.cookie.split('; ').find((c) => c.startsWith('csrf_token='))
  return match ? decodeURIComponent(match.split('=')[1]) : null
}

function generateUUID(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0
    const v = c === 'x' ? r : (r & 0x3) | 0x8
    return v.toString(16)
  })
}

/**
 * Add X-Idempotency-Key header for idempotent POST/PATCH operations.
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
 *
 * @example
 * const customers = await api.get<PagedResponse<Customer>>('/v1/customers')
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
