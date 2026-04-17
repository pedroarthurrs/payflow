import { useAuthStore } from '@/stores/auth'
import type { Role } from '@/types'
import { computed } from 'vue'
import { useRouter } from 'vue-router'

const ROLE_HIERARCHY: Record<Role, number> = {
  member: 0,
  admin: 1,
  owner: 2,
}

export function useAuth() {
  const store = useAuthStore()
  const router = useRouter()

  const isAuthenticated = computed(() => store.isAuthenticated)
  const isMultiOrgPending = computed(() => store.isMultiOrgPending)
  const pendingOrganizations = computed(() => store.pendingOrganizations)

  async function login(email: string, password: string): Promise<void> {
    await store.login(email, password)

    if (store.isMultiOrgPending) {
      router.push('/select-organization')
      return
    }

    router.push('/dashboard')
  }

  async function selectOrganization(organizationId: string): Promise<void> {
    await store.selectOrganization(organizationId)
    router.push('/dashboard')
  }

  function logout(): void {
    store.logout()
  }

  function hasRole(minRole: Role): boolean {
    const userRole = store.user?.role
    if (!userRole) return false
    return ROLE_HIERARCHY[userRole] >= ROLE_HIERARCHY[minRole]
  }

  return {
    login,
    selectOrganization,
    logout,
    hasRole,
    isAuthenticated,
    isMultiOrgPending,
    pendingOrganizations,
  }
}
