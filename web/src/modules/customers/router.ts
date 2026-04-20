import type { RouteRecordRaw } from 'vue-router'

export const customerRoutes: RouteRecordRaw[] = [
    {
        path: '/customers',
        component: () => import('@/modules/customers/views/CustomerListView.vue'),
        meta: { requiresAuth: true, keepAlive: true },
    },
    {
        path: '/customers/new',
        component: () => import('@/modules/customers/views/CustomerListView.vue'),
        meta: { requiresAuth: true, keepAlive: true },
    },
    {
        path: '/customers/:id',
        component: () => import('@/modules/customers/views/CustomerDetailView.vue'),
        meta: { requiresAuth: true },
    },
    {
        path: '/customers/:id/edit',
        component: () => import('@/modules/customers/views/CustomerFormView.vue'),
        meta: { requiresAuth: true },
    },
    {
        path: '/customers/:id/addresses/new',
        component: () => import('@/modules/customers/views/AddressFormView.vue'),
        meta: { requiresAuth: true },
    },
    {
        path: '/customers/:id/addresses/:addressId/edit',
        component: () => import('@/modules/customers/views/AddressFormView.vue'),
        meta: { requiresAuth: true },
    },
]
