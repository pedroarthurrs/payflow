<script setup lang="ts">
import { api } from '@/core/api/http'
import FormField from '@/core/components/FormField.vue'
import LoadingSpinner from '@/core/components/LoadingSpinner.vue'
import { Button } from '@/core/components/ui/button'
import { useErrorHandler } from '@/core/composables/useErrorHandler'
import { useToast } from '@/core/composables/useToast'
import type { Address, ApiError } from '@/core/types'
import { useCreateAddress, useUpdateAddress } from '@/modules/customers/composables/useCustomers'
import { addressSchema } from '@/modules/customers/schemas'
import { toTypedSchema } from '@vee-validate/zod'
import { useForm } from 'vee-validate'
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()
const toast = useToast()

const customerId = computed(() => route.params.id as string)
const addressId = computed(() => route.params.addressId as string | undefined)
const isEditMode = computed(() => !!addressId.value)

const { handleSubmit, errors, defineField, setFieldError, resetForm, setValues } = useForm({
  validationSchema: toTypedSchema(addressSchema),
})

const [type] = defineField('type')
const [postalCode] = defineField('postal_code')
const [street] = defineField('street')
const [number] = defineField('number')
const [complement] = defineField('complement')
const [neighborhood] = defineField('neighborhood')
const [city] = defineField('city')
const [state] = defineField('state')
const [country] = defineField('country')
const [isMain] = defineField('is_main')

const formError = ref('')
const fetchError = ref('')
const notFound = ref(false)
const fetchLoading = ref(false)

const createMutation = useCreateAddress(customerId)
const updateMutation = useUpdateAddress(customerId)

const loading = computed(() => createMutation.isPending.value || updateMutation.isPending.value)

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
  fetchLoading.value = true
  try {
    const address = await api.get<Address>(`/v1/customers/${customerId.value}/addresses/${addressId.value}`)
    setValues({
      type: address.type,
      postal_code: address.postal_code,
      street: address.street,
      number: address.number ?? '',
      complement: address.complement ?? '',
      neighborhood: address.neighborhood ?? '',
      city: address.city,
      state: address.state,
      country: address.country,
      is_main: address.is_main,
    })
  } catch (e) {
    const err = e as ApiError
    const handled = handleError(err, loadAddress)
    if (!handled) {
      fetchError.value = err.message || 'Erro ao carregar endereço'
    }
  } finally {
    fetchLoading.value = false
  }
}

const onSubmit = handleSubmit(async (values) => {
  formError.value = ''

  try {
    if (isEditMode.value) {
      const payload = {
        type: values.type,
        postal_code: values.postal_code,
        street: values.street,
        number: values.number || undefined,
        complement: values.complement || undefined,
        neighborhood: values.neighborhood || undefined,
        city: values.city,
        state: values.state,
        country: values.country,
        is_main: values.is_main,
      }
      await updateMutation.mutateAsync({ addressId: addressId.value!, data: payload })
      toast.success('Endereço atualizado com sucesso')
    } else {
      const payload = {
        type: values.type,
        postal_code: values.postal_code,
        street: values.street,
        number: values.number || undefined,
        complement: values.complement || undefined,
        neighborhood: values.neighborhood || undefined,
        city: values.city,
        state: values.state,
        country: values.country,
        is_main: values.is_main,
      }
      await createMutation.mutateAsync(payload)
      toast.success('Endereço criado com sucesso')
      resetForm()
    }
    router.push(`/customers/${customerId.value}`)
  } catch (e) {
    const err = e as ApiError
    if (err.status === 400 && err.validationErrors) {
      for (const [field, message] of Object.entries(err.validationErrors)) {
        setFieldError(field, message)
      }
    } else {
      const handled = handleError(err)
      if (!handled) {
        formError.value = err.message || 'Erro ao salvar endereço'
      }
    }
  }
})

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

    <LoadingSpinner v-else-if="fetchLoading && isEditMode && !street" size="lg" />

    <form v-else class="flex flex-col" @submit.prevent="onSubmit">
      <div v-if="formError" class="bg-card border border-destructive text-destructive px-4 py-3 rounded-md mb-4 text-sm">
        {{ formError }}
      </div>

      <FormField
        label="Tipo"
        type="select"
        :modelValue="type ?? 'shipping'"
        @update:modelValue="type = $event as string"
        :error="errors.type"
        :required="true"
        :options="addressTypeOptions"
      />

      <FormField
        label="CEP"
        :modelValue="postalCode ?? ''"
        @update:modelValue="postalCode = $event as string"
        :error="errors.postal_code"
        :required="true"
        placeholder="00000-000"
      />

      <FormField
        label="Logradouro"
        :modelValue="street ?? ''"
        @update:modelValue="street = $event as string"
        :error="errors.street"
        :required="true"
        placeholder="Rua, Avenida, etc."
      />

      <FormField
        label="Número"
        :modelValue="number ?? ''"
        @update:modelValue="number = $event as string"
        :error="errors.number"
        placeholder="123"
      />

      <FormField
        label="Complemento"
        :modelValue="complement ?? ''"
        @update:modelValue="complement = $event as string"
        :error="errors.complement"
        placeholder="Apto, Bloco, etc."
      />

      <FormField
        label="Bairro"
        :modelValue="neighborhood ?? ''"
        @update:modelValue="neighborhood = $event as string"
        :error="errors.neighborhood"
        placeholder="Bairro"
      />

      <FormField
        label="Cidade"
        :modelValue="city ?? ''"
        @update:modelValue="city = $event as string"
        :error="errors.city"
        :required="true"
        placeholder="Cidade"
      />

      <FormField
        label="Estado"
        :modelValue="state ?? ''"
        @update:modelValue="state = $event as string"
        :error="errors.state"
        :required="true"
        placeholder="UF"
      />

      <FormField
        label="País"
        :modelValue="country ?? 'BR'"
        @update:modelValue="country = $event as string"
        :error="errors.country"
        placeholder="BR"
      />

      <FormField
        label="Endereço principal"
        type="checkbox"
        :modelValue="isMain ?? false"
        @update:modelValue="isMain = $event as boolean"
        :error="errors.is_main"
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
