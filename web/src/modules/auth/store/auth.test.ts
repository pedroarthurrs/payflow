import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useAuthStore } from './auth'

// The store keeps tokens in HttpOnly cookies (set by the backend) and persists
// only non-sensitive user metadata — decoded from the access token — to
// localStorage under this key.
const USER_KEY = 'payflow:auth:user'

// Helper: build a fake JWT with the given payload (no signature validation needed)
function fakeJwt(payload: Record<string, unknown>): string {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }))
  const body = btoa(JSON.stringify(payload))
  return `${header}.${body}.fake-signature`
}

const defaultClaims = {
  sub: 'org-123',
  role: 'admin',
  auth_type: 'user',
  user_id: 'user-456',
  member_id: 'member-789',
  token_type: 'access',
}

function makeTokenPair(overrides: Record<string, unknown> = {}) {
  const claims = { ...defaultClaims, ...overrides }
  return {
    access_token: fakeJwt(claims),
    refresh_token: 'refresh-token-abc',
    access_token_expires_at: new Date().toISOString(),
    refresh_token_expires_at: new Date().toISOString(),
  }
}

function makeMultiOrgResponse(orgs = [{ organization_id: 'org-1', name: 'Org 1', role: 'owner' }]) {
  return {
    session_token: 'session-token-xyz',
    organizations: orgs,
  }
}

// Mock the HTTP client the store depends on.
vi.mock('@/core/api/http', () => ({
  default: {
    post: vi.fn(),
  },
}))

// Mock the RBAC store — login/refresh/logout trigger permission side effects
// (fetchPermissions / $reset) that are out of scope for these tests.
vi.mock('@/core/rbac', () => ({
  useRBACStore: () => ({
    fetchPermissions: vi.fn().mockResolvedValue(undefined),
    $reset: vi.fn(),
  }),
}))

async function getHttp() {
  return (await import('@/core/api/http')).default
}

