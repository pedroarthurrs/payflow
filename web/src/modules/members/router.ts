import type { RouteRecordRaw } from 'vue-router'

export const memberRoutes: RouteRecordRaw[] = [
    {
        path: '/members',
        component: () => import('@/modules/members/views/MemberListView.vue'),
        meta: { requiresAuth: true, requiredRole: 'admin', keepAlive: true },
    },
    {
        path: '/members/new',
        component: () => import('@/modules/members/views/MemberListView.vue'),
        meta: { requiresAuth: true, requiredRole: 'owner', keepAlive: true },
    },
    {
        path: '/members/:id/edit',
        component: () => import('@/modules/members/views/MemberFormView.vue'),
        meta: { requiresAuth: true, requiredRole: 'owner' },
    },
]
