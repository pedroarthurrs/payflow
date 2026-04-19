<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import FormField from '@/components/FormField.vue'
import LoadingSpinner from '@/components/LoadingSpinner.vue'
import { Button } from '@/components/ui/button'
import { useCrud } from '@/composables/useCrud'
import { useToast } from '@/composables/useToast'
import { useErrorHandler } from '@/composables/useErrorHandler'
import type { Customer, CreateCustomerRequest, UpdateCustomerRequest, ApiError } from '@/types'

const route = useRoute()
const router = useRouter()
const toast = useToast()
const { getById, create, update, loading } = useCrud<Customer>('/v1/customers')

const customerId = computed(() => route.params.id as string | undefined)
const isEditMode = computed(() => !!customerId.value)

const name = ref('')
const email = ref('')
const phone = ref('')
const document = ref('')
const documentType = ref<'CPF' | 'CNPJ'>('CPF')
const metadata = ref('')

const fieldErrors = ref<Record<string, string>>({})
const formError = ref('')
const fetchError = ref('')
const notFound = ref(false)

const { handleError } = useErrorHandler({
  onNotFound: () => {
    notFound.value = true
  },
})

const documentTypeOptions = [
  { value: 'CPF', label: 'CPF' },
  { value: 'CNPJ', label: 'CNPJ' },
]

const pageTitle = computed(() => (isEditMode.value ? 'Editar Cliente' : 'Novo Cliente'))

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

async function loadCustomer() {
  if (!customerId.value) return
  fetchError.value = ''
  notFound.value = false
  try {
    const customer = await getById(customerId.value)
    name.value = customer.name
    email.value = customer.email
    phone.value = customer.phone ?? ''
    document.value = customer.document
    documentType.value = customer.document_type
    metadata.value = customer.metadata ? JSON.stringify(customer.metadata, null, 2) : ''
  } catch (e) {
    const err = e as ApiError
    const handled = handleError(err, loadCustomer)
    if (!handled) {
      fetchError.value = err.message || 'Erro ao carregar cliente'
    }
  }
}

async function handleSubmit() {
  fieldErrors.value = {}
  formError.value = ''

  const parsedMetadata = parseMetadata()
  if (parsedMetadata === null) return

  try {
    if (isEditMode.value) {
      const payload: UpdateCustomerRequest = {
        name: name.value,
        email: email.value,
        document: document.value,
        document_type: documentType.value,
        phone: phone.value || undefined,
        metadata: parsedMetadata,
      }
      await update(customerId.value!, payload as unknown as Partial<Customer>)
      toast.success('Cliente atualizado com sucesso')
      router.push(`/customers/${customerId.value}`)
    } else {
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
      router.push(`/customers/${created.id}`)
    }
  } catch (e) {
    const err = e as ApiError
    if (err.status === 400 && err.validationErrors) {
      fieldErrors.value = { ...err.validationErrors }
    } else if (err.status === 409) {
      formError.value = err.message || 'Conflito: registro já existe'
    } else {
      const handled = handleError(err)
      if (!handled) {
        formError.value = err.message || 'Erro ao salvar cliente'
      }
    }
  }
}

onMounted(() => {
  if (isEditMode.value) {
    loadCustomer()
  }
})
</script>

<template>
  <div class="mx-auto max-w-[600px] p-6">
    <h1 class="text-2xl font-bold text-foreground mb-6">{{ pageTitle }}</h1>

    <div v-if="notFound" class="text-center p-8 bg-card border border-destructive rounded-lg">
      <p class="text-destructive mb-4">Cliente não encontrado</p>
      <router-link to="/customers" class="text-primary text-sm font-medium hover:underline">Voltar para listagem</router-link>
    </div>

    <div v-else-if="fetchError" class="text-center p-8">
      <p class="text-destructive mb-4">{{ fetchError }}</p>
      <Button @click="loadCustomer">Tentar novamente</Button>
    </div>

    <LoadingSpinner v-else-if="loading && isEditMode && !name" size="lg" />

    <form v-else class="flex flex-col" @submit.prevent="handleSubmit">
      <div v-if="formError" class="bg-card border border-destructive text-destructive px-4 py-3 rounded-md mb-4 text-sm">
        {{ formError }}
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

      <div class="flex justify-end gap-3 mt-2">
        <Button type="button" variant="outline" @click="router.push('/customers')">Cancelar</Button>
        <Button type="submit" :disabled="loading">
          {{ loading ? 'Salvando...' : (isEditMode ? 'Salvar' : 'Criar') }}
        </Button>
      </div>
    </form>
  </div>
</template>
