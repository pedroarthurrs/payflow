<script setup lang="ts">
import ConfirmDialog from '@/core/components/ConfirmDialog.vue'
import type { Column } from '@/core/components/DataTable.vue'
import DataTable from '@/core/components/DataTable.vue'
import { Button } from '@/core/components/ui/button'
import {
  Dialog,
  DialogContent,
} from '@/core/components/ui/dialog'
import { useAuth } from '@/core/composables/useAuth'
import { useErrorHandler } from '@/core/composables/useErrorHandler'
import { useToast } from '@/core/composables/useToast'
import type { ApiError, CreateUserRequest, User } from '@/core/types'
import { useCreateMember, useDeleteMember, useMemberList } from '@/modules/members/composables/useMembers'
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import MemberFormView from './MemberFormView.vue'

const route = useRoute()
const router = useRouter()
const { hasRole } = useAuth()
const toast = useToast()
const { handleError } = useErrorHandler()

const { data: queryData, isPending } = useMemberList()
const createMutation = useCreateMember()
const deleteMutation = useDeleteMember()

const members = computed(() => queryData.value?.data ?? [])
const fetchError = ref<string | null>(null)

const isOwner = hasRole('owner')

const showCreateModal = ref(false)
const creating = ref(false)
const formError = ref('')
const fieldErrors = ref<Record<string, string>>({})

const name = ref('')
const email = ref('')
const password = ref('')
const role = ref<'admin' | 'member'>('member')

const roleOptions = [
  { value: 'admin', label: 'Admin' },
  { value: 'member', label: 'Member' },
]

const columns: Column[] = [
  { key: 'name', label: 'Nome' },
  { key: 'email', label: 'Email' },
  { key: 'role', label: 'Role' },
  {
    key: 'joined_at',
    label: 'Data de Ingresso',
    format: (value: string) => {
      if (!value) return ''
      return new Date(value).toLocaleDateString('pt-BR')
    },
  },
]

// Confirm dialog state
const showConfirmDialog = ref(false)
const memberToDelete = ref<User | null>(null)

function navigateToNew() {
  router.push('/members/new')
}

function resetCreateForm() {
  name.value = ''
  email.value = ''
  password.value = ''
  role.value = 'member'
  formError.value = ''
  fieldErrors.value = {}
}

function closeCreateModal() {
  router.push('/members')
}

async function handleCreateMember() {
  fieldErrors.value = {}
  formError.value = ''
  creating.value = true
  try {
    const payload: CreateUserRequest = {
      name: name.value,
      email: email.value,
      password: password.value,
      role: role.value,
    }
    await createMutation.mutateAsync(payload)
    toast.success('Member criado com sucesso')
    resetCreateForm()
    router.push('/members')
  } catch (e: any) {
    const apiErr = e as ApiError
    if (apiErr.status === 400 && apiErr.validationErrors) {
      fieldErrors.value = { ...apiErr.validationErrors }
    } else if (apiErr.status === 409) {
      formError.value = apiErr.message || 'Conflito: email já cadastrado'
    } else {
      const handled = handleError(apiErr)
      if (!handled) {
        formError.value = apiErr?.message || 'Erro ao criar member'
      }
    }
  } finally {
    creating.value = false
  }
}

function handleEdit(member: User) {
  router.push(`/members/${member.id}/edit`)
}

function handleDelete(member: User) {
  memberToDelete.value = member
  showConfirmDialog.value = true
}

async function confirmDelete() {
  if (!memberToDelete.value) return
  try {
    await deleteMutation.mutateAsync(memberToDelete.value.id)
    toast.success('Member excluído com sucesso')
  } catch (e: any) {
    const apiErr = e as ApiError
    const handled = handleError(apiErr)
    if (!handled) {
      toast.error(apiErr?.message || 'Erro ao excluir member')
    }
  } finally {
    showConfirmDialog.value = false
    memberToDelete.value = null
  }
}

function cancelDelete() {
  showConfirmDialog.value = false
  memberToDelete.value = null
}

watch(
  () => route.path,
  (path) => {
    showCreateModal.value = path === '/members/new'
    if (!showCreateModal.value) {
      resetCreateForm()
    }
  },
  { immediate: true },
)
</script>

<template>
  <div class="p-6">
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-semibold text-foreground">Members</h1>
      <Button v-if="isOwner" @click="navigateToNew">Novo Member</Button>
    </div>

    <div v-if="fetchError" class="text-center p-8 bg-card border border-destructive rounded-lg">
      <p class="text-destructive mb-4">{{ fetchError }}</p>
    </div>

    <DataTable
      v-else
      :columns="columns"
      :data="members"
      :loading="isPending"
      :show-actions="isOwner"
      @edit="handleEdit"
      @delete="handleDelete"
    />

    <ConfirmDialog
      :visible="showConfirmDialog"
      title="Excluir Member"
      :message="`Tem certeza que deseja excluir o member '${memberToDelete?.name ?? ''}'?`"
      confirm-label="Excluir"
      variant="danger"
      @confirm="confirmDelete"
      @cancel="cancelDelete"
    />

    <Dialog v-if="isOwner" :open="showCreateModal" @update:open="(open) => !open && closeCreateModal()">
      <DialogContent class="max-w-[600px]">
        <MemberFormView />
      </DialogContent>
    </Dialog>
  </div>
</template>
