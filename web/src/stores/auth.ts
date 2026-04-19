import http from '@/services/http'
import type { LoginRequest, OrgRef, Role, SelectOrganizationRequest, TokenPair } from '@/types'
import { isMultiOrgLoginResponse } from '@/types'
import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

// localStorage namespace to prevent key collisions
const AUTH_STORAGE_PREFIX = 'i9pay:auth'
const ACCESS_TOKEN_KEY = `${AUTH_STORAGE_PREFIX}:accessToken`
const REFRESH_TOKEN_KEY = `${AUTH_STORAGE_PREFIX}:refreshToken`

interface AuthUser {
  organizationId: string
  userId?: string
  memberId: string
  role?: Role
  authType: 'user' | 'system'
}

interface JwtPayload {
  sub: string
  role?: Role
  auth_type: 'user' | 'system'
  user_id?: string
  member_id?: string
  token_type?: string
  iat?: number
  exp?: number
}

export class AuthError extends Error {
  constructor(message: string, public code?: string) {
    super(message)
    this.name = 'AuthError'
  }
}

function decodeJwtPayload(token: string): JwtPayload {
  try {
    const parts = token.split('.')
    if (parts.length !== 3) {
      throw new AuthError('Invalid JWT format: expected 3 parts', 'INVALID_JWT_FORMAT')
    }
    const base64Url = parts[1]
    const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/')
    const json = decodeURIComponent(
      atob(base64)
        .split('')
        .map((c) => '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2))
        .join(''),
    )
    return JSON.parse(json)
  } catch (err) {
    if (err instanceof AuthError) throw err
    throw new AuthError('Failed to decode JWT payload', 'JWT_DECODE_ERROR')
  }
}

function isTokenExpired(token: string): boolean {
  try {
    const payload = decodeJwtPayload(token)
    if (!payload.exp) return false
    const now = Math.floor(Date.now() / 1000)
    return payload.exp < now
  } catch {
    return true
  }
}

export const useAuthStore = defineStore('auth', () => {
  const accessToken = ref<string | null>(null)
  const refreshToken = ref<string | null>(null)
  const user = ref<AuthUser | null>(null)
  const lastError = ref<AuthError | null>(null)

  const pendingSessionToken = ref<string | null>(null)
  const pendingOrganizations = ref<OrgRef[]>([])

  const isAuthenticated = computed(() => !!accessToken.value && !isTokenExpired(accessToken.value))
  const userRole = computed(() => user.value?.role ?? null)
  const isOwner = computed(() => user.value?.role === 'owner')
  const isAdmin = computed(() => user.value?.role === 'admin' || user.value?.role === 'owner')
  const isMultiOrgPending = computed(() => !!pendingSessionToken.value)
  const organizationId = computed(() => user.value?.organizationId ?? null)

  function setTokens(access: string, refresh: string) {
    try {
      accessToken.value = access
      refreshToken.value = refresh
      localStorage.setItem(ACCESS_TOKEN_KEY, access)
      localStorage.setItem(REFRESH_TOKEN_KEY, refresh)
      lastError.value = null

      const claims = decodeJwtPayload(access)
      user.value = {
        organizationId: claims.sub,
        userId: claims.user_id,
        memberId: claims.member_id ?? '',
        role: claims.role,
        authType: claims.auth_type,
      }
    } catch (err) {
      const error = err instanceof AuthError ? err : new AuthError('Failed to set tokens')
      lastError.value = error
      logout()
      throw error
    }
  }

  async function login(email: string, password: string): Promise<void> {
    try {
      lastError.value = null
      const { data } = await http.post<TokenPair | { session_token: string; organizations: OrgRef[] }>(
        '/v1/auth/login',
        { email, password } satisfies LoginRequest,
      )

      if (isMultiOrgLoginResponse(data)) {
        pendingSessionToken.value = data.session_token
        pendingOrganizations.value = data.organizations
        return
      }

      setTokens(data.access_token, data.refresh_token)
    } catch (err) {
      const error = new AuthError(
        err instanceof Error ? err.message : 'Login failed',
        'LOGIN_ERROR',
      )
      lastError.value = error
      throw error
    }
  }

  async function selectOrganization(organizationId: string): Promise<void> {
    try {
      lastError.value = null
      const sessionToken = pendingSessionToken.value
      if (!sessionToken) {
        throw new AuthError('No pending session token', 'NO_SESSION_TOKEN')
      }

      const { data } = await http.post<TokenPair>('/v1/auth/select-organization', {
        session_token: sessionToken,
        organization_id: organizationId,
      } satisfies SelectOrganizationRequest)

      pendingSessionToken.value = null
      pendingOrganizations.value = []

      setTokens(data.access_token, data.refresh_token)
    } catch (err) {
      const error = err instanceof AuthError ? err : new AuthError('Organization selection failed')
      lastError.value = error
      throw error
    }
  }

  function logout(): void {
    accessToken.value = null
    refreshToken.value = null
    user.value = null
    pendingSessionToken.value = null
    pendingOrganizations.value = []
    lastError.value = null
    localStorage.removeItem(ACCESS_TOKEN_KEY)
    localStorage.removeItem(REFRESH_TOKEN_KEY)
  }

  async function refreshAccessToken(): Promise<void> {
    try {
      lastError.value = null
      const currentRefresh = refreshToken.value ?? localStorage.getItem(REFRESH_TOKEN_KEY)
      if (!currentRefresh) {
        throw new AuthError('No refresh token available', 'NO_REFRESH_TOKEN')
      }

      const { data } = await http.post<TokenPair>('/v1/auth/refresh', {
        refresh_token: currentRefresh,
      })

      setTokens(data.access_token, data.refresh_token)
    } catch (err) {
      logout()
      const error = new AuthError('Token refresh failed', 'REFRESH_ERROR')
      lastError.value = error
      throw error
    }
  }

  function loadFromStorage(): void {
    try {
      const storedAccess = localStorage.getItem(ACCESS_TOKEN_KEY)
      const storedRefresh = localStorage.getItem(REFRESH_TOKEN_KEY)

      if (!storedAccess) {
        return
      }

      if (isTokenExpired(storedAccess)) {
        localStorage.removeItem(ACCESS_TOKEN_KEY)
        localStorage.removeItem(REFRESH_TOKEN_KEY)
        return
      }

      accessToken.value = storedAccess
      refreshToken.value = storedRefresh

      const claims = decodeJwtPayload(storedAccess)
      user.value = {
        organizationId: claims.sub,
        userId: claims.user_id,
        memberId: claims.member_id ?? '',
        role: claims.role,
        authType: claims.auth_type,
      }
      lastError.value = null
    } catch (err) {
      logout()
      lastError.value = err instanceof AuthError ? err : new AuthError('Failed to load auth from storage')
    }
  }

  return {
    accessToken,
    refreshToken,
    user,
    lastError,
    pendingSessionToken,
    pendingOrganizations,
    isAuthenticated,
    isMultiOrgPending,
    userRole,
    isOwner,
    isAdmin,
    organizationId,
    login,
    selectOrganization,
    logout,
    refreshAccessToken,
    loadFromStorage,
  }
})
