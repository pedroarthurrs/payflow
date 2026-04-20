import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useAuth } from './useAuth'

// Mock vue-router
const pushMock = vi.fn()
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: pushMock }),
}))

// Mock http so the auth store doesn't make real requests
vi.mock('@/api/http', () => ({
  default: { post: vi.fn() },
}))

// Helper: build a fake JWT
function fakeJwt(payload: Record<string, unknown>): string {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }))
  const body = btoa(JSON.stringify(payload))
  return `${header}.${body}.fake-signature`
}

function makeTokenPair(role = 'admin') {
  return {
    access_token: fakeJwt({ sub: 'org-1', role, auth_type: 'user', member_id: 'member-1' }),
    refresh_token: 'refresh-abc',
    access_token_expires_at: new Date().toISOString(),
    refresh_token_expires_at: new Date().toISOString(),
  }
}

function makeMultiOrgResponse() {
  return {
    session_token: 'session-token-xyz',
    organizations: [
      { organization_id: 'org-1', name: 'Empresa A', role: 'owner' },
      { organization_id: 'org-2', name: 'Empresa B', role: 'member' },
    ],
  }
}

describe('useAuth', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('isAuthenticated', () => {
    it('returns false when not logged in', () => {
      const { isAuthenticated } = useAuth()
      expect(isAuthenticated.value).toBe(false)
    })

    it('returns true after single-org login', async () => {
      const http = (await import('@/core/api/http')).default
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair() })

      const { login, isAuthenticated } = useAuth()
      await login('a@b.com', 'pass')

      expect(isAuthenticated.value).toBe(true)
    })
  })

  describe('login — single org', () => {
    it('calls store.login and navigates to /dashboard', async () => {
      const http = (await import('@/core/api/http')).default
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair() })

      const { login } = useAuth()
      await login('user@test.com', 'secret')

      expect(http.post).toHaveBeenCalledWith('/v1/auth/login', {
        email: 'user@test.com',
        password: 'secret',
      })
      expect(pushMock).toHaveBeenCalledWith('/dashboard')
    })

    it('does not navigate if store.login throws', async () => {
      const http = (await import('@/core/api/http')).default
      vi.mocked(http.post).mockRejectedValueOnce(new Error('401'))

      const { login } = useAuth()
      await expect(login('a@b.com', 'bad')).rejects.toThrow('401')
      expect(pushMock).not.toHaveBeenCalled()
    })
  })

  describe('login — multi org', () => {
    it('navigates to /select-organization instead of /dashboard', async () => {
      const http = (await import('@/core/api/http')).default
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeMultiOrgResponse() })

      const { login, isMultiOrgPending, pendingOrganizations } = useAuth()
      await login('user@test.com', 'secret')

      expect(pushMock).toHaveBeenCalledWith('/select-organization')
      expect(pushMock).not.toHaveBeenCalledWith('/dashboard')
      expect(isMultiOrgPending.value).toBe(true)
      expect(pendingOrganizations.value).toHaveLength(2)
    })
  })

  describe('selectOrganization', () => {
    it('completes auth and navigates to /dashboard', async () => {
      const http = (await import('@/core/api/http')).default
      vi.mocked(http.post)
        .mockResolvedValueOnce({ data: makeMultiOrgResponse() })
        .mockResolvedValueOnce({ data: makeTokenPair('owner') })

      const { login, selectOrganization, isAuthenticated } = useAuth()
      await login('user@test.com', 'secret')

      pushMock.mockClear()
      await selectOrganization('org-1')

      expect(http.post).toHaveBeenLastCalledWith('/v1/auth/select-organization', {
        session_token: 'session-token-xyz',
        organization_id: 'org-1',
      })
      expect(pushMock).toHaveBeenCalledWith('/dashboard')
      expect(isAuthenticated.value).toBe(true)
    })
  })

  describe('logout', () => {
    it('calls store.logout', async () => {
      // Setup: login first
      const http = (await import('@/core/api/http')).default
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair() })

      const { login, logout } = useAuth()
      await login('a@b.com', 'pass')

      // Mock window.location for store.logout redirect
      delete (window as any).location
      window.location = { href: '' } as any

      logout()

      expect(localStorage.getItem('access_token')).toBeNull()
      expect(localStorage.getItem('refresh_token')).toBeNull()
    })
  })

  describe('hasRole', () => {
    it('returns false when no user is logged in', () => {
      const { hasRole } = useAuth()
      expect(hasRole('member')).toBe(false)
    })

    it('owner has all roles', async () => {
      const http = (await import('@/core/api/http')).default
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair('owner') })

      const { login, hasRole } = useAuth()
      await login('a@b.com', 'p')

      expect(hasRole('member')).toBe(true)
      expect(hasRole('admin')).toBe(true)
      expect(hasRole('owner')).toBe(true)
    })

    it('admin has admin and member roles but not owner', async () => {
      const http = (await import('@/core/api/http')).default
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair('admin') })

      const { login, hasRole } = useAuth()
      await login('a@b.com', 'p')

      expect(hasRole('member')).toBe(true)
      expect(hasRole('admin')).toBe(true)
      expect(hasRole('owner')).toBe(false)
    })

    it('member has only member role', async () => {
      const http = (await import('@/core/api/http')).default
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair('member') })

      const { login, hasRole } = useAuth()
      await login('a@b.com', 'p')

      expect(hasRole('member')).toBe(true)
      expect(hasRole('admin')).toBe(false)
      expect(hasRole('owner')).toBe(false)
    })
  })
})


