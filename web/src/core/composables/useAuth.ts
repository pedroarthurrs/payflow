import type { Role } from '@/core/types'
import { useAuthStore } from '@/modules/auth/store/auth'
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
  const user = computed(() => store.user)
  const userRole = computed(() => store.userRole)
  const organizationId = computed(() => store.organizationId)
  const lastError = computed(() => store.lastError)

  async function login(email: string, password: string): Promise<void> {
    try {
      await store.login(email, password)

      if (store.isMultiOrgPending) {
        await router.push('/select-organization')
        return
      }

      await router.push('/dashboard')
    } catch (err) {
      console.error('Login failed:', err)
      throw err
    }
  }

  async function selectOrganization(orgId: string): Promise<void> {
    try {
      await store.selectOrganization(orgId)
      await router.push('/dashboard')
    } catch (err) {
      console.error('Organization selection failed:', err)
      throw err
    }
  }

  function logout(): void {
    store.logout()
    router.push('/login')
  }

  async function refreshToken(): Promise<void> {
    try {
      await store.refreshAccessToken()
    } catch (err) {
      console.error('Token refresh failed:', err)
      throw err
    }
  }

  function hasRole(minRole: Role): boolean {
    const role = store.user?.role
    if (!role) return false
    return ROLE_HIERARCHY[role] >= ROLE_HIERARCHY[minRole]
  }

  function hasAnyRole(roles: Role[]): boolean {
    const role = store.user?.role
    if (!role) return false
    return roles.includes(role)
  }

  return {
    login,
    selectOrganization,
    logout,
    refreshToken,
    hasRole,
    hasAnyRole,
    isAuthenticated,
    isMultiOrgPending,
    pendingOrganizations,
    user,
    userRole,
    organizationId,
    lastError,
  }
}
