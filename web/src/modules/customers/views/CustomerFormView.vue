<script setup lang="ts">
import { api } from '@/core/api/http'
import FormField from '@/core/components/FormField.vue'
import LoadingSpinner from '@/core/components/LoadingSpinner.vue'
import { Button } from '@/core/components/ui/button'
import { useErrorHandler } from '@/core/composables/useErrorHandler'
import { useToast } from '@/core/composables/useToast'
import type { ApiError, Customer } from '@/core/types'
import { useCreateCustomer, useUpdateCustomer } from '@/modules/customers/composables/useCustomers'
import { createCustomerSchema, updateCustomerSchema } from '@/modules/customers/schemas'
import { toTypedSchema } from '@vee-validate/zod'
import { useForm } from 'vee-validate'
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()
const toast = useToast()

const customerId = computed(() => route.params.id as string | undefined)
const isEditMode = computed(() => !!customerId.value)

const schema = computed(() => isEditMode.value ? updateCustomerSchema : createCustomerSchema)

const { handleSubmit, errors, defineField, setFieldError, resetForm, setValues } = useForm({
  validationSchema: computed(() => toTypedSchema(schema.value)),
})

const [name] = defineField('name')
const [email] = defineField('email')
const [phone] = defineField('phone')
const [document] = defineField('document')
const [documentType] = defineField('document_type')

const metadata = ref('')
const metadataError = ref('')
const formError = ref('')
const fetchError = ref('')
const notFound = ref(false)
const fetchLoading = ref(false)

const createMutation = useCreateCustomer()
const updateMutation = useUpdateCustomer()

const loading = computed(() => createMutation.isPending.value || updateMutation.isPending.value)

const { handleError } = useErrorHandler({
  onNotFound: () => {
    notFound.value = true
  },
})

const documentTypeOptions = [
  { value: 'cpf', label: 'CPF' },
  { value: 'cnpj', label: 'CNPJ' },
]

const pageTitle = computed(() => (isEditMode.value ? 'Editar Cliente' : 'Novo Cliente'))

function parseMetadata(): Record<string, unknown> | undefined | null {
  const trimmed = metadata.value.trim()
  if (!trimmed) return undefined
  try {
    return JSON.parse(trimmed)
  } catch {
    metadataError.value = 'JSON inválido'
    return null
  }
}

async function loadCustomer() {
  if (!customerId.value) return
  fetchError.value = ''
  notFound.value = false
  fetchLoading.value = true
  try {
    const customer = await api.get<Customer>(`/v1/customers/${customerId.value}`)
    setValues({
      name: customer.name,
      email: customer.email,
      phone: customer.phone ?? '',
      document: customer.document,
      document_type: customer.document_type,
    })
    metadata.value = customer.metadata ? JSON.stringify(customer.metadata, null, 2) : ''
  } catch (e) {
    const err = e as ApiError
    const handled = handleError(err, loadCustomer)
    if (!handled) {
      fetchError.value = err.message || 'Erro ao carregar cliente'
    }
  } finally {
    fetchLoading.value = false
  }
}

const onSubmit = handleSubmit(async (values) => {
  formError.value = ''
  metadataError.value = ''

  const parsedMetadata = parseMetadata()
  if (parsedMetadata === null) return

  try {
    if (isEditMode.value) {
      const payload = {
        name: values.name,
        email: values.email,
        document: values.document,
        document_type: values.document_type,
        phone: values.phone || undefined,
        metadata: parsedMetadata,
      }
      await updateMutation.mutateAsync({ id: customerId.value!, data: payload })
      toast.success('Cliente atualizado com sucesso')
      router.push(`/customers/${customerId.value}`)
    } else {
      const payload = {
        name: values.name,
        email: values.email,
        document: values.document,
        document_type: values.document_type,
        phone: values.phone || undefined,
        metadata: parsedMetadata,
      }
      const created = await createMutation.mutateAsync(payload)
      toast.success('Cliente criado com sucesso')
      resetForm()
      router.push(`/customers/${created.id}`)
    }
  } catch (e) {
    const err = e as ApiError
    if (err.status === 400 && err.validationErrors) {
      for (const [field, message] of Object.entries(err.validationErrors)) {
        setFieldError(field, message)
      }
    } else if (err.status === 409) {
      formError.value = err.message || 'Conflito: registro já existe'
    } else {
      const handled = handleError(err)
      if (!handled) {
        formError.value = err.message || 'Erro ao salvar cliente'
      }
    }
  }
})

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

    <LoadingSpinner v-else-if="fetchLoading && isEditMode && !name" size="lg" />

    <form v-else class="flex flex-col" @submit.prevent="onSubmit">
      <div v-if="formError" class="bg-card border border-destructive text-destructive px-4 py-3 rounded-md mb-4 text-sm">
        {{ formError }}
      </div>

      <FormField
        label="Nome"
        :modelValue="name ?? ''"
        @update:modelValue="name = $event as string"
        :error="errors.name"
        :required="true"
        placeholder="Nome do cliente"
      />

      <FormField
        label="Email"
        type="email"
        :modelValue="email ?? ''"
        @update:modelValue="email = $event as string"
        :error="errors.email"
        :required="true"
        placeholder="email@exemplo.com"
      />

      <FormField
        label="Telefone"
        type="tel"
        :modelValue="phone ?? ''"
        @update:modelValue="phone = $event as string"
        :error="errors.phone"
        placeholder="(00) 00000-0000"
      />

      <FormField
        label="Documento"
        :modelValue="document ?? ''"
        @update:modelValue="document = $event as string"
        :error="errors.document"
        :required="true"
        placeholder="Número do documento"
      />

      <FormField
        label="Tipo de Documento"
        type="select"
        :modelValue="documentType ?? 'cpf'"
        @update:modelValue="documentType = $event as string"
        :error="errors.document_type"
        :required="true"
        :options="documentTypeOptions"
      />

      <FormField
        label="Metadata (JSON)"
        type="textarea"
        :modelValue="metadata"
        @update:modelValue="metadata = $event as string"
        :error="metadataError"
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
