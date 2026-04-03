<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import FormField from '@/components/FormField.vue'
import LoadingSpinner from '@/components/LoadingSpinner.vue'
import { useCrud } from '@/composables/useCrud'
import { useToast } from '@/composables/useToast'
import { useErrorHandler } from '@/composables/useErrorHandler'
import type { Address, CreateAddressRequest, UpdateAddressRequest, ApiError, AddressType } from '@/types'

const route = useRoute()
const router = useRouter()
const toast = useToast()

const customerId = computed(() => route.params.id as string)
const addressId = computed(() => route.params.addressId as string | undefined)
const isEditMode = computed(() => !!addressId.value)

const { getById, create, update, loading } = useCrud<Address>(`/v1/customers/${customerId.value}/addresses`)

const type = ref<AddressType>('shipping')
const postalCode = ref('')
const street = ref('')
const number = ref('')
const complement = ref('')
const neighborhood = ref('')
const city = ref('')
const state = ref('')
const country = ref('BR')
const isMain = ref(false)

const fieldErrors = ref<Record<string, string>>({})
const formError = ref('')
const fetchError = ref('')
const notFound = ref(false)

const { handleError } = useErrorHandler({
  onNotFound: () => {
    notFound.value = true
  },
})

const addressTypeOptions = [
  { value: 'billing', label: 'Cobrança' },
  { value: 'shipping', label: 'Entrega' },
  { value: 'home', label: 'Residencial' },
]

const pageTitle = computed(() => (isEditMode.value ? 'Editar Endereço' : 'Novo Endereço'))

async function loadAddress() {
  if (!addressId.value) return
  fetchError.value = ''
  notFound.value = false
  try {
    const address = await getById(addressId.value)
    type.value = address.type
    postalCode.value = address.postal_code
    street.value = address.street
    number.value = address.number ?? ''
    complement.value = address.complement ?? ''
    neighborhood.value = address.neighborhood ?? ''
    city.value = address.city
    state.value = address.state
    country.value = address.country
    isMain.value = address.is_main
  } catch (e) {
    const err = e as ApiError
    const handled = handleError(err, loadAddress)
    if (!handled) {
      fetchError.value = err.message || 'Erro ao carregar endereço'
    }
  }
}

async function handleSubmit() {
  fieldErrors.value = {}
  formError.value = ''

  try {
    if (isEditMode.value) {
      const payload: UpdateAddressRequest = {
        type: type.value,
        postal_code: postalCode.value,
        street: street.value,
        number: number.value || undefined,
        complement: complement.value || undefined,
        neighborhood: neighborhood.value || undefined,
        city: city.value,
        state: state.value,
        country: country.value,
        is_main: isMain.value,
      }
      await update(addressId.value!, payload as unknown as Partial<Address>)
      toast.success('Endereço atualizado com sucesso')
    } else {
      const payload: CreateAddressRequest = {
        type: type.value,
        postal_code: postalCode.value,
        street: street.value,
        number: number.value || undefined,
        complement: complement.value || undefined,
        neighborhood: neighborhood.value || undefined,
        city: city.value,
        state: state.value,
        country: country.value,
        is_main: isMain.value,
      }
      await create(payload as unknown as Partial<Address>)
      toast.success('Endereço criado com sucesso')
    }
    router.push(`/customers/${customerId.value}`)
  } catch (e) {
    const err = e as ApiError
    if (err.status === 400 && err.validationErrors) {
      fieldErrors.value = { ...err.validationErrors }
    } else {
      const handled = handleError(err)
      if (!handled) {
        formError.value = err.message || 'Erro ao salvar endereço'
      }
    }
  }
}

onMounted(() => {
  if (isEditMode.value) {
    loadAddress()
  }
})
</script>

