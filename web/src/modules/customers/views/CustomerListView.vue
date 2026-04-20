<script setup lang="ts">
import type { Column } from '@/core/components/DataTable.vue'
import DataTable from '@/core/components/DataTable.vue'
import FormField from '@/core/components/FormField.vue'
import { Button } from '@/core/components/ui/button'
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from '@/core/components/ui/dialog'
import { useErrorHandler } from '@/core/composables/useErrorHandler'
import { useToast } from '@/core/composables/useToast'
import type { ApiError, CreateCustomerRequest, Customer } from '@/core/types'
import { formatDocument, formatPhone } from '@/core/utils/formatting'
import { useCreateCustomer, useCustomerList } from '@/modules/customers/composables/useCustomers'
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const { data: queryData, isPending } = useCustomerList()
const createMutation = useCreateCustomer()
const customers = computed(() => queryData.value?.data ?? [])
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
  { key: 'phone', label: 'Telefone', format: (value: string) => formatPhone(value) || '—' },
  { key: 'document', label: 'Documento', format: (value: string, row: Customer) => formatDocument(value, row.document_type) },
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

function handleRowClick(row: Customer) {
  router.push(`/customers/${row.id}`)
}

function openCreateModal() {
  showCreateModal.value = true
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
  showCreateModal.value = false
  resetCreateForm()
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
    const created = await createMutation.mutateAsync(payload)
    toast.success('Cliente criado com sucesso')
    closeCreateModal()
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
</script>

<template>
  <div class="p-6">
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-semibold text-foreground">Clientes</h1>
      <Button @click="openCreateModal">Novo Cliente</Button>
    </div>

    <div v-if="fetchError" class="text-center p-8 bg-card border border-destructive rounded-lg">
      <p class="text-destructive mb-4">{{ fetchError }}</p>
    </div>

    <DataTable
      v-else
      :columns="columns"
      :data="customers"
      :loading="isPending"
      @row-click="handleRowClick"
    />

    <Dialog v-model:open="showCreateModal" @update:open="(open) => !open && closeCreateModal()">
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Novo Cliente</DialogTitle>
          <DialogDescription>Preencha os campos para criar um novo cliente.</DialogDescription>
        </DialogHeader>

        <div v-if="formError" class="text-center py-3 px-4 bg-card border border-destructive rounded-lg mb-3">
          <p class="text-destructive">{{ formError }}</p>
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