describe('Auth Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('initial state', () => {
    it('starts with no user', () => {
      const store = useAuthStore()
      expect(store.user).toBeNull()
    })

    it('isAuthenticated is false initially', () => {
      const store = useAuthStore()
      expect(store.isAuthenticated).toBe(false)
    })

    it('userRole is null initially', () => {
      const store = useAuthStore()
      expect(store.userRole).toBeNull()
    })

    it('isMultiOrgPending is false initially', () => {
      const store = useAuthStore()
      expect(store.isMultiOrgPending).toBe(false)
    })
  })

  describe('login — single org', () => {
    it('decodes user metadata from the access token and persists it', async () => {
      const tokens = makeTokenPair({ role: 'owner' })
      const http = await getHttp()
      vi.mocked(http.post).mockResolvedValueOnce({ data: tokens })

      const store = useAuthStore()
      await store.login('test@example.com', 'password123')

      expect(http.post).toHaveBeenCalledWith('/v1/auth/login', {
        email: 'test@example.com',
        password: 'password123',
      })
      expect(store.user).toEqual({
        organizationId: 'org-123',
        userId: 'user-456',
        memberId: 'member-789',
        role: 'owner',
        authType: 'user',
      })
      // Only user metadata is persisted — never the raw tokens.
      const stored = JSON.parse(localStorage.getItem(USER_KEY)!)
      expect(stored.organizationId).toBe('org-123')
      expect(localStorage.getItem('access_token')).toBeNull()
      expect(localStorage.getItem('refresh_token')).toBeNull()
    })

    it('sets isAuthenticated to true after login', async () => {
      const http = await getHttp()
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair() })

      const store = useAuthStore()
      await store.login('test@example.com', 'pass')

      expect(store.isAuthenticated).toBe(true)
    })

    it('does not set isMultiOrgPending for single-org login', async () => {
      const http = await getHttp()
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair() })

      const store = useAuthStore()
      await store.login('test@example.com', 'pass')

      expect(store.isMultiOrgPending).toBe(false)
      expect(store.pendingOrganizations).toHaveLength(0)
    })
  })

  describe('login — multi org', () => {
    it('sets pendingSessionToken and pendingOrganizations', async () => {
      const multiOrg = makeMultiOrgResponse([
        { organization_id: 'org-1', name: 'Empresa A', role: 'owner' },
        { organization_id: 'org-2', name: 'Empresa B', role: 'member' },
      ])
      const http = await getHttp()
      vi.mocked(http.post).mockResolvedValueOnce({ data: multiOrg })

      const store = useAuthStore()
      await store.login('test@example.com', 'password123')

      expect(store.isMultiOrgPending).toBe(true)
      expect(store.pendingSessionToken).toBe('session-token-xyz')
      expect(store.pendingOrganizations).toHaveLength(2)
      expect(store.pendingOrganizations[0].name).toBe('Empresa A')
    })

    it('does not authenticate during multi-org login', async () => {
      const http = await getHttp()
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeMultiOrgResponse() })

      const store = useAuthStore()
      await store.login('test@example.com', 'pass')

      expect(store.isAuthenticated).toBe(false)
      expect(store.user).toBeNull()
    })
  })

  describe('selectOrganization', () => {
    it('completes auth after org selection', async () => {
      const multiOrg = makeMultiOrgResponse([{ organization_id: 'org-2', name: 'Empresa B', role: 'admin' }])
      const tokens = makeTokenPair({ role: 'admin' })
      const http = await getHttp()
      vi.mocked(http.post)
        .mockResolvedValueOnce({ data: multiOrg })
        .mockResolvedValueOnce({ data: tokens })

      const store = useAuthStore()
      await store.login('test@example.com', 'pass')
      await store.selectOrganization('org-2')

      expect(http.post).toHaveBeenLastCalledWith('/v1/auth/select-organization', {
        session_token: 'session-token-xyz',
        organization_id: 'org-2',
      })
      expect(store.isAuthenticated).toBe(true)
      expect(store.isMultiOrgPending).toBe(false)
      expect(store.pendingOrganizations).toHaveLength(0)
      expect(store.pendingSessionToken).toBeNull()
    })

    it('throws when called without a pending session token', async () => {
      const store = useAuthStore()
      await expect(store.selectOrganization('org-1')).rejects.toThrow('No pending session token')
    })
  })

  describe('logout', () => {
    it('clears state and persisted user metadata', async () => {
      const http = await getHttp()
      vi.mocked(http.post)
        .mockResolvedValueOnce({ data: makeTokenPair() }) // login
        .mockResolvedValueOnce({ data: {} }) // logout

      const store = useAuthStore()
      await store.login('test@example.com', 'pass')
      await store.logout()

      expect(http.post).toHaveBeenLastCalledWith('/v1/auth/logout', {})
      expect(store.user).toBeNull()
      expect(store.pendingSessionToken).toBeNull()
      expect(store.pendingOrganizations).toHaveLength(0)
      expect(localStorage.getItem(USER_KEY)).toBeNull()
      expect(store.isAuthenticated).toBe(false)
    })

    it('clears local state even when the logout request fails', async () => {
      const http = await getHttp()
      vi.mocked(http.post)
        .mockResolvedValueOnce({ data: makeTokenPair() }) // login
        .mockRejectedValueOnce(new Error('network')) // logout

      const store = useAuthStore()
      await store.login('test@example.com', 'pass')
      await store.logout()

      expect(store.user).toBeNull()
      expect(store.isAuthenticated).toBe(false)
    })
  })

  describe('refreshAccessToken', () => {
    it('re-hydrates the user from the refreshed access token', async () => {
      const initialTokens = makeTokenPair({ role: 'admin' })
      const newTokens = makeTokenPair({ role: 'owner' })

      const http = await getHttp()
      vi.mocked(http.post)
        .mockResolvedValueOnce({ data: initialTokens })
        .mockResolvedValueOnce({ data: newTokens })

      const store = useAuthStore()
      await store.login('test@example.com', 'pass')
      await store.refreshAccessToken()

      // The refresh token lives in the HttpOnly cookie — the body is empty.
      expect(http.post).toHaveBeenLastCalledWith('/v1/auth/refresh', {})
      expect(store.user?.role).toBe('owner')
      expect(store.isAuthenticated).toBe(true)
    })

    it('clears state and throws when refresh fails', async () => {
      const http = await getHttp()
      vi.mocked(http.post)
        .mockResolvedValueOnce({ data: makeTokenPair() }) // login
        .mockRejectedValueOnce(new Error('401')) // refresh

      const store = useAuthStore()
      await store.login('test@example.com', 'pass')

      await expect(store.refreshAccessToken()).rejects.toThrow('Token refresh failed')
      expect(store.user).toBeNull()
      expect(store.isAuthenticated).toBe(false)
    })
  })

  describe('loadFromStorage', () => {
    it('restores user metadata from localStorage', () => {
      const storedUser = {
        organizationId: 'org-789',
        userId: 'user-1',
        memberId: 'member-111',
        role: 'member',
        authType: 'user',
      }
      localStorage.setItem(USER_KEY, JSON.stringify(storedUser))

      const store = useAuthStore()
      store.loadFromStorage()

      expect(store.user).toEqual(storedUser)
      expect(store.isAuthenticated).toBe(true)
    })

    it('discards expired stored metadata', () => {
      const expiredUser = {
        organizationId: 'org-789',
        memberId: 'member-111',
        role: 'member',
        authType: 'user',
        exp: Math.floor(Date.now() / 1000) - 60,
      }
      localStorage.setItem(USER_KEY, JSON.stringify(expiredUser))

      const store = useAuthStore()
      store.loadFromStorage()

      expect(store.user).toBeNull()
      expect(store.isAuthenticated).toBe(false)
      expect(localStorage.getItem(USER_KEY)).toBeNull()
    })

    it('does nothing when no user in localStorage', () => {
      const store = useAuthStore()
      store.loadFromStorage()

      expect(store.user).toBeNull()
      expect(store.isAuthenticated).toBe(false)
    })
  })

  describe('getters', () => {
    it('userRole returns the role from the user', async () => {
      const http = await getHttp()
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair({ role: 'member' }) })

      const store = useAuthStore()
      await store.login('test@example.com', 'pass')

      expect(store.userRole).toBe('member')
    })

    it('isOwner is true only for the owner role', async () => {
      const http = await getHttp()

      vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair({ role: 'owner' }) })
      const store = useAuthStore()
      await store.login('a@b.com', 'p')
      expect(store.isOwner).toBe(true)

      setActivePinia(createPinia())
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair({ role: 'admin' }) })
      const store2 = useAuthStore()
      await store2.login('a@b.com', 'p')
      expect(store2.isOwner).toBe(false)
    })

    it('isAdmin is true for admin and owner roles', async () => {
      const http = await getHttp()

      vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair({ role: 'admin' }) })
      const store = useAuthStore()
      await store.login('a@b.com', 'p')
      expect(store.isAdmin).toBe(true)

      setActivePinia(createPinia())
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair({ role: 'owner' }) })
      const store2 = useAuthStore()
      await store2.login('a@b.com', 'p')
      expect(store2.isAdmin).toBe(true)

      setActivePinia(createPinia())
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair({ role: 'member' }) })
      const store3 = useAuthStore()
      await store3.login('a@b.com', 'p')
      expect(store3.isAdmin).toBe(false)
    })
  })
})