<template>
  <div class="address-form">
    <h1 class="address-form__title">{{ pageTitle }}</h1>

    <div v-if="notFound" class="address-form__not-found">
      <p class="address-form__not-found-message">Endereço não encontrado</p>
      <router-link :to="`/customers/${customerId}`" class="address-form__back-link">Voltar para detalhes do cliente</router-link>
    </div>

    <div v-else-if="fetchError" class="address-form__fetch-error">
      <p class="address-form__fetch-error-message">{{ fetchError }}</p>
      <button class="address-form__retry-btn" @click="loadAddress">Tentar novamente</button>
    </div>

    <LoadingSpinner v-else-if="loading && isEditMode && !street" size="lg" />

    <form v-else class="address-form__form" @submit.prevent="handleSubmit">
      <div v-if="formError" class="address-form__error">
        {{ formError }}
      </div>

      <FormField
        label="Tipo"
        type="select"
        :modelValue="type"
        @update:modelValue="type = $event as AddressType"
        :error="fieldErrors.type"
        :required="true"
        :options="addressTypeOptions"
      />

      <FormField
        label="CEP"
        :modelValue="postalCode"
        @update:modelValue="postalCode = $event as string"
        :error="fieldErrors.postal_code"
        :required="true"
        placeholder="00000-000"
      />

      <FormField
        label="Logradouro"
        :modelValue="street"
        @update:modelValue="street = $event as string"
        :error="fieldErrors.street"
        :required="true"
        placeholder="Rua, Avenida, etc."
      />

      <FormField
        label="Número"
        :modelValue="number"
        @update:modelValue="number = $event as string"
        :error="fieldErrors.number"
        placeholder="123"
      />

      <FormField
        label="Complemento"
        :modelValue="complement"
        @update:modelValue="complement = $event as string"
        :error="fieldErrors.complement"
        placeholder="Apto, Bloco, etc."
      />

      <FormField
        label="Bairro"
        :modelValue="neighborhood"
        @update:modelValue="neighborhood = $event as string"
        :error="fieldErrors.neighborhood"
        placeholder="Bairro"
      />

      <FormField
        label="Cidade"
        :modelValue="city"
        @update:modelValue="city = $event as string"
        :error="fieldErrors.city"
        :required="true"
        placeholder="Cidade"
      />

      <FormField
        label="Estado"
        :modelValue="state"
        @update:modelValue="state = $event as string"
        :error="fieldErrors.state"
        :required="true"
        placeholder="UF"
      />

      <FormField
        label="País"
        :modelValue="country"
        @update:modelValue="country = $event as string"
        :error="fieldErrors.country"
        placeholder="BR"
      />

      <FormField
        label="Endereço principal"
        type="checkbox"
        :modelValue="isMain"
        @update:modelValue="isMain = $event as boolean"
        :error="fieldErrors.is_main"
      />

      <div class="address-form__actions">
        <button
          type="button"
          class="address-form__cancel-btn"
          @click="router.push(`/customers/${customerId}`)"
        >
          Cancelar
        </button>
        <button
          type="submit"
          class="address-form__submit-btn"
          :disabled="loading"
        >
          {{ loading ? 'Salvando...' : (isEditMode ? 'Salvar' : 'Criar') }}
        </button>
      </div>
    </form>
  </div>
</template>

<style scoped>
.address-form {
  max-width: 600px;
  margin: 0 auto;
  padding: 1.5rem;
}

.address-form__title {
  font-size: 1.5rem;
  font-weight: 700;
  color: #111827;
  margin-bottom: 1.5rem;
}

.address-form__form {
  display: flex;
  flex-direction: column;
}

.address-form__error {
  background-color: #fef2f2;
  border: 1px solid #fecaca;
  color: #dc2626;
  padding: 0.75rem 1rem;
  border-radius: 0.375rem;
  margin-bottom: 1rem;
  font-size: 0.875rem;
}

.address-form__fetch-error {
  text-align: center;
  padding: 2rem;
}

.address-form__not-found {
  text-align: center;
  padding: 2rem;
  background: #fef2f2;
  border: 1px solid #fecaca;
  border-radius: 0.5rem;
}

.address-form__not-found-message {
  color: #dc2626;
  margin-bottom: 1rem;
  font-size: 1rem;
}

.address-form__back-link {
  color: #2563eb;
  text-decoration: none;
  font-size: 0.875rem;
  font-weight: 500;
}

.address-form__back-link:hover {
  text-decoration: underline;
}

.address-form__fetch-error-message {
  color: #dc2626;
  margin-bottom: 1rem;
}

.address-form__retry-btn {
  padding: 0.5rem 1rem;
  background-color: #2563eb;
  color: #fff;
  border: none;
  border-radius: 0.375rem;
  cursor: pointer;
  font-size: 0.875rem;
}

.address-form__retry-btn:hover {
  background-color: #1d4ed8;
}

.address-form__actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 0.5rem;
}

.address-form__cancel-btn {
  padding: 0.5rem 1rem;
  background-color: #fff;
  color: #374151;
  border: 1px solid #d1d5db;
  border-radius: 0.375rem;
  cursor: pointer;
  font-size: 0.875rem;
}

.address-form__cancel-btn:hover {
  background-color: #f9fafb;
}

.address-form__submit-btn {
  padding: 0.5rem 1.5rem;
  background-color: #2563eb;
  color: #fff;
  border: none;
  border-radius: 0.375rem;
  cursor: pointer;
  font-size: 0.875rem;
  font-weight: 600;
}

.address-form__submit-btn:hover:not(:disabled) {
  background-color: #1d4ed8;
}

.address-form__submit-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
