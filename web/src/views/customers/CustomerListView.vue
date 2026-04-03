<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import DataTable from '@/components/DataTable.vue'
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
import type { Customer, ApiError, CreateCustomerRequest } from '@/types'
import { useCrud } from '@/composables/useCrud'
import { useErrorHandler } from '@/composables/useErrorHandler'
import { useToast } from '@/composables/useToast'

const route = useRoute()
const router = useRouter()
const { list, create, loading } = useCrud<Customer>('/v1/customers')
const customers = ref<Customer[]>([])
const fetchError = ref<string | null>(null)
const { handleError } = useErrorHandler()
const toast = useToast()

const showCreateModal = ref(false)
const creating = ref(false)
const formError = ref('')
const fieldErrors = ref<Record<string, string>>({})

const name = ref('')
const email = ref('')
const phone = ref('')
const document = ref('')
const documentType = ref<'CPF' | 'CNPJ'>('CPF')
const metadata = ref('')

const documentTypeOptions = [
  { value: 'CPF', label: 'CPF' },
  { value: 'CNPJ', label: 'CNPJ' },
]

const columns: Column[] = [
  { key: 'name', label: 'Nome' },
  { key: 'email', label: 'Email' },
  { key: 'document', label: 'Documento' },
  { key: 'status', label: 'Status' },
  {
    key: 'created_at',
    label: 'Data de Criação',
    format: (value: string) => {
      if (!value) return ''
      return new Date(value).toLocaleDateString('pt-BR')
    },
  },
]

async function fetchCustomers() {
  fetchError.value = null
  try {
    customers.value = await list()
  } catch (e: any) {
    const apiErr = e as ApiError
    const handled = handleError(apiErr, fetchCustomers)
    if (!handled) {
      fetchError.value = apiErr?.message || 'Erro ao carregar clientes'
    }
  }
}

function handleRowClick(row: Customer) {
  router.push(`/customers/${row.id}`)
}

function navigateToNew() {
  router.push('/customers/new')
}

function resetCreateForm() {
  name.value = ''
  email.value = ''
  phone.value = ''
  document.value = ''
  documentType.value = 'CPF'
  metadata.value = ''
  formError.value = ''
  fieldErrors.value = {}
}

function closeCreateModal() {
  router.push('/customers')
}

function parseMetadata(): Record<string, unknown> | undefined {
  const trimmed = metadata.value.trim()
  if (!trimmed) return undefined
  try {
    return JSON.parse(trimmed)
  } catch {
    fieldErrors.value = { ...fieldErrors.value, metadata: 'JSON inválido' }
    return null as unknown as undefined
  }
}

async function handleCreateCustomer() {
  fieldErrors.value = {}
  formError.value = ''

  const parsedMetadata = parseMetadata()
  if (parsedMetadata === null) return

  creating.value = true
  try {
    const payload: CreateCustomerRequest = {
      name: name.value,
      email: email.value,
      document: document.value,
      document_type: documentType.value,
      phone: phone.value || undefined,
      metadata: parsedMetadata,
    }
    const created = await create(payload as unknown as Partial<Customer>)
    toast.success('Cliente criado com sucesso')
    resetCreateForm()
    router.push(`/customers/${created.id}`)
  } catch (e: any) {
    const apiErr = e as ApiError
    if (apiErr.status === 400 && apiErr.validationErrors) {
      fieldErrors.value = { ...apiErr.validationErrors }
    } else {
      const handled = handleError(apiErr)
      if (!handled) {
        formError.value = apiErr?.message || 'Erro ao criar cliente'
      }
    }
  } finally {
    creating.value = false
  }
}

onMounted(fetchCustomers)

watch(
  () => route.path,
  (path) => {
    showCreateModal.value = path === '/customers/new'
    if (!showCreateModal.value) {
      resetCreateForm()
    }
  },
  { immediate: true },
)
</script>

<template>
  <div class="customer-list">
    <div class="customer-list__header">
      <h1 class="customer-list__title">Clientes</h1>
      <Button class="customer-list__new-btn" @click="navigateToNew">Novo Cliente</Button>
    </div>

    <div v-if="fetchError" class="customer-list__error">
      <p class="customer-list__error-message">{{ fetchError }}</p>
      <button class="customer-list__retry-btn" @click="fetchCustomers">Tentar novamente</button>
    </div>

    <DataTable
      v-else
      :columns="columns"
      :data="customers"
      :loading="loading"
      @row-click="handleRowClick"
    />

    <Dialog :open="showCreateModal" @update:open="(open) => !open && closeCreateModal()">
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Novo Cliente</DialogTitle>
          <DialogDescription>Preencha os campos para criar um novo cliente.</DialogDescription>
        </DialogHeader>

        <div v-if="formError" class="customer-list__error customer-list__modal-error">
          <p class="customer-list__error-message">{{ formError }}</p>
        </div>

        <FormField
          label="Nome"
          :modelValue="name"
          @update:modelValue="name = $event as string"
          :error="fieldErrors.name"
          :required="true"
          placeholder="Nome do cliente"
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
          label="Telefone"
          type="tel"
          :modelValue="phone"
          @update:modelValue="phone = $event as string"
          :error="fieldErrors.phone"
          placeholder="(00) 00000-0000"
        />
        <FormField
          label="Documento"
          :modelValue="document"
          @update:modelValue="document = $event as string"
          :error="fieldErrors.document"
          :required="true"
          placeholder="Número do documento"
        />
        <FormField
          label="Tipo de Documento"
          type="select"
          :modelValue="documentType"
          @update:modelValue="documentType = $event as 'CPF' | 'CNPJ'"
          :error="fieldErrors.document_type"
          :required="true"
          :options="documentTypeOptions"
        />
        <FormField
          label="Metadata (JSON)"
          type="textarea"
          :modelValue="metadata"
          @update:modelValue="metadata = $event as string"
          :error="fieldErrors.metadata"
          placeholder='{"chave": "valor"}'
        />

        <DialogFooter>
          <Button variant="outline" :disabled="creating" @click="closeCreateModal">Cancelar</Button>
          <Button :disabled="creating" @click="handleCreateCustomer">
            {{ creating ? 'Criando...' : 'Criar Cliente' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>

<style scoped>
.customer-list {
  padding: 1.5rem;
}

.customer-list__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
}

.customer-list__title {
  font-size: 1.5rem;
  font-weight: 600;
  color: var(--foreground);
}

.customer-list__new-btn {
  font-size: 0.875rem;
  font-weight: 500;
}

.customer-list__error {
  text-align: center;
  padding: 2rem;
  background: var(--card);
  border: 1px solid var(--destructive);
  border-radius: 0.5rem;
}

.customer-list__modal-error {
  padding: 0.75rem 1rem;
  margin-bottom: 0.75rem;
}

.customer-list__error-message {
  color: var(--destructive);
  margin-bottom: 1rem;
}

.customer-list__retry-btn {
  padding: 0.5rem 1rem;
  background: var(--primary);
  color: var(--primary-foreground);
  border: none;
  border-radius: 0.375rem;
  cursor: pointer;
  font-size: 0.875rem;
}

.customer-list__retry-btn:hover {
  filter: brightness(1.08);
}
</style>
