/**
 * v-can directive — hide elements based on permission.
 *
 * Usage:
 *   <button v-can="'customers:delete'">Delete</button>
 *   <div v-can:role="'admin'">Admin only</div>
 *
 * Register globally:
 *   app.directive('can', vCan)
 */
import type { Directive } from 'vue'
import { useRBACStore } from './store'
import type { Permission, Role } from './types'

export const vCan: Directive<HTMLElement, Permission | Permission[] | Role> = {
    mounted(el, binding) {
        check(el, binding.arg, binding.value)
    },
    updated(el, binding) {
        check(el, binding.arg, binding.value)
    },
}

function check(el: HTMLElement, arg: string | undefined, value: Permission | Permission[] | Role) {
    const store = useRBACStore()
    let allowed = false

    if (arg === 'role') {
        allowed = store.hasRole(value as Role)
    } else if (arg === 'any' && Array.isArray(value)) {
        allowed = store.canAny(value as Permission[])
    } else if (arg === 'all' && Array.isArray(value)) {
        allowed = store.canAll(value as Permission[])
    } else {
        allowed = store.can(value as Permission)
    }

    el.style.display = allowed ? '' : 'none'
}