// Mock vue-router
const pushMock = vi.fn()
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: pushMock }),
}))

// Mock http so the auth store doesn't make real requests
vi.mock('@/api/http', () => ({
  default: { post: vi.fn() },
}))

// Helper: build a fake JWT
function fakeJwt(payload: Record<string, unknown>): string {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }))
  const body = btoa(JSON.stringify(payload))
  return `${header}.${body}.fake-signature`
}

function makeTokenPair(role = 'admin') {
  return {
    access_token: fakeJwt({ sub: 'org-1', role, auth_type: 'user' }),
    refresh_token: 'refresh-abc',
    access_token_expires_at: new Date().toISOString(),
    refresh_token_expires_at: new Date().toISOString(),
  }
}

describe('useAuth', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('isAuthenticated', () => {
    it('returns false when not logged in', () => {
      const { isAuthenticated } = useAuth()
      expect(isAuthenticated.value).toBe(false)
    })

    it('returns true after login', async () => {
      const http = (await import('@/core/api/http')).default
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair() })

      const { login, isAuthenticated } = useAuth()
      await login('a@b.com', 'pass')

      expect(isAuthenticated.value).toBe(true)
    })
  })

  describe('login', () => {
    it('calls store.login and navigates to /dashboard', async () => {
      const http = (await import('@/core/api/http')).default
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair() })

      const { login } = useAuth()
      await login('user@test.com', 'secret')

      expect(http.post).toHaveBeenCalledWith('/v1/auth/login', {
        email: 'user@test.com',
        password: 'secret',
      })
      expect(pushMock).toHaveBeenCalledWith('/dashboard')
    })

    it('does not navigate if store.login throws', async () => {
      const http = (await import('@/core/api/http')).default
      vi.mocked(http.post).mockRejectedValueOnce(new Error('401'))

      const { login } = useAuth()
      await expect(login('a@b.com', 'bad')).rejects.toThrow('401')
      expect(pushMock).not.toHaveBeenCalled()
    })
  })

  describe('logout', () => {
    it('calls store.logout', async () => {
      // Setup: login first
      const http = (await import('@/core/api/http')).default
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair() })

      const { login, logout } = useAuth()
      await login('a@b.com', 'pass')

      // Mock window.location for store.logout redirect
      delete (window as any).location
      window.location = { href: '' } as any

      logout()

      expect(localStorage.getItem('access_token')).toBeNull()
      expect(localStorage.getItem('refresh_token')).toBeNull()
    })
  })

  describe('hasRole', () => {
    it('returns false when no user is logged in', () => {
      const { hasRole } = useAuth()
      expect(hasRole('member')).toBe(false)
    })

    it('owner has all roles', async () => {
      const http = (await import('@/core/api/http')).default
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair('owner') })

      const { login, hasRole } = useAuth()
      await login('a@b.com', 'p')

      expect(hasRole('member')).toBe(true)
      expect(hasRole('admin')).toBe(true)
      expect(hasRole('owner')).toBe(true)
    })

    it('admin has admin and member roles but not owner', async () => {
      const http = (await import('@/core/api/http')).default
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair('admin') })

      const { login, hasRole } = useAuth()
      await login('a@b.com', 'p')

      expect(hasRole('member')).toBe(true)
      expect(hasRole('admin')).toBe(true)
      expect(hasRole('owner')).toBe(false)
    })

    it('member has only member role', async () => {
      const http = (await import('@/core/api/http')).default
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair('member') })

      const { login, hasRole } = useAuth()
      await login('a@b.com', 'p')

      expect(hasRole('member')).toBe(true)
      expect(hasRole('admin')).toBe(false)
      expect(hasRole('owner')).toBe(false)
    })
  })
})
