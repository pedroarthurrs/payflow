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
const toast = useToast()
const { handleError } = useErrorHandler()
const { list, create, remove, loading } = useCrud<User>('/v1/members')

const members = ref<User[]>([])
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

async function fetchMembers() {
  fetchError.value = null
  try {
    members.value = await list()
  } catch (e: any) {
    const apiErr = e as ApiError
    const handled = handleError(apiErr, fetchMembers)
    if (!handled) {
      fetchError.value = apiErr?.message || 'Erro ao carregar members'
    }
  }
}

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
    await create(payload)
    toast.success('Member criado com sucesso')
    resetCreateForm()
    await fetchMembers()
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
    await remove(memberToDelete.value.id)
    toast.success('Member excluído com sucesso')
    await fetchMembers()
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

onMounted(fetchMembers)

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
  <div class="member-list">
    <div class="member-list__header">
      <h1 class="member-list__title">Members</h1>
      <Button v-if="isOwner" class="member-list__new-btn" @click="navigateToNew">Novo Member</Button>
    </div>

    <div v-if="fetchError" class="member-list__error">
      <p class="member-list__error-message">{{ fetchError }}</p>
      <button class="member-list__retry-btn" @click="fetchMembers">Tentar novamente</button>
    </div>

    <DataTable
      v-else
      :columns="columns"
      :data="members"
      :loading="loading"
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
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Novo Member</DialogTitle>
          <DialogDescription>Preencha os campos para criar um novo member.</DialogDescription>
        </DialogHeader>

        <div v-if="formError" class="member-list__error member-list__modal-error">
          <p class="member-list__error-message">{{ formError }}</p>
        </div>

        <FormField
          label="Nome"
          :modelValue="name"
          @update:modelValue="name = $event as string"
          :error="fieldErrors.name"
          :required="true"
          placeholder="Nome do member"
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
          <Button :disabled="creating" @click="handleCreateMember">
            {{ creating ? 'Criando...' : 'Criar Member' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>

<style scoped>
.member-list {
  padding: 1.5rem;
}

.member-list__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
}

.member-list__title {
  font-size: 1.5rem;
  font-weight: 600;
  color: var(--foreground);
}

.member-list__new-btn {
  font-size: 0.875rem;
  font-weight: 500;
}

.member-list__error {
  text-align: center;
  padding: 2rem;
  background: var(--card);
  border: 1px solid var(--destructive);
  border-radius: 0.5rem;
}

.member-list__error-message {
  color: var(--destructive);
  margin: 0;
}

.member-list__modal-error {
  border-color: var(--destructive);
  background-color: hsl(var(--destructive) / 0.1);
}

.member-list__retry-btn {
  padding: 0.5rem 1rem;
  margin-top: 1rem;
  background: var(--primary);
  color: var(--primary-foreground);
  border: none;
  border-radius: 0.25rem;
  cursor: pointer;
  font-size: 0.875rem;
}

.member-list__retry-btn:hover {
  opacity: 0.9;
}
</style>
