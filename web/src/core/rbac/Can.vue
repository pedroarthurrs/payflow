<script setup lang="ts">
/**
 * <Can> — renderless guard component.
 *
 * Usage:
 *   <Can permission="customers:delete">
 *     <Button>Delete</Button>
 *   </Can>
 *
 *   <Can :any="['wallets:payout', 'wallets:adjust']">
 *     <WalletActions />
 *   </Can>
 *
 *   <Can role="admin">
 *     <AdminPanel />
 *   </Can>
 *
 *   <Can permission="customers:delete" #default="{ allowed }">
 *     <Button :disabled="!allowed">Delete</Button>
 *   </Can>
 */
import { computed } from 'vue'
import { useRBACStore } from './store'
import type { Permission, Role } from './types'

const props = defineProps<{
  permission?: Permission
  any?: Permission[]
  all?: Permission[]
  role?: Role
}>()

const store = useRBACStore()

const allowed = computed(() => {
  if (props.permission) return store.can(props.permission)
  if (props.any?.length) return store.canAny(props.any)
  if (props.all?.length) return store.canAll(props.all)
  if (props.role) return store.hasRole(props.role)
  return true
})
</script>

<template>
  <slot v-if="allowed" :allowed="allowed" />
</template>
