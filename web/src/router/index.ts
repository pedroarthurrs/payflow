import type { Permission, Role } from '@/core/rbac'
import { ROLE_HIERARCHY, useRBACStore } from '@/core/rbac'
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
    requiredPermission?: Permission
    requiredPermissions?: Permission[]
    guestOnly?: boolean
    keepAlive?: boolean
  }
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

  const rbac = useRBACStore()

  // Role hierarchy check
  if (to.meta.requiredRole) {
    const userRole = authStore.user?.role as Role | undefined
    if (!userRole || (ROLE_HIERARCHY[userRole] ?? 0) < (ROLE_HIERARCHY[to.meta.requiredRole] ?? 0)) {
      return '/403'
    }
  }

  // Single permission check
  if (to.meta.requiredPermission) {
    if (!rbac.loaded) await rbac.fetchPermissions()
    if (!rbac.can(to.meta.requiredPermission)) {
      return '/403'
    }
  }

  // Multiple permissions check (all required)
  if (to.meta.requiredPermissions?.length) {
    if (!rbac.loaded) await rbac.fetchPermissions()
    if (!rbac.canAll(to.meta.requiredPermissions)) {
      return '/403'
    }
  }
})

export default router
