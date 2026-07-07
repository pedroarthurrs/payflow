import type { RouteRecordRaw } from 'vue-router'

export const customerRoutes: RouteRecordRaw[] = [
    {
        path: '/customers',
        component: () => import('@/modules/customers/views/CustomerListView.vue'),
        meta: { requiresAuth: true, keepAlive: true, requiredPermission: 'customers:read' },
    },
    {
        path: '/customers/new',
        component: () => import('@/modules/customers/views/CustomerListView.vue'),
        meta: { requiresAuth: true, keepAlive: true, requiredPermission: 'customers:write' },
    },
    {
        path: '/customers/:id',
        component: () => import('@/modules/customers/views/CustomerDetailView.vue'),
        meta: { requiresAuth: true, requiredPermission: 'customers:read' },
    },
    {
        path: '/customers/:id/edit',
        component: () => import('@/modules/customers/views/CustomerFormView.vue'),
        meta: { requiresAuth: true, requiredPermission: 'customers:write' },
    },
    {
        path: '/customers/:id/addresses/new',
        component: () => import('@/modules/customers/views/AddressFormView.vue'),
        meta: { requiresAuth: true, requiredPermission: 'customers:write' },
    },
    {
        path: '/customers/:id/addresses/:addressId/edit',
        component: () => import('@/modules/customers/views/AddressFormView.vue'),
        meta: { requiresAuth: true, requiredPermission: 'customers:write' },
    },
]
