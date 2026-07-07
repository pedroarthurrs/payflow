/**
 * Pinia store for RBAC state — permissions fetched from GET /v1/me.
 * Single source of truth for role + effective permissions.
 */
import http from '@/core/api/http'
import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { ROLE_DEFAULTS } from './defaults'
import type { Permission, Role } from './types'
import { ROLE_HIERARCHY } from './types'

interface MeResponse {
    organization_id: string
    user_id?: string
    role?: string
    permissions?: string[]
    issued_at: string | number
    expires_at: string | number
}

export const useRBACStore = defineStore('rbac', () => {
    const role = ref<Role | null>(null)
    const permissions = ref<Set<Permission>>(new Set())
    const loaded = ref(false)

    const roleLevel = computed(() => (role.value ? ROLE_HIERARCHY[role.value] ?? 0 : 0))

    /** Fetch effective permissions from /v1/me and hydrate store. */
    async function fetchPermissions(): Promise<void> {
        try {
            const { data } = await http.get<MeResponse>('/v1/me')
            const r = data.role as Role | undefined
            role.value = r ?? null

            if (data.permissions?.length) {
                permissions.value = new Set(data.permissions as Permission[])
            } else if (r && ROLE_DEFAULTS[r]) {
                permissions.value = new Set(ROLE_DEFAULTS[r])
            } else {
                permissions.value = new Set()
            }
            loaded.value = true
        } catch {
            // On failure keep whatever was previously loaded; auth interceptor
            // handles 401 → refresh → retry.
        }
    }

    /** Check single permission. */
    function can(perm: Permission): boolean {
        return permissions.value.has(perm)
    }

    /** Check ANY of given permissions. */
    function canAny(perms: Permission[]): boolean {
        return perms.some((p) => permissions.value.has(p))
    }

    /** Check ALL of given permissions. */
    function canAll(perms: Permission[]): boolean {
        return perms.every((p) => permissions.value.has(p))
    }

    /** Role hierarchy check: user role >= minRole. */
    function hasRole(minRole: Role): boolean {
        return roleLevel.value >= (ROLE_HIERARCHY[minRole] ?? 0)
    }

    /** Reset on logout. */
    function $reset(): void {
        role.value = null
        permissions.value = new Set()
        loaded.value = false
    }

    return {
        role,
        permissions,
        loaded,
        roleLevel,
        fetchPermissions,
        can,
        canAny,
        canAll,
        hasRole,
        $reset,
    }
})
