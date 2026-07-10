import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useAuth } from './useAuth'

// Mock vue-router — useAuth navigates on login/logout.
const pushMock = vi.fn()
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: pushMock }),
}))

// Mock the HTTP client so the auth store makes no real requests.
vi.mock('@/core/api/http', () => ({
  default: { post: vi.fn() },
}))

// Mock the RBAC store: it is used both directly by useAuth (hasRole/can) and
// internally by the auth store (fetchPermissions/$reset on login/logout).
const rbacHasRole = vi.fn()
vi.mock('@/core/rbac', () => ({
  useRBACStore: () => ({
    hasRole: rbacHasRole,
    can: vi.fn(),
    canAny: vi.fn(),
    canAll: vi.fn(),
    fetchPermissions: vi.fn().mockResolvedValue(undefined),
    $reset: vi.fn(),
  }),
}))

// Helper: build a fake JWT (no signature validation needed).
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
      { organization_id: 'org-2', name: 'Empresa B', role: 'admin' },
    ],
  }
}

async function getHttp() {
  return (await import('@/core/api/http')).default
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
      const http = await getHttp()
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair() })

      const { login, isAuthenticated } = useAuth()
      await login('a@b.com', 'pass')

      expect(isAuthenticated.value).toBe(true)
    })
  })

  describe('login — single org', () => {
    it('calls store.login and navigates to /dashboard', async () => {
      const http = await getHttp()
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
      const http = await getHttp()
      vi.mocked(http.post).mockRejectedValueOnce(new Error('401'))

      const { login } = useAuth()
      await expect(login('a@b.com', 'bad')).rejects.toThrow()
      expect(pushMock).not.toHaveBeenCalled()
    })
  })

  describe('login — multi org', () => {
    it('navigates to /select-organization instead of /dashboard', async () => {
      const http = await getHttp()
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
      const http = await getHttp()
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
    it('navigates to /login', async () => {
      const http = await getHttp()
      vi.mocked(http.post).mockResolvedValue({ data: makeTokenPair() })

      const { login, logout } = useAuth()
      await login('a@b.com', 'pass')

      logout()

      expect(pushMock).toHaveBeenCalledWith('/login')
    })
  })

  describe('hasRole', () => {
    it('delegates to the RBAC store', () => {
      rbacHasRole.mockReturnValue(true)

      const { hasRole } = useAuth()

      expect(hasRole('admin')).toBe(true)
      expect(rbacHasRole).toHaveBeenCalledWith('admin')
    })

    it('returns false when the RBAC store denies the role', () => {
      rbacHasRole.mockReturnValue(false)

      const { hasRole } = useAuth()

      expect(hasRole('owner')).toBe(false)
    })
  })
})
