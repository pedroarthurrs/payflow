import { useAuthStore } from '@/modules/auth/store/auth'
import { useStatusStore } from '@/modules/auth/store/status'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import router from './index'

// Some routes are gated by RBAC permissions (customers:*) rather than by role.
// The guard consults the RBAC store for those; role-gated routes (/members)
// use the real ROLE_HIERARCHY, so we preserve it from the actual module and
// only stub the store instance. `rbacState.can` toggles permission outcomes.
const rbacState = { can: true }
vi.mock('@/core/rbac', async (importActual) => {
  const actual = await importActual<typeof import('@/core/rbac')>()
  return {
    ...actual,
    useRBACStore: () => ({
      loaded: true,
      can: () => rbacState.can,
      canAll: () => rbacState.can,
      canAny: () => rbacState.can,
      fetchPermissions: vi.fn().mockResolvedValue(undefined),
    }),
  }
})

type TestRole = 'support' | 'admin' | 'owner'

function authenticateAs(store: ReturnType<typeof useAuthStore>, role: TestRole) {
  store.$patch({
    // The router guard evaluates roles through the RBAC ROLE_HIERARCHY
    // (owner/admin/developer/support), casting from the auth store's own,
    // narrower Role union — so we exercise the RBAC roles the guard actually
    // ranks. The cast mirrors that same widening the guard performs.
    user: { organizationId: 'org-1', memberId: 'member-1', role, authType: 'user' } as never,
  })
}

describe('Router', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    rbacState.can = true
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
      expect(paths).toContain('/members')
      expect(paths).toContain('/members/new')
      expect(paths).toContain('/members/:id/edit')
      expect(paths).toContain('/403')
    })

    it('marks /login as guestOnly', () => {
      const route = router.getRoutes().find((r) => r.path === '/login')
      expect(route?.meta.guestOnly).toBe(true)
    })

    it('marks protected routes with requiresAuth', () => {
      const protectedPaths = ['/dashboard', '/customers', '/customers/new', '/members', '/403']
      for (const path of protectedPaths) {
        const route = router.getRoutes().find((r) => r.path === path)
        expect(route?.meta.requiresAuth, `${path} should require auth`).toBe(true)
      }
    })

    it('marks /members with requiredRole admin', () => {
      const route = router.getRoutes().find((r) => r.path === '/members')
      expect(route?.meta.requiredRole).toBe('admin')
    })

    it('marks /members/new and /members/:id/edit with requiredRole owner', () => {
      const newRoute = router.getRoutes().find((r) => r.path === '/members/new')
      const editRoute = router.getRoutes().find((r) => r.path === '/members/:id/edit')
      expect(newRoute?.meta.requiredRole).toBe('owner')
      expect(editRoute?.meta.requiredRole).toBe('owner')
    })

    it('gates customer routes with an RBAC permission', () => {
      const list = router.getRoutes().find((r) => r.path === '/customers')
      const create = router.getRoutes().find((r) => r.path === '/customers/new')
      expect(list?.meta.requiredPermission).toBe('customers:read')
      expect(create?.meta.requiredPermission).toBe('customers:write')
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
      authenticateAs(store, 'admin')

      // Navigate to a known route first to ensure guard triggers on /login push
      await router.push('/customers')
      await router.push('/login')
      await router.isReady()
      expect(router.currentRoute.value.path).toBe('/dashboard')
    })

    it('allows a user with the permission to access a permission-gated route', async () => {
      const store = useAuthStore()
      authenticateAs(store, 'support')
      rbacState.can = true

      await router.push('/customers')
      await router.isReady()
      expect(router.currentRoute.value.path).toBe('/customers')
    })

    it('redirects to /403 when the user lacks the required permission', async () => {
      const store = useAuthStore()
      authenticateAs(store, 'support')
      rbacState.can = false

      // Navigate away first so the push to /customers is not a duplicate
      // navigation (which would skip the guard and leave us on /customers).
      await router.push('/dashboard')
      await router.push('/customers')
      await router.isReady()
      expect(router.currentRoute.value.path).toBe('/403')
    })

    it('redirects a below-rank user to /403 on an admin route', async () => {
      const store = useAuthStore()
      authenticateAs(store, 'support')

      await router.push('/members')
      await router.isReady()
      expect(router.currentRoute.value.path).toBe('/403')
    })

    it('allows admin to access /members', async () => {
      const store = useAuthStore()
      authenticateAs(store, 'admin')

      await router.push('/members')
      await router.isReady()
      expect(router.currentRoute.value.path).toBe('/members')
    })

    it('redirects admin to /403 when accessing an owner route', async () => {
      const store = useAuthStore()
      authenticateAs(store, 'admin')

      await router.push('/members/new')
      await router.isReady()
      expect(router.currentRoute.value.path).toBe('/403')
    })

    it('allows owner to access owner routes', async () => {
      const store = useAuthStore()
      authenticateAs(store, 'owner')

      await router.push('/members/new')
      await router.isReady()
      expect(router.currentRoute.value.path).toBe('/members/new')
    })

    it('redirects / to /dashboard for authenticated user', async () => {
      const store = useAuthStore()
      authenticateAs(store, 'admin')

      await router.push('/')
      await router.isReady()
      expect(router.currentRoute.value.path).toBe('/dashboard')
    })
  })
})
