import { useAuthStore } from '@/stores/auth'
import { useStatusStore } from '@/stores/status'
import type { Role } from '@/types'
import { createRouter, createWebHistory } from 'vue-router'

declare module 'vue-router' {
  interface RouteMeta {
    requiresAuth?: boolean
    requiredRole?: Role
    guestOnly?: boolean
    keepAlive?: boolean
  }
}

const ROLE_HIERARCHY: Record<Role, number> = {
  member: 0,
  admin: 1,
  owner: 2,
}

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      redirect: '/dashboard',
    },
    {
      path: '/login',
      component: () => import('@/views/LoginView.vue'),
      meta: { guestOnly: true },
    },
    {
      path: '/select-organization',
      component: () => import('@/views/SelectOrganizationView.vue'),
      meta: { guestOnly: true },
    },
    {
      path: '/dashboard',
      component: () => import('@/views/DashboardView.vue'),
      meta: { requiresAuth: true, keepAlive: true },
    },
    {
      path: '/customers',
      component: () => import('@/views/customers/CustomerListView.vue'),
      meta: { requiresAuth: true, keepAlive: true },
    },
    {
      path: '/customers/new',
      component: () => import('@/views/customers/CustomerListView.vue'),
      meta: { requiresAuth: true, keepAlive: true },
    },
    {
      path: '/customers/:id',
      component: () => import('@/views/customers/CustomerDetailView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/customers/:id/edit',
      component: () => import('@/views/customers/CustomerFormView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/customers/:id/addresses/new',
      component: () => import('@/views/addresses/AddressFormView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/customers/:id/addresses/:addressId/edit',
      component: () => import('@/views/addresses/AddressFormView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/members',
      component: () => import('@/views/members/MemberListView.vue'),
      meta: { requiresAuth: true, requiredRole: 'admin', keepAlive: true },
    },
    {
      path: '/members/new',
      component: () => import('@/views/members/MemberListView.vue'),
      meta: { requiresAuth: true, requiredRole: 'owner', keepAlive: true },
    },
    {
      path: '/members/:id/edit',
      component: () => import('@/views/members/MemberFormView.vue'),
      meta: { requiresAuth: true, requiredRole: 'owner' },
    },
    {
      path: '/403',
      component: () => import('@/components/AccessDenied.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/maintenance',
      component: () => import('@/views/MaintenanceView.vue'),
    },
    {
      path: '/:pathMatch(.*)*',
      component: () => import('@/views/NotFoundView.vue'),
    },
  ],
})

router.beforeEach(async (to) => {
  // For guest-only routes (login, etc.) verify maintenance status before proceeding.
  // For authenticated routes, a 503 response from any API call will redirect to /maintenance via the HTTP interceptor.
  if (to.meta.guestOnly) {
    const statusStore = useStatusStore()
    await statusStore.check()
    if (statusStore.maintenance) {
      return '/maintenance'
    }
  }

  const authStore = useAuthStore()

  if (to.meta.requiresAuth && !authStore.isAuthenticated) {
    return '/login'
  }

  if (to.meta.guestOnly && authStore.isAuthenticated) {
    return '/dashboard'
  }

  if (to.meta.requiredRole) {
    const userRole = authStore.user?.role
    if (!userRole || ROLE_HIERARCHY[userRole] < ROLE_HIERARCHY[to.meta.requiredRole]) {
      return '/403'
    }
  }
})

export default router
