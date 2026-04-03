<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import DataTable from '@/components/DataTable.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import FormField from '@/components/FormField.vue'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import type { Column } from '@/components/DataTable.vue'
import type { User, ApiError, CreateUserRequest } from '@/types'
import { useCrud } from '@/composables/useCrud'
import { useAuth } from '@/composables/useAuth'
import { useToast } from '@/composables/useToast'
import { useErrorHandler } from '@/composables/useErrorHandler'

const route = useRoute()
const router = useRouter()
const { hasRole } = useAuth()
const { list, create, remove, loading } = useCrud<User>('/v1/users')
const toast = useToast()
const { handleError } = useErrorHandler()

const users = ref<User[]>([])
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

function resetCreateForm() {
  name.value = ''
  email.value = ''
  password.value = ''
  role.value = 'member'
  formError.value = ''
  fieldErrors.value = {}
}

function closeCreateModal() {
  router.push('/users')
}

async function handleCreateUser() {
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
    await create(payload as unknown as Partial<User>)
    toast.success('Usuário criado com sucesso')
    resetCreateForm()
    await fetchUsers()
    router.push('/users')
  } catch (e: any) {
    const apiErr = e as ApiError
    if (apiErr.status === 400 && apiErr.validationErrors) {
      fieldErrors.value = { ...apiErr.validationErrors }
    } else if (apiErr.status === 409) {
      formError.value = apiErr.message || 'Conflito: email já cadastrado'
    } else {
      const handled = handleError(apiErr)
      if (!handled) {
        formError.value = apiErr?.message || 'Erro ao criar usuário'
      }
    }
  } finally {
    creating.value = false
  }
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

watch(
  () => route.path,
  (path) => {
    showCreateModal.value = path === '/users/new'
    if (!showCreateModal.value) {
      resetCreateForm()
    }
  },
  { immediate: true },
)
</script>

<template>
  <div class="user-list">
    <div class="user-list__header">
      <h1 class="user-list__title">Usuários</h1>
      <Button v-if="isOwner" class="user-list__new-btn" @click="navigateToNew">Novo Usuário</Button>
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

    <Dialog v-if="isOwner" :open="showCreateModal" @update:open="(open) => !open && closeCreateModal()">
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Novo Usuário</DialogTitle>
          <DialogDescription>Preencha os campos para criar um novo usuário.</DialogDescription>
        </DialogHeader>

        <div v-if="formError" class="user-list__error user-list__modal-error">
          <p class="user-list__error-message">{{ formError }}</p>
        </div>

        <FormField
          label="Nome"
          :modelValue="name"
          @update:modelValue="name = $event as string"
          :error="fieldErrors.name"
          :required="true"
          placeholder="Nome do usuário"
        />
        <FormField
          label="Email"
          type="email"
          :modelValue="email"
          @update:modelValue="email = $event as string"
          :error="fieldErrors.email"
          :required="true"
          placeholder="email@exemplo.com"
        />
        <FormField
          label="Senha"
          type="password"
          :modelValue="password"
          @update:modelValue="password = $event as string"
          :error="fieldErrors.password"
          :required="true"
          placeholder="Mínimo 8 caracteres"
        />
        <FormField
          label="Role"
          type="select"
          :modelValue="role"
          @update:modelValue="role = $event as 'admin' | 'member'"
          :error="fieldErrors.role"
          :required="true"
          :options="roleOptions"
        />

        <DialogFooter>
          <Button variant="outline" :disabled="creating" @click="closeCreateModal">Cancelar</Button>
          <Button :disabled="creating" @click="handleCreateUser">
            {{ creating ? 'Criando...' : 'Criar Usuário' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
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
  color: var(--foreground);
}

.user-list__new-btn {
  font-size: 0.875rem;
  font-weight: 500;
}

.user-list__error {
  text-align: center;
  padding: 2rem;
  background: var(--card);
  border: 1px solid var(--destructive);
  border-radius: 0.5rem;
}

.user-list__modal-error {
  padding: 0.75rem 1rem;
  margin-bottom: 0.75rem;
}

.user-list__error-message {
  color: var(--destructive);
  margin-bottom: 1rem;
}

.user-list__retry-btn {
  padding: 0.5rem 1rem;
  background: var(--primary);
  color: var(--primary-foreground);
  border: none;
  border-radius: 0.375rem;
  cursor: pointer;
  font-size: 0.875rem;
}

.user-list__retry-btn:hover {
  filter: brightness(1.08);
}
</style>
