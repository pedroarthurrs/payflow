<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import DataTable from '@/components/DataTable.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import type { Column } from '@/components/DataTable.vue'
import type { User, ApiError } from '@/types'
import { useCrud } from '@/composables/useCrud'
import { useAuth } from '@/composables/useAuth'
import { useToast } from '@/composables/useToast'
import { useErrorHandler } from '@/composables/useErrorHandler'

const router = useRouter()
const { hasRole } = useAuth()
const { list, remove, loading } = useCrud<User>('/v1/users')
const toast = useToast()
const { handleError } = useErrorHandler()

const users = ref<User[]>([])
const fetchError = ref<string | null>(null)

const isOwner = hasRole('owner')

const columns: Column[] = [
  { key: 'name', label: 'Nome' },
  { key: 'email', label: 'Email' },
  { key: 'role', label: 'Role' },
  {
    key: 'created_at',
    label: 'Data de Criação',
    format: (value: string) => {
      if (!value) return ''
      return new Date(value).toLocaleDateString('pt-BR')
    },
  },
]

// Confirm dialog state
const showConfirmDialog = ref(false)
const userToDelete = ref<User | null>(null)

async function fetchUsers() {
  fetchError.value = null
  try {
    users.value = await list()
  } catch (e: any) {
    const apiErr = e as ApiError
    const handled = handleError(apiErr, fetchUsers)
    if (!handled) {
      fetchError.value = apiErr?.message || 'Erro ao carregar usuários'
    }
  }
}

function navigateToNew() {
  router.push('/users/new')
}

function handleEdit(user: User) {
  router.push(`/users/${user.id}/edit`)
}

function handleDelete(user: User) {
  userToDelete.value = user
  showConfirmDialog.value = true
}

async function confirmDelete() {
  if (!userToDelete.value) return
  try {
    await remove(userToDelete.value.id)
    toast.success('Usuário excluído com sucesso')
    await fetchUsers()
  } catch (e: any) {
    const apiErr = e as ApiError
    const handled = handleError(apiErr)
    if (!handled) {
      toast.error(apiErr?.message || 'Erro ao excluir usuário')
    }
  } finally {
    showConfirmDialog.value = false
    userToDelete.value = null
  }
}

function cancelDelete() {
  showConfirmDialog.value = false
  userToDelete.value = null
}

onMounted(fetchUsers)
</script>

<template>
  <div class="user-list">
    <div class="user-list__header">
      <h1 class="user-list__title">Usuários</h1>
      <button v-if="isOwner" class="user-list__new-btn" @click="navigateToNew">
        Novo Usuário
      </button>
    </div>

    <div v-if="fetchError" class="user-list__error">
      <p class="user-list__error-message">{{ fetchError }}</p>
      <button class="user-list__retry-btn" @click="fetchUsers">Tentar novamente</button>
    </div>

    <DataTable
      v-else
      :columns="columns"
      :data="users"
      :loading="loading"
      :show-actions="isOwner"
      @edit="handleEdit"
      @delete="handleDelete"
    />

    <ConfirmDialog
      :visible="showConfirmDialog"
      title="Excluir Usuário"
      :message="`Tem certeza que deseja excluir o usuário '${userToDelete?.name ?? ''}'?`"
      confirm-label="Excluir"
      variant="danger"
      @confirm="confirmDelete"
      @cancel="cancelDelete"
    />
  </div>
</template>

<style scoped>
.user-list {
  padding: 1.5rem;
}

.user-list__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
}

.user-list__title {
  font-size: 1.5rem;
  font-weight: 600;
  color: #1f2937;
}

.user-list__new-btn {
  padding: 0.5rem 1rem;
  background: #2563eb;
  color: #fff;
  border: none;
  border-radius: 0.375rem;
  cursor: pointer;
  font-size: 0.875rem;
  font-weight: 500;
}

.user-list__new-btn:hover {
  background: #1d4ed8;
}

.user-list__error {
  text-align: center;
  padding: 2rem;
  background: #fef2f2;
  border: 1px solid #fecaca;
  border-radius: 0.5rem;
}

.user-list__error-message {
  color: #dc2626;
  margin-bottom: 1rem;
}

.user-list__retry-btn {
  padding: 0.5rem 1rem;
  background: #2563eb;
  color: #fff;
  border: none;
  border-radius: 0.375rem;
  cursor: pointer;
  font-size: 0.875rem;
}

.user-list__retry-btn:hover {
  background: #1d4ed8;
}
</style>
