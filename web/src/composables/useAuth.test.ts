import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAuth } from './useAuth'

// Mock vue-router
const pushMock = vi.fn()
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: pushMock }),
}))

// Mock http so the auth store doesn't make real requests
vi.mock('@/services/http', () => ({
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
      const http = (await import('@/services/http')).default
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair() })

      const { login, isAuthenticated } = useAuth()
      await login('a@b.com', 'pass')

      expect(isAuthenticated.value).toBe(true)
    })
  })

  describe('login', () => {
    it('calls store.login and navigates to /dashboard', async () => {
      const http = (await import('@/services/http')).default
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
      const http = (await import('@/services/http')).default
      vi.mocked(http.post).mockRejectedValueOnce(new Error('401'))

      const { login } = useAuth()
      await expect(login('a@b.com', 'bad')).rejects.toThrow('401')
      expect(pushMock).not.toHaveBeenCalled()
    })
  })

  describe('logout', () => {
    it('calls store.logout', async () => {
      // Setup: login first
      const http = (await import('@/services/http')).default
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
      const http = (await import('@/services/http')).default
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair('owner') })

      const { login, hasRole } = useAuth()
      await login('a@b.com', 'p')

      expect(hasRole('member')).toBe(true)
      expect(hasRole('admin')).toBe(true)
      expect(hasRole('owner')).toBe(true)
    })

    it('admin has admin and member roles but not owner', async () => {
      const http = (await import('@/services/http')).default
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair('admin') })

      const { login, hasRole } = useAuth()
      await login('a@b.com', 'p')

      expect(hasRole('member')).toBe(true)
      expect(hasRole('admin')).toBe(true)
      expect(hasRole('owner')).toBe(false)
    })

    it('member has only member role', async () => {
      const http = (await import('@/services/http')).default
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair('member') })

      const { login, hasRole } = useAuth()
      await login('a@b.com', 'p')

      expect(hasRole('member')).toBe(true)
      expect(hasRole('admin')).toBe(false)
      expect(hasRole('owner')).toBe(false)
    })
  })
})
