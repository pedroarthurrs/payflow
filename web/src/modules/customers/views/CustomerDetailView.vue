<script setup lang="ts">
import ConfirmDialog from '@/core/components/ConfirmDialog.vue'
import type { Column } from '@/core/components/DataTable.vue'
import DataTable from '@/core/components/DataTable.vue'
import LoadingSpinner from '@/core/components/LoadingSpinner.vue'
import { Button } from '@/core/components/ui/button'
import { useErrorHandler } from '@/core/composables/useErrorHandler'
import { useToast } from '@/core/composables/useToast'
import Can from '@/core/rbac/Can.vue'
import { useRBAC } from '@/core/rbac/composable'
import type { Address, ApiError } from '@/core/types'
import { formatDocument, formatMetadata, formatPhone } from '@/core/utils/formatting'
import { useAddressList, useCustomerDetail, useDeleteAddress, useDeleteCustomer } from '@/modules/customers/composables/useCustomers'
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()
const toast = useToast()
const { can } = useRBAC()

const customerId = route.params.id as string
const customerIdRef = ref(customerId)

const { data: customer, isPending: customerLoading, isError: customerIsError, error: customerError } = useCustomerDetail(customerIdRef)
const { data: addressData, isPending: addressLoading } = useAddressList(customerIdRef)
const deleteCustomerMutation = useDeleteCustomer()
const deleteAddressMutation = useDeleteAddress(customerIdRef)

const addresses = computed(() => addressData.value?.data ?? [])

const notFound = ref(false)
const fetchError = ref<string | null>(null)
const addressFetchError = ref<string | null>(null)

const { handleError } = useErrorHandler({
  onNotFound: () => {
    notFound.value = true
  },
})

const showDeleteCustomer = ref(false)
const hardDelete = ref(false)
const deletingCustomer = ref(false)

const showDeleteAddress = ref(false)
const addressToDelete = ref<Address | null>(null)
const deletingAddress = ref(false)

const addressColumns: Column[] = [
  { key: 'type', label: 'Tipo' },
  { key: 'street', label: 'Logradouro' },
  { key: 'city', label: 'Cidade' },
  { key: 'state', label: 'Estado' },
  { key: 'postal_code', label: 'CEP' },
  {
    key: 'is_main',
    label: 'Principal',
    format: (value: boolean) => (value ? 'Sim' : 'Não'),
  },
]

function navigateToEdit() {
  router.push(`/customers/${customerId}/edit`)
}

function navigateToNewAddress() {
  router.push(`/customers/${customerId}/addresses/new`)
}

function openDeleteCustomerDialog() {
  hardDelete.value = false
  showDeleteCustomer.value = true
}

async function confirmDeleteCustomer() {
  deletingCustomer.value = true
  try {
    await deleteCustomerMutation.mutateAsync(customerId)
    toast.success('Cliente excluído com sucesso')
    showDeleteCustomer.value = false
    router.push('/customers')
  } catch (e: any) {
    const apiErr = e as ApiError
    showDeleteCustomer.value = false
    const handled = handleError(apiErr)
    if (!handled) {
      toast.error(apiErr?.message || 'Erro ao excluir cliente')
    }
  } finally {
    deletingCustomer.value = false
  }
}

function openDeleteAddressDialog(address: Address) {
  addressToDelete.value = address
  showDeleteAddress.value = true
}

async function confirmDeleteAddress() {
  if (!addressToDelete.value) return
  deletingAddress.value = true
  try {
    await deleteAddressMutation.mutateAsync(addressToDelete.value.id)
    toast.success('Endereço excluído com sucesso')
    showDeleteAddress.value = false
    addressToDelete.value = null
  } catch (e: any) {
    const apiErr = e as ApiError
    showDeleteAddress.value = false
    const handled = handleError(apiErr)
    if (!handled) {
      toast.error(apiErr?.message || 'Erro ao excluir endereço')
    }
  } finally {
    deletingAddress.value = false
  }
}

function handleEditAddress(address: Address) {
  router.push(`/customers/${customerId}/addresses/${address.id}/edit`)
}

function formatDate(value: string) {
  if (!value) return ''
  return new Date(value).toLocaleDateString('pt-BR')
}
</script>

