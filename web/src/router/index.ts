import { createRouter, createWebHistory } from 'vue-router'
import type { Role } from '@/types'
import { useAuthStore } from '@/stores/auth'

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
      path: '/users',
      component: () => import('@/views/users/UserListView.vue'),
      meta: { requiresAuth: true, requiredRole: 'admin', keepAlive: true },
    },
    {
      path: '/users/new',
      component: () => import('@/views/users/UserListView.vue'),
      meta: { requiresAuth: true, requiredRole: 'owner', keepAlive: true },
    },
    {
      path: '/users/:id/edit',
      component: () => import('@/views/users/UserFormView.vue'),
      meta: { requiresAuth: true, requiredRole: 'owner' },
    },
    {
      path: '/403',
      component: () => import('@/components/AccessDenied.vue'),
      meta: { requiresAuth: true },
    },
  ],
})

router.beforeEach((to) => {
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
