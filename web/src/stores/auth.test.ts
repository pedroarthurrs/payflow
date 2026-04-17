import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useAuthStore } from './auth'

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

// Mock the http module
vi.mock('@/services/http', () => ({
  default: {
    post: vi.fn(),
  },
}))

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
    it('starts with null tokens and user', () => {
      const store = useAuthStore()
      expect(store.accessToken).toBeNull()
      expect(store.refreshToken).toBeNull()
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
    it('stores tokens and decodes user from JWT', async () => {
      const tokens = makeTokenPair({ role: 'owner' })
      const http = (await import('@/services/http')).default
      vi.mocked(http.post).mockResolvedValueOnce({ data: tokens })

      const store = useAuthStore()
      await store.login('test@example.com', 'password123')

      expect(http.post).toHaveBeenCalledWith('/v1/auth/login', {
        email: 'test@example.com',
        password: 'password123',
      })
      expect(store.accessToken).toBe(tokens.access_token)
      expect(store.refreshToken).toBe(tokens.refresh_token)
      expect(store.user).toEqual({
        organizationId: 'org-123',
        memberId: 'member-789',
        role: 'owner',
        authType: 'user',
      })
      expect(localStorage.getItem('access_token')).toBe(tokens.access_token)
      expect(localStorage.getItem('refresh_token')).toBe(tokens.refresh_token)
    })

    it('sets isAuthenticated to true after login', async () => {
      const tokens = makeTokenPair()
      const http = (await import('@/services/http')).default
      vi.mocked(http.post).mockResolvedValueOnce({ data: tokens })

      const store = useAuthStore()
      await store.login('test@example.com', 'pass')

      expect(store.isAuthenticated).toBe(true)
    })

    it('does not set isMultiOrgPending for single-org login', async () => {
      const tokens = makeTokenPair()
      const http = (await import('@/services/http')).default
      vi.mocked(http.post).mockResolvedValueOnce({ data: tokens })

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
      const http = (await import('@/services/http')).default
      vi.mocked(http.post).mockResolvedValueOnce({ data: multiOrg })

      const store = useAuthStore()
      await store.login('test@example.com', 'password123')

      expect(store.isMultiOrgPending).toBe(true)
      expect(store.pendingSessionToken).toBe('session-token-xyz')
      expect(store.pendingOrganizations).toHaveLength(2)
      expect(store.pendingOrganizations[0].name).toBe('Empresa A')
    })

    it('does not set tokens or authenticate during multi-org login', async () => {
      const multiOrg = makeMultiOrgResponse()
      const http = (await import('@/services/http')).default
      vi.mocked(http.post).mockResolvedValueOnce({ data: multiOrg })

      const store = useAuthStore()
      await store.login('test@example.com', 'pass')

      expect(store.isAuthenticated).toBe(false)
      expect(store.accessToken).toBeNull()
    })
  })

  describe('selectOrganization', () => {
    it('completes auth after org selection', async () => {
      // Step 1: multi-org login
      const multiOrg = makeMultiOrgResponse([{ organization_id: 'org-2', name: 'Empresa B', role: 'admin' }])
      const tokens = makeTokenPair({ role: 'admin' })
      const http = (await import('@/services/http')).default
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
    it('clears state and localStorage', async () => {
      // Setup: login first
      const tokens = makeTokenPair()
      const http = (await import('@/services/http')).default
      vi.mocked(http.post).mockResolvedValueOnce({ data: tokens })

      const store = useAuthStore()
      await store.login('test@example.com', 'pass')

      delete (window as any).location
      window.location = { href: '' } as any

      store.logout()

      expect(store.accessToken).toBeNull()
      expect(store.refreshToken).toBeNull()
      expect(store.user).toBeNull()
      expect(store.pendingSessionToken).toBeNull()
      expect(store.pendingOrganizations).toHaveLength(0)
      expect(localStorage.getItem('access_token')).toBeNull()
      expect(localStorage.getItem('refresh_token')).toBeNull()
      expect(store.isAuthenticated).toBe(false)
    })

    it('clears pending multi-org state on logout', async () => {
      const multiOrg = makeMultiOrgResponse()
      const http = (await import('@/services/http')).default
      vi.mocked(http.post).mockResolvedValueOnce({ data: multiOrg })

      const store = useAuthStore()
      await store.login('test@example.com', 'pass')

      delete (window as any).location
      window.location = { href: '' } as any

      store.logout()

      expect(store.isMultiOrgPending).toBe(false)
      expect(store.pendingOrganizations).toHaveLength(0)
    })
  })

  describe('refresh', () => {
    it('updates tokens after successful refresh', async () => {
      const initialTokens = makeTokenPair({ role: 'admin' })
      const newTokens = makeTokenPair({ role: 'owner' })

      const http = (await import('@/services/http')).default
      vi.mocked(http.post)
        .mockResolvedValueOnce({ data: initialTokens })
        .mockResolvedValueOnce({ data: newTokens })

      const store = useAuthStore()
      await store.login('test@example.com', 'pass')
      await store.refresh()

      expect(store.accessToken).toBe(newTokens.access_token)
      expect(store.user?.role).toBe('owner')
      expect(http.post).toHaveBeenLastCalledWith('/v1/auth/refresh', {
        refresh_token: initialTokens.refresh_token,
      })
    })

    it('throws when no refresh token is available', async () => {
      const store = useAuthStore()
      await expect(store.refresh()).rejects.toThrow('No refresh token available')
    })
  })

  describe('loadFromStorage', () => {
    it('restores state from localStorage including member_id', () => {
      const token = fakeJwt({
        sub: 'org-789',
        role: 'member',
        auth_type: 'user',
        member_id: 'member-111',
      })
      localStorage.setItem('access_token', token)
      localStorage.setItem('refresh_token', 'stored-refresh')

      const store = useAuthStore()
      store.loadFromStorage()

      expect(store.accessToken).toBe(token)
      expect(store.refreshToken).toBe('stored-refresh')
      expect(store.isAuthenticated).toBe(true)
      expect(store.user).toEqual({
        organizationId: 'org-789',
        memberId: 'member-111',
        role: 'member',
        authType: 'user',
      })
    })

    it('falls back to empty memberId when member_id is absent in token', () => {
      const token = fakeJwt({ sub: 'org-789', role: 'member', auth_type: 'user' })
      localStorage.setItem('access_token', token)

      const store = useAuthStore()
      store.loadFromStorage()

      expect(store.user?.memberId).toBe('')
    })

    it('does nothing when no token in localStorage', () => {
      const store = useAuthStore()
      store.loadFromStorage()

      expect(store.accessToken).toBeNull()
      expect(store.isAuthenticated).toBe(false)
    })
  })

  describe('getters', () => {
    it('userRole returns the role from user', async () => {
      const tokens = makeTokenPair({ role: 'member' })
      const http = (await import('@/services/http')).default
      vi.mocked(http.post).mockResolvedValueOnce({ data: tokens })

      const store = useAuthStore()
      await store.login('test@example.com', 'pass')

      expect(store.userRole).toBe('member')
    })

    it('isOwner is true only for owner role', async () => {
      const http = (await import('@/services/http')).default

      // owner
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair({ role: 'owner' }) })
      const store = useAuthStore()
      await store.login('a@b.com', 'p')
      expect(store.isOwner).toBe(true)

      // admin — need fresh store
      setActivePinia(createPinia())
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair({ role: 'admin' }) })
      const store2 = useAuthStore()
      await store2.login('a@b.com', 'p')
      expect(store2.isOwner).toBe(false)
    })

    it('isAdmin is true for admin and owner roles', async () => {
      const http = (await import('@/services/http')).default

      // admin
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair({ role: 'admin' }) })
      const store = useAuthStore()
      await store.login('a@b.com', 'p')
      expect(store.isAdmin).toBe(true)

      // owner
      setActivePinia(createPinia())
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair({ role: 'owner' }) })
      const store2 = useAuthStore()
      await store2.login('a@b.com', 'p')
      expect(store2.isAdmin).toBe(true)

      // member
      setActivePinia(createPinia())
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair({ role: 'member' }) })
      const store3 = useAuthStore()
      await store3.login('a@b.com', 'p')
      expect(store3.isAdmin).toBe(false)
    })
  })
})


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