<template>
  <div class="p-6">
    <!-- Loading -->
    <div v-if="customerLoading && !customer" class="flex justify-center py-12">
      <LoadingSpinner size="lg" />
    </div>

    <!-- Not Found -->
    <div v-else-if="notFound" class="text-center p-8 bg-card border border-destructive rounded-lg">
      <p class="text-destructive mb-4">Cliente não encontrado</p>
      <router-link to="/customers" class="text-primary text-sm font-medium hover:underline">Voltar para listagem</router-link>
    </div>

    <!-- Fetch Error -->
    <div v-else-if="customerIsError" class="text-center p-8 bg-card border border-destructive rounded-lg">
      <p class="text-destructive mb-4">{{ (customerError as any)?.message || 'Erro ao carregar cliente' }}</p>
    </div>

    <!-- Customer Data -->
    <template v-else-if="customer">
      <div class="flex items-center justify-between mb-6">
        <h1 class="text-2xl font-semibold text-foreground">{{ customer.name }}</h1>
        <div class="flex gap-3">
          <Can permission="customers:write">
            <Button @click="navigateToEdit">Editar</Button>
          </Can>
          <Can permission="customers:delete">
            <Button variant="destructive" @click="openDeleteCustomerDialog">Excluir</Button>
          </Can>
        </div>
      </div>

      <div class="bg-card border border-border rounded-lg p-6 mb-8">
        <dl class="grid grid-cols-2 gap-4">
          <div>
            <dt class="text-xs font-semibold text-muted-foreground uppercase tracking-wide mb-1">Email</dt>
            <dd class="text-[0.9375rem] text-foreground m-0">{{ customer.email }}</dd>
          </div>
          <div>
            <dt class="text-xs font-semibold text-muted-foreground uppercase tracking-wide mb-1">Telefone</dt>
            <dd class="text-[0.9375rem] text-foreground m-0">{{ customer.phone ? formatPhone(customer.phone) : '—' }}</dd>
          </div>
          <div>
            <dt class="text-xs font-semibold text-muted-foreground uppercase tracking-wide mb-1">Documento</dt>
            <dd class="text-[0.9375rem] text-foreground m-0">{{ formatDocument(customer.document, customer.document_type) }}</dd>
          </div>
          <div>
            <dt class="text-xs font-semibold text-muted-foreground uppercase tracking-wide mb-1">Tipo de Documento</dt>
            <dd class="text-[0.9375rem] text-foreground m-0">{{ customer.document_type }}</dd>
          </div>
          <div>
            <dt class="text-xs font-semibold text-muted-foreground uppercase tracking-wide mb-1">Status</dt>
            <dd class="text-[0.9375rem] text-foreground m-0">{{ customer.status }}</dd>
          </div>
          <div>
            <dt class="text-xs font-semibold text-muted-foreground uppercase tracking-wide mb-1">Criado em</dt>
            <dd class="text-[0.9375rem] text-foreground m-0">{{ formatDate(customer.created_at) }}</dd>
          </div>
          <div>
            <dt class="text-xs font-semibold text-muted-foreground uppercase tracking-wide mb-1">Atualizado em</dt>
            <dd class="text-[0.9375rem] text-foreground m-0">{{ formatDate(customer.updated_at) }}</dd>
          </div>
          <div class="col-span-2">
            <dt class="text-xs font-semibold text-muted-foreground uppercase tracking-wide mb-1">Metadata</dt>
            <dd class="m-0">
              <span v-if="!customer.metadata || Object.keys(customer.metadata).length === 0" class="text-muted-foreground">—</span>
              <dl v-else class="grid grid-cols-1 gap-1 bg-muted p-3 rounded-md border border-border">
                <div v-for="entry in formatMetadata(customer.metadata)" :key="entry.key" class="flex gap-2 text-[0.8125rem]">
                  <dt class="font-semibold text-muted-foreground whitespace-nowrap">{{ entry.key }}:</dt>
                  <dd class="text-foreground m-0 break-all">{{ entry.value }}</dd>
                </div>
              </dl>
            </dd>
          </div>
        </dl>
      </div>

      <!-- Addresses Section -->
      <div class="mt-8">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-xl font-semibold text-foreground">Endereços</h2>
          <Can permission="customers:write">
            <Button @click="navigateToNewAddress">Novo Endereço</Button>
          </Can>
        </div>

        <div v-if="addressFetchError" class="text-center p-8 bg-card border border-destructive rounded-lg">
          <p class="text-destructive mb-4">{{ addressFetchError }}</p>
        </div>

        <DataTable
          v-else
          :columns="addressColumns"
          :data="addresses"
          :loading="addressLoading"
          @edit="handleEditAddress"
          @delete="openDeleteAddressDialog"
        />
      </div>
    </template>

    <!-- Delete Customer Dialog -->
    <ConfirmDialog
      :visible="showDeleteCustomer"
      title="Excluir Cliente"
      message="Tem certeza que deseja excluir este cliente? Esta ação não pode ser desfeita."
      confirm-label="Excluir"
      variant="danger"
      @confirm="confirmDeleteCustomer"
      @cancel="showDeleteCustomer = false"
    >
      <label class="flex items-center gap-2 text-sm text-foreground cursor-pointer mt-2">
        <input v-model="hardDelete" type="checkbox" class="size-4 cursor-pointer" />
        Exclusão permanente (hard delete)
      </label>
    </ConfirmDialog>

    <!-- Delete Address Dialog -->
    <ConfirmDialog
      :visible="showDeleteAddress"
      title="Excluir Endereço"
      message="Tem certeza que deseja excluir este endereço?"
      confirm-label="Excluir"
      variant="danger"
      @confirm="confirmDeleteAddress"
      @cancel="showDeleteAddress = false"
    />
  </div>
</template>
