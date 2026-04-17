import http from '@/services/http'
import type { LoginRequest, OrgRef, Role, SelectOrganizationRequest, TokenPair } from '@/types'
import { isMultiOrgLoginResponse } from '@/types'
import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

interface AuthUser {
  organizationId: string
  memberId: string
  role: Role
  authType: string
}

interface JwtPayload {
  sub: string
  role: Role
  auth_type: string
  user_id?: string
  member_id?: string
  token_type?: string
}

function decodeJwtPayload(token: string): JwtPayload {
  const parts = token.split('.')
  if (parts.length !== 3) {
    throw new Error('Invalid JWT format')
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
}

export const useAuthStore = defineStore('auth', () => {
  const accessToken = ref<string | null>(null)
  const refreshToken = ref<string | null>(null)
  const user = ref<AuthUser | null>(null)

  /** Populated during multi-org login; cleared after selectOrganization(). */
  const pendingSessionToken = ref<string | null>(null)
  const pendingOrganizations = ref<OrgRef[]>([])

  const isAuthenticated = computed(() => !!accessToken.value)
  const userRole = computed(() => user.value?.role ?? null)
  const isOwner = computed(() => user.value?.role === 'owner')
  const isAdmin = computed(() => user.value?.role === 'admin' || user.value?.role === 'owner')
  const isMultiOrgPending = computed(() => !!pendingSessionToken.value)

  function setTokens(access: string, refresh: string) {
    accessToken.value = access
    refreshToken.value = refresh
    localStorage.setItem('access_token', access)
    localStorage.setItem('refresh_token', refresh)

    const claims = decodeJwtPayload(access)
    user.value = {
      organizationId: claims.sub,
      memberId: claims.member_id ?? '',
      role: claims.role,
      authType: claims.auth_type,
    }
  }

  async function login(email: string, password: string) {
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
  }

  async function selectOrganization(organizationId: string) {
    const sessionToken = pendingSessionToken.value
    if (!sessionToken) throw new Error('No pending session token')

    const { data } = await http.post<TokenPair>('/v1/auth/select-organization', {
      session_token: sessionToken,
      organization_id: organizationId,
    } satisfies SelectOrganizationRequest)

    pendingSessionToken.value = null
    pendingOrganizations.value = []

    setTokens(data.access_token, data.refresh_token)
  }

  function logout() {
    accessToken.value = null
    refreshToken.value = null
    user.value = null
    pendingSessionToken.value = null
    pendingOrganizations.value = []
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
    window.location.href = '/login'
  }

  async function refresh() {
    const currentRefresh = refreshToken.value ?? localStorage.getItem('refresh_token')
    if (!currentRefresh) throw new Error('No refresh token available')

    const { data } = await http.post<TokenPair>('/v1/auth/refresh', {
      refresh_token: currentRefresh,
    })

    setTokens(data.access_token, data.refresh_token)
  }

  function loadFromStorage() {
    const storedAccess = localStorage.getItem('access_token')
    const storedRefresh = localStorage.getItem('refresh_token')

    if (storedAccess) {
      accessToken.value = storedAccess
      refreshToken.value = storedRefresh

      try {
        const claims = decodeJwtPayload(storedAccess)
        user.value = {
          organizationId: claims.sub,
          memberId: claims.member_id ?? '',
          role: claims.role,
          authType: claims.auth_type,
        }
      } catch {
        // Corrupted token — clear everything
        logout()
      }
    }
  }

  return {
    accessToken,
    refreshToken,
    user,
    pendingSessionToken,
    pendingOrganizations,
    isAuthenticated,
    isMultiOrgPending,
    userRole,
    isOwner,
    isAdmin,
    login,
    selectOrganization,
    logout,
    refresh,
    loadFromStorage,
  }
})
