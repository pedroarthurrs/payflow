<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import FormField from '@/components/FormField.vue'
import LoadingSpinner from '@/components/LoadingSpinner.vue'
import { Button } from '@/components/ui/button'
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
  <div class="mx-auto max-w-[600px] p-6">
    <h1 class="text-2xl font-bold text-foreground mb-6">{{ pageTitle }}</h1>

    <div v-if="notFound" class="text-center p-8 bg-card border border-destructive rounded-lg">
      <p class="text-destructive mb-4">Endereço não encontrado</p>
      <router-link :to="`/customers/${customerId}`" class="text-primary text-sm font-medium hover:underline">Voltar para detalhes do cliente</router-link>
    </div>

    <div v-else-if="fetchError" class="text-center p-8">
      <p class="text-destructive mb-4">{{ fetchError }}</p>
      <Button @click="loadAddress">Tentar novamente</Button>
    </div>

    <LoadingSpinner v-else-if="loading && isEditMode && !street" size="lg" />

    <form v-else class="flex flex-col" @submit.prevent="handleSubmit">
      <div v-if="formError" class="bg-card border border-destructive text-destructive px-4 py-3 rounded-md mb-4 text-sm">
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

      <div class="flex justify-end gap-3 mt-2">
        <Button type="button" variant="outline" @click="router.push(`/customers/${customerId}`)">Cancelar</Button>
        <Button type="submit" :disabled="loading">
          {{ loading ? 'Salvando...' : (isEditMode ? 'Salvar' : 'Criar') }}
        </Button>
      </div>
    </form>
  </div>
</template>
