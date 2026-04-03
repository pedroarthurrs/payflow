<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import FormField from '@/components/FormField.vue'
import LoadingSpinner from '@/components/LoadingSpinner.vue'
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
  <div class="customer-form">
    <h1 class="customer-form__title">{{ pageTitle }}</h1>

    <div v-if="notFound" class="customer-form__not-found">
      <p class="customer-form__not-found-message">Cliente não encontrado</p>
      <router-link to="/customers" class="customer-form__back-link">Voltar para listagem</router-link>
    </div>

    <div v-else-if="fetchError" class="customer-form__fetch-error">
      <p class="customer-form__fetch-error-message">{{ fetchError }}</p>
      <button class="customer-form__retry-btn" @click="loadCustomer">Tentar novamente</button>
    </div>

    <LoadingSpinner v-else-if="loading && isEditMode && !name" size="lg" />

    <form v-else class="customer-form__form" @submit.prevent="handleSubmit">
      <div v-if="formError" class="customer-form__error">
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

      <div class="customer-form__actions">
        <button
          type="button"
          class="customer-form__cancel-btn"
          @click="router.push('/customers')"
        >
          Cancelar
        </button>
        <button
          type="submit"
          class="customer-form__submit-btn"
          :disabled="loading"
        >
          {{ loading ? 'Salvando...' : (isEditMode ? 'Salvar' : 'Criar') }}
        </button>
      </div>
    </form>
  </div>
</template>

<style scoped>
.customer-form {
  max-width: 600px;
  margin: 0 auto;
  padding: 1.5rem;
}

.customer-form__title {
  font-size: 1.5rem;
  font-weight: 700;
  color: #111827;
  margin-bottom: 1.5rem;
}

.customer-form__form {
  display: flex;
  flex-direction: column;
}

.customer-form__error {
  background-color: #fef2f2;
  border: 1px solid #fecaca;
  color: #dc2626;
  padding: 0.75rem 1rem;
  border-radius: 0.375rem;
  margin-bottom: 1rem;
  font-size: 0.875rem;
}

.customer-form__fetch-error {
  text-align: center;
  padding: 2rem;
}

.customer-form__not-found {
  text-align: center;
  padding: 2rem;
  background: #fef2f2;
  border: 1px solid #fecaca;
  border-radius: 0.5rem;
}

.customer-form__not-found-message {
  color: #dc2626;
  margin-bottom: 1rem;
  font-size: 1rem;
}

.customer-form__back-link {
  color: #2563eb;
  text-decoration: none;
  font-size: 0.875rem;
  font-weight: 500;
}

.customer-form__back-link:hover {
  text-decoration: underline;
}

.customer-form__fetch-error-message {
  color: #dc2626;
  margin-bottom: 1rem;
}

.customer-form__retry-btn {
  padding: 0.5rem 1rem;
  background-color: #2563eb;
  color: #fff;
  border: none;
  border-radius: 0.375rem;
  cursor: pointer;
  font-size: 0.875rem;
}

.customer-form__retry-btn:hover {
  background-color: #1d4ed8;
}

.customer-form__actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 0.5rem;
}

.customer-form__cancel-btn {
  padding: 0.5rem 1rem;
  background-color: #fff;
  color: #374151;
  border: 1px solid #d1d5db;
  border-radius: 0.375rem;
  cursor: pointer;
  font-size: 0.875rem;
}

.customer-form__cancel-btn:hover {
  background-color: #f9fafb;
}

.customer-form__submit-btn {
  padding: 0.5rem 1.5rem;
  background-color: #2563eb;
  color: #fff;
  border: none;
  border-radius: 0.375rem;
  cursor: pointer;
  font-size: 0.875rem;
  font-weight: 600;
}

.customer-form__submit-btn:hover:not(:disabled) {
  background-color: #1d4ed8;
}

.customer-form__submit-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
