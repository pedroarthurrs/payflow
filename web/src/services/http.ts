import axios from 'axios'
import type { AxiosError, InternalAxiosRequestConfig } from 'axios'
import type { ApiError, ApiValidationError, TokenPair } from '@/types'

interface RetryableRequestConfig extends InternalAxiosRequestConfig {
  _retry?: boolean
}

const http = axios.create({
  baseURL: '/api',
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor: inject Authorization header
http.interceptors.request.use((config) => {
  const token = localStorage.getItem('access_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

let isRefreshing = false
let refreshPromise: Promise<boolean> | null = null

async function refreshToken(): Promise<boolean> {
  const refresh = localStorage.getItem('refresh_token')
  if (!refresh) return false

  try {
    const { data } = await axios.post<TokenPair>('/api/v1/auth/refresh', {
      refresh_token: refresh,
    })
    localStorage.setItem('access_token', data.access_token)
    localStorage.setItem('refresh_token', data.refresh_token)
    return true
  } catch {
    return false
  }
}

// Response interceptor: auto-refresh on 401
http.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const config = error.config as RetryableRequestConfig | undefined
    if (error.response?.status === 401 && config && !config._retry) {
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
        const newToken = localStorage.getItem('access_token')
        config.headers.Authorization = `Bearer ${newToken}`
        return http(config)
      }

      // Refresh failed — clear session and redirect
      localStorage.removeItem('access_token')
      localStorage.removeItem('refresh_token')
      window.location.href = '/login'
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
    const message = data?.error || error.message || 'Erro desconhecido'

    const apiError: ApiError = { message, status }

    if (status === 400 && Array.isArray(data?.errors) && data.errors.length > 0) {
      apiError.validationErrors = {}
      for (const ve of data.errors) {
        apiError.validationErrors[ve.field] = ve.message
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

export default http
