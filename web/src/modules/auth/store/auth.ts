import http from '@/core/api/http'
import { useRBACStore } from '@/core/rbac'
import type { LoginRequest, OrgRef, Role, SelectOrganizationRequest, TokenPair } from '@/modules/auth/types'
import { isMultiOrgLoginResponse } from '@/modules/auth/types'
import { defineStore } from 'pinia'
import { computed, ref, watch } from 'vue'

// localStorage stores only non-sensitive user metadata decoded from the JWT.
// The actual tokens are kept in HttpOnly cookies (set by the backend) and are
// never accessible to JavaScript — this eliminates the XSS token-theft vector.
const AUTH_STORAGE_PREFIX = 'payflow:auth'
const USER_KEY = `${AUTH_STORAGE_PREFIX}:user`

interface AuthUser {
  organizationId: string
  userId?: string
  memberId: string
  role?: Role
  authType: 'user' | 'system'
  // exp is stored so isAuthenticated can check expiry without the raw token
  exp?: number
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

function isUserExpired(u: AuthUser | null): boolean {
  if (!u?.exp) return false
  return Math.floor(Date.now() / 1000) >= u.exp
}

export const useAuthStore = defineStore('auth', () => {
  const user = ref<AuthUser | null>(null)
  const lastError = ref<AuthError | null>(null)

  const pendingSessionToken = ref<string | null>(null)
  const pendingOrganizations = ref<OrgRef[]>([])

  // isAuthenticated is derived from the stored user metadata expiry.
  // The actual token validity is enforced by the backend on every request.
  const isAuthenticated = computed(() => !!user.value && !isUserExpired(user.value))
  const userRole = computed(() => user.value?.role ?? null)
  const isOwner = computed(() => user.value?.role === 'owner')
  const isAdmin = computed(() => user.value?.role === 'admin' || user.value?.role === 'owner')
  const isMultiOrgPending = computed(() => !!pendingSessionToken.value)
  const organizationId = computed(() => user.value?.organizationId ?? null)

  // Decode claims from the access token returned in the response body and
  // persist only the non-sensitive user metadata to localStorage.
  // The raw token is NOT stored — it lives in the HttpOnly cookie only.
  function hydrateUser(accessToken: string) {
    const claims = decodeJwtPayload(accessToken)
    const u: AuthUser = {
      organizationId: claims.sub,
      userId: claims.user_id,
      memberId: claims.member_id ?? '',
      role: claims.role,
      authType: claims.auth_type,
      exp: claims.exp,
    }
    user.value = u
    localStorage.setItem(USER_KEY, JSON.stringify(u))
    lastError.value = null

    scheduleExpiryWarning(claims.exp)
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

      hydrateUser(data.access_token)
      // Fetch effective permissions from /me after successful login.
      await useRBACStore().fetchPermissions()
    } catch (err) {
      const error = new AuthError(
        err instanceof Error ? err.message : 'Login failed',
        'LOGIN_ERROR',
      )
      lastError.value = error
      throw error
    }
  }

  async function selectOrganization(orgId: string): Promise<void> {
    try {
      lastError.value = null
      const sessionToken = pendingSessionToken.value
      if (!sessionToken) {
        throw new AuthError('No pending session token', 'NO_SESSION_TOKEN')
      }

      const { data } = await http.post<TokenPair>('/v1/auth/select-organization', {
        session_token: sessionToken,
        organization_id: orgId,
      } satisfies SelectOrganizationRequest)

      pendingSessionToken.value = null
      pendingOrganizations.value = []

      hydrateUser(data.access_token)
      await useRBACStore().fetchPermissions()
    } catch (err) {
      if (err instanceof AuthError) {
        lastError.value = err
        throw err
      }
      const message = err instanceof Error ? err.message : 'Organization selection failed'
      const error = new AuthError(message, 'ORG_SELECTION_ERROR')
      lastError.value = error
      throw error
    }
  }

  async function logout(): Promise<void> {
    // Ask the backend to expire the HttpOnly cookies. Best-effort — clear
    // local state regardless of whether the request succeeds.
    try {
      await http.post('/v1/auth/logout', {})
    } catch {
      // ignore — local cleanup always runs
    } finally {
      clearLocalState()
    }
  }

  function clearLocalState(): void {
    user.value = null
    pendingSessionToken.value = null
    pendingOrganizations.value = []
    lastError.value = null
    localStorage.removeItem(USER_KEY)
    clearExpiryWarning()
    useRBACStore().$reset()
  }

  async function refreshAccessToken(): Promise<void> {
    try {
      lastError.value = null
      // The backend reads the refresh token from the HttpOnly cookie.
      // No token needed in the body for browser clients.
      const { data } = await http.post<TokenPair>('/v1/auth/refresh', {})
      hydrateUser(data.access_token)
      await useRBACStore().fetchPermissions()
    } catch (err) {
      clearLocalState()
      const error = new AuthError('Token refresh failed', 'REFRESH_ERROR')
      lastError.value = error
      throw error
    }
  }

  // Restore user metadata from localStorage on page load.
  // The actual token is in the HttpOnly cookie — the backend will validate it.
  // If the stored metadata is expired, clear it; the 401 interceptor in http.ts
  // will attempt a silent refresh on the first protected request.
  function loadFromStorage(): void {
    try {
      const raw = localStorage.getItem(USER_KEY)
      if (!raw) return

      const stored = JSON.parse(raw) as AuthUser
      if (isUserExpired(stored)) {
        localStorage.removeItem(USER_KEY)
        return
      }

      user.value = stored
      scheduleExpiryWarning(stored.exp)
      // Hydrate RBAC permissions from /me on page reload.
      useRBACStore().fetchPermissions()
    } catch {
      localStorage.removeItem(USER_KEY)
    }
  }

  // ---- Token expiry warning ----
  // Emits "payflow:tokenExpiring" 5 min before expiry so the UI can notify the user.

  let expiryWarningTimer: ReturnType<typeof setTimeout> | null = null

  function clearExpiryWarning() {
    if (expiryWarningTimer !== null) {
      clearTimeout(expiryWarningTimer)
      expiryWarningTimer = null
    }
  }

  function scheduleExpiryWarning(exp?: number) {
    clearExpiryWarning()
    if (!exp) return

    const expiresAt = new Date(exp * 1000)
    const delay = expiresAt.getTime() - 5 * 60 * 1000 - Date.now()
    if (delay > 0) {
      expiryWarningTimer = setTimeout(() => {
        window.dispatchEvent(new CustomEvent('payflow:tokenExpiring', { detail: { expiresAt } }))
      }, delay)
    }
  }

  // Keep expiry warning in sync when user changes (e.g. after refresh)
  watch(user, (u) => {
    if (!u) clearExpiryWarning()
  })

  return {
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
