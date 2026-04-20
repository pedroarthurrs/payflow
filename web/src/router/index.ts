import type { Role } from '@/core/types'
import { authRoutes } from '@/modules/auth/router'
import { useAuthStore } from '@/modules/auth/store/auth'
import { useStatusStore } from '@/modules/auth/store/status'
import { customerRoutes } from '@/modules/customers/router'
import { memberRoutes } from '@/modules/members/router'
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
    ...authRoutes,
    {
      path: '/dashboard',
      component: () => import('@/views/dashboard/DashboardView.vue'),
      meta: { requiresAuth: true, keepAlive: true },
    },
    ...customerRoutes,
    ...memberRoutes,
    {
      path: '/403',
      component: () => import('@/core/components/AccessDenied.vue'),
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
