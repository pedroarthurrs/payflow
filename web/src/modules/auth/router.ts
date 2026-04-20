import type { RouteRecordRaw } from 'vue-router'

export const authRoutes: RouteRecordRaw[] = [
    {
        path: '/login',
        component: () => import('@/modules/auth/views/LoginView.vue'),
        meta: { guestOnly: true },
    },
    {
        path: '/select-organization',
        component: () => import('@/modules/auth/views/SelectOrganizationView.vue'),
        meta: { guestOnly: true },
    },
]
