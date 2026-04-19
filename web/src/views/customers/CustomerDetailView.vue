<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import DataTable from '@/components/DataTable.vue'
import type { Column } from '@/components/DataTable.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import LoadingSpinner from '@/components/LoadingSpinner.vue'
import { Button } from '@/components/ui/button'
import { useCrud } from '@/composables/useCrud'
import { useToast } from '@/composables/useToast'
import { useErrorHandler } from '@/composables/useErrorHandler'
import { formatDocument, formatPhone, formatMetadata } from '@/lib/utils'
import type { Customer, Address, ApiError } from '@/types'

const route = useRoute()
const router = useRouter()
const toast = useToast()

const customerId = route.params.id as string

const customerCrud = useCrud<Customer>('/v1/customers')
const addressCrud = useCrud<Address>(`/v1/customers/${customerId}/addresses`)

const customer = ref<Customer | null>(null)
const addresses = ref<Address[]>([])
const fetchError = ref<string | null>(null)
const notFound = ref(false)
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

async function fetchCustomer() {
  fetchError.value = null
  notFound.value = false
  try {
    customer.value = await customerCrud.getById(customerId)
  } catch (e: any) {
    const apiErr = e as ApiError
    const handled = handleError(apiErr, fetchCustomer)
    if (!handled) {
      fetchError.value = apiErr?.message || 'Erro ao carregar cliente'
    }
  }
}

async function fetchAddresses() {
  addressFetchError.value = null
  try {
    addresses.value = await addressCrud.list()
  } catch (e: any) {
    const apiErr = e as ApiError
    const handled = handleError(apiErr)
    if (!handled) {
      addressFetchError.value = apiErr?.message || 'Erro ao carregar endereços'
    }
  }
}

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
    await customerCrud.remove(customerId, hardDelete.value || undefined)
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
    await addressCrud.remove(addressToDelete.value.id)
    toast.success('Endereço excluído com sucesso')
    showDeleteAddress.value = false
    addressToDelete.value = null
    await fetchAddresses()
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

onMounted(() => {
  fetchCustomer()
  fetchAddresses()
})
</script>

<template>
  <div class="p-6">
    <!-- Loading -->
    <div v-if="customerCrud.loading.value && !customer" class="flex justify-center py-12">
      <LoadingSpinner size="lg" />
    </div>

    <!-- Not Found -->
    <div v-else-if="notFound" class="text-center p-8 bg-card border border-destructive rounded-lg">
      <p class="text-destructive mb-4">Cliente não encontrado</p>
      <router-link to="/customers" class="text-primary text-sm font-medium hover:underline">Voltar para listagem</router-link>
    </div>

    <!-- Fetch Error -->
    <div v-else-if="fetchError" class="text-center p-8 bg-card border border-destructive rounded-lg">
      <p class="text-destructive mb-4">{{ fetchError }}</p>
      <Button @click="fetchCustomer">Tentar novamente</Button>
    </div>

    <!-- Customer Data -->
    <template v-else-if="customer">
      <div class="flex items-center justify-between mb-6">
        <h1 class="text-2xl font-semibold text-foreground">{{ customer.name }}</h1>
        <div class="flex gap-3">
          <Button @click="navigateToEdit">Editar</Button>
          <Button variant="destructive" @click="openDeleteCustomerDialog">Excluir</Button>
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
          <Button @click="navigateToNewAddress">Novo Endereço</Button>
        </div>

        <div v-if="addressFetchError" class="text-center p-8 bg-card border border-destructive rounded-lg">
          <p class="text-destructive mb-4">{{ addressFetchError }}</p>
          <Button @click="fetchAddresses">Tentar novamente</Button>
        </div>

        <DataTable
          v-else
          :columns="addressColumns"
          :data="addresses"
          :loading="addressCrud.loading.value"
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
