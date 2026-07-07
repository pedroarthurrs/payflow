import { useAuthStore } from '@/modules/auth/store/auth'
import { useStatusStore } from '@/modules/auth/store/status'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import router from './index'

function authenticateAs(
  store: ReturnType<typeof useAuthStore>,
  role: 'owner' | 'admin' | 'member',
) {
  store.$patch({
    accessToken: 'fake-token',
    refreshToken: 'refresh-token',
    user: { organizationId: 'org-1', role, authType: 'user' },
  })
}

describe('Router', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    // Stub status check so guestOnly routes don't make real HTTP calls
    const statusStore = useStatusStore()
    vi.spyOn(statusStore, 'check').mockResolvedValue()
    // Reset router to a known state
    router.currentRoute.value.path !== '/' && router.push('/')
  })

  describe('route definitions', () => {
    it('has all expected routes', () => {
      const paths = router.getRoutes().map((r) => r.path)
      expect(paths).toContain('/login')
      expect(paths).toContain('/dashboard')
      expect(paths).toContain('/customers')
      expect(paths).toContain('/customers/new')
      expect(paths).toContain('/customers/:id')
      expect(paths).toContain('/customers/:id/edit')
      expect(paths).toContain('/customers/:id/addresses/new')
      expect(paths).toContain('/customers/:id/addresses/:addressId/edit')
      expect(paths).toContain('/users')
      expect(paths).toContain('/users/new')
      expect(paths).toContain('/users/:id/edit')
      expect(paths).toContain('/403')
    })

    it('marks /login as guestOnly', () => {
      const route = router.getRoutes().find((r) => r.path === '/login')
      expect(route?.meta.guestOnly).toBe(true)
    })

    it('marks protected routes with requiresAuth', () => {
      const protectedPaths = ['/dashboard', '/customers', '/customers/new', '/users', '/403']
      for (const path of protectedPaths) {
        const route = router.getRoutes().find((r) => r.path === path)
        expect(route?.meta.requiresAuth, `${path} should require auth`).toBe(true)
      }
    })

    it('marks /users with requiredRole admin', () => {
      const route = router.getRoutes().find((r) => r.path === '/users')
      expect(route?.meta.requiredRole).toBe('admin')
    })

    it('marks /users/new and /users/:id/edit with requiredRole owner', () => {
      const newRoute = router.getRoutes().find((r) => r.path === '/users/new')
      const editRoute = router.getRoutes().find((r) => r.path === '/users/:id/edit')
      expect(newRoute?.meta.requiredRole).toBe('owner')
      expect(editRoute?.meta.requiredRole).toBe('owner')
    })
  })

  describe('beforeEach guard', () => {
    it('redirects unauthenticated user to /login on protected route', async () => {
      await router.push('/dashboard')
      await router.isReady()
      expect(router.currentRoute.value.path).toBe('/login')
    })

    it('redirects authenticated user from /login to /dashboard', async () => {
      const store = useAuthStore()
      authenticateAs(store, 'member')

      // Navigate to a known route first to ensure guard triggers on /login push
      await router.push('/customers')
      await router.push('/login')
      await router.isReady()
      expect(router.currentRoute.value.path).toBe('/dashboard')
    })

    it('allows authenticated user to access protected route', async () => {
      const store = useAuthStore()
      authenticateAs(store, 'member')

      await router.push('/customers')
      await router.isReady()
      expect(router.currentRoute.value.path).toBe('/customers')
    })

    it('redirects member to /403 when accessing admin route', async () => {
      const store = useAuthStore()
      authenticateAs(store, 'member')

      await router.push('/users')
      await router.isReady()
      expect(router.currentRoute.value.path).toBe('/403')
    })

    it('allows admin to access /users', async () => {
      const store = useAuthStore()
      authenticateAs(store, 'admin')

      await router.push('/users')
      await router.isReady()
      expect(router.currentRoute.value.path).toBe('/users')
    })

    it('redirects admin to /403 when accessing owner route', async () => {
      const store = useAuthStore()
      authenticateAs(store, 'admin')

      await router.push('/users/new')
      await router.isReady()
      expect(router.currentRoute.value.path).toBe('/403')
    })

    it('allows owner to access owner routes', async () => {
      const store = useAuthStore()
      authenticateAs(store, 'owner')

      await router.push('/users/new')
      await router.isReady()
      expect(router.currentRoute.value.path).toBe('/users/new')
    })

    it('redirects / to /dashboard for authenticated user', async () => {
      const store = useAuthStore()
      authenticateAs(store, 'member')

      await router.push('/')
      await router.isReady()
      expect(router.currentRoute.value.path).toBe('/dashboard')
    })
  })
})