// Mock the http module
vi.mock('@/services/http', () => ({
  default: {
    post: vi.fn(),
  },
}))

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
    it('starts with null tokens and user', () => {
      const store = useAuthStore()
      expect(store.accessToken).toBeNull()
      expect(store.refreshToken).toBeNull()
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
  })

  describe('login', () => {
    it('stores tokens and decodes user from JWT', async () => {
      const tokens = makeTokenPair({ role: 'owner' })
      const http = (await import('@/services/http')).default
      vi.mocked(http.post).mockResolvedValueOnce({ data: tokens })

      const store = useAuthStore()
      await store.login('test@example.com', 'password123')

      expect(http.post).toHaveBeenCalledWith('/v1/auth/login', {
        email: 'test@example.com',
        password: 'password123',
      })
      expect(store.accessToken).toBe(tokens.access_token)
      expect(store.refreshToken).toBe(tokens.refresh_token)
      expect(store.user).toEqual({
        organizationId: 'org-123',
        role: 'owner',
        authType: 'user',
      })
      expect(localStorage.getItem('access_token')).toBe(tokens.access_token)
      expect(localStorage.getItem('refresh_token')).toBe(tokens.refresh_token)
    })

    it('sets isAuthenticated to true after login', async () => {
      const tokens = makeTokenPair()
      const http = (await import('@/services/http')).default
      vi.mocked(http.post).mockResolvedValueOnce({ data: tokens })

      const store = useAuthStore()
      await store.login('test@example.com', 'pass')

      expect(store.isAuthenticated).toBe(true)
    })
  })

  describe('logout', () => {
    it('clears state and localStorage', async () => {
      // Setup: login first
      const tokens = makeTokenPair()
      const http = (await import('@/services/http')).default
      vi.mocked(http.post).mockResolvedValueOnce({ data: tokens })

      const store = useAuthStore()
      await store.login('test@example.com', 'pass')

      // Mock window.location.href
      const locationSpy = vi.spyOn(window, 'location', 'get').mockReturnValue({
        ...window.location,
        href: '',
      } as Location)
      const hrefSetter = vi.fn()
      Object.defineProperty(window, 'location', {
        get: () => ({ href: '' }),
        set: () => { },
        configurable: true,
      })
      // Use a simpler approach: just spy on the assignment
      delete (window as any).location
      window.location = { href: '' } as any

      store.logout()

      expect(store.accessToken).toBeNull()
      expect(store.refreshToken).toBeNull()
      expect(store.user).toBeNull()
      expect(localStorage.getItem('access_token')).toBeNull()
      expect(localStorage.getItem('refresh_token')).toBeNull()
      expect(store.isAuthenticated).toBe(false)

      // Restore
      locationSpy?.mockRestore()
    })
  })

  describe('refresh', () => {
    it('updates tokens after successful refresh', async () => {
      const initialTokens = makeTokenPair({ role: 'admin' })
      const newTokens = makeTokenPair({ role: 'owner' })

      const http = (await import('@/services/http')).default
      vi.mocked(http.post)
        .mockResolvedValueOnce({ data: initialTokens })
        .mockResolvedValueOnce({ data: newTokens })

      const store = useAuthStore()
      await store.login('test@example.com', 'pass')
      await store.refresh()

      expect(store.accessToken).toBe(newTokens.access_token)
      expect(store.user?.role).toBe('owner')
      expect(http.post).toHaveBeenLastCalledWith('/v1/auth/refresh', {
        refresh_token: initialTokens.refresh_token,
      })
    })

    it('throws when no refresh token is available', async () => {
      const store = useAuthStore()
      await expect(store.refresh()).rejects.toThrow('No refresh token available')
    })
  })

  describe('loadFromStorage', () => {
    it('restores state from localStorage', () => {
      const token = fakeJwt({
        sub: 'org-789',
        role: 'member',
        auth_type: 'user',
      })
      localStorage.setItem('access_token', token)
      localStorage.setItem('refresh_token', 'stored-refresh')

      const store = useAuthStore()
      store.loadFromStorage()

      expect(store.accessToken).toBe(token)
      expect(store.refreshToken).toBe('stored-refresh')
      expect(store.isAuthenticated).toBe(true)
      expect(store.user).toEqual({
        organizationId: 'org-789',
        role: 'member',
        authType: 'user',
      })
    })

    it('does nothing when no token in localStorage', () => {
      const store = useAuthStore()
      store.loadFromStorage()

      expect(store.accessToken).toBeNull()
      expect(store.isAuthenticated).toBe(false)
    })
  })

  describe('getters', () => {
    it('userRole returns the role from user', async () => {
      const tokens = makeTokenPair({ role: 'member' })
      const http = (await import('@/services/http')).default
      vi.mocked(http.post).mockResolvedValueOnce({ data: tokens })

      const store = useAuthStore()
      await store.login('test@example.com', 'pass')

      expect(store.userRole).toBe('member')
    })

    it('isOwner is true only for owner role', async () => {
      const http = (await import('@/services/http')).default

      // owner
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair({ role: 'owner' }) })
      const store = useAuthStore()
      await store.login('a@b.com', 'p')
      expect(store.isOwner).toBe(true)

      // admin — need fresh store
      setActivePinia(createPinia())
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair({ role: 'admin' }) })
      const store2 = useAuthStore()
      await store2.login('a@b.com', 'p')
      expect(store2.isOwner).toBe(false)
    })

    it('isAdmin is true for admin and owner roles', async () => {
      const http = (await import('@/services/http')).default

      // admin
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair({ role: 'admin' }) })
      const store = useAuthStore()
      await store.login('a@b.com', 'p')
      expect(store.isAdmin).toBe(true)

      // owner
      setActivePinia(createPinia())
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair({ role: 'owner' }) })
      const store2 = useAuthStore()
      await store2.login('a@b.com', 'p')
      expect(store2.isAdmin).toBe(true)

      // member
      setActivePinia(createPinia())
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair({ role: 'member' }) })
      const store3 = useAuthStore()
      await store3.login('a@b.com', 'p')
      expect(store3.isAdmin).toBe(false)
    })
  })
})
