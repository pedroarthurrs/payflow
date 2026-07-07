/**
 * useRBAC — composable for permission/role checks in components.
 *
 * Usage:
 *   const { can, canAny, hasRole } = useRBAC()
 *   if (can('customers:delete')) { ... }
 */
import { computed } from 'vue'
import { useRBACStore } from './store'
import type { Permission, Role } from './types'

export function useRBAC() {
    const store = useRBACStore()

    return {
        /** Reactive current role. */
        role: computed(() => store.role),

        /** Whether permissions have been loaded from /me. */
        loaded: computed(() => store.loaded),

        /** Check single permission (reactive). */
        can: (perm: Permission) => computed(() => store.can(perm)),

        /** Check any of permissions (reactive). */
        canAny: (perms: Permission[]) => computed(() => store.canAny(perms)),

        /** Check all permissions (reactive). */
        canAll: (perms: Permission[]) => computed(() => store.canAll(perms)),

        /** Role hierarchy check (reactive). */
        hasRole: (minRole: Role) => computed(() => store.hasRole(minRole)),

        /** Imperative single check (non-reactive). */
        checkPermission: (perm: Permission) => store.can(perm),

        /** Imperative role check (non-reactive). */
        checkRole: (minRole: Role) => store.hasRole(minRole),

        /** Re-fetch permissions from /me. */
        refresh: () => store.fetchPermissions(),
    }
}
