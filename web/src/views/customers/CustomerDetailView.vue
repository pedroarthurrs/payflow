<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import DataTable from '@/components/DataTable.vue'
import type { Column } from '@/components/DataTable.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import LoadingSpinner from '@/components/LoadingSpinner.vue'
import { useCrud } from '@/composables/useCrud'
import { useToast } from '@/composables/useToast'
import { useErrorHandler } from '@/composables/useErrorHandler'
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

function formatMetadata(metadata?: Record<string, unknown>) {
  if (!metadata) return '—'
  return JSON.stringify(metadata, null, 2)
}

onMounted(() => {
  fetchCustomer()
  fetchAddresses()
})
</script>

<template>
  <div class="customer-detail">
    <!-- Loading -->
    <div v-if="customerCrud.loading.value && !customer" class="customer-detail__loading">
      <LoadingSpinner size="lg" />
    </div>

    <!-- Fetch Error -->
    <div v-else-if="notFound" class="customer-detail__not-found">
      <p class="customer-detail__not-found-message">Cliente não encontrado</p>
      <router-link to="/customers" class="customer-detail__back-link">Voltar para listagem</router-link>
    </div>

    <div v-else-if="fetchError" class="customer-detail__error">
      <p class="customer-detail__error-message">{{ fetchError }}</p>
      <button class="customer-detail__retry-btn" @click="fetchCustomer">Tentar novamente</button>
    </div>

    <!-- Customer Data -->
    <template v-else-if="customer">
      <div class="customer-detail__header">
        <h1 class="customer-detail__title">{{ customer.name }}</h1>
        <div class="customer-detail__actions">
          <button class="customer-detail__edit-btn" @click="navigateToEdit">Editar</button>
          <button class="customer-detail__delete-btn" @click="openDeleteCustomerDialog">Excluir</button>
        </div>
      </div>

      <div class="customer-detail__info">
        <dl class="customer-detail__fields">
          <div class="customer-detail__field">
            <dt>Email</dt>
            <dd>{{ customer.email }}</dd>
          </div>
          <div class="customer-detail__field">
            <dt>Telefone</dt>
            <dd>{{ customer.phone || '—' }}</dd>
          </div>
          <div class="customer-detail__field">
            <dt>Documento</dt>
            <dd>{{ customer.document }}</dd>
          </div>
          <div class="customer-detail__field">
            <dt>Tipo de Documento</dt>
            <dd>{{ customer.document_type }}</dd>
          </div>
          <div class="customer-detail__field">
            <dt>Status</dt>
            <dd>{{ customer.status }}</dd>
          </div>
          <div class="customer-detail__field">
            <dt>Criado em</dt>
            <dd>{{ formatDate(customer.created_at) }}</dd>
          </div>
          <div class="customer-detail__field">
            <dt>Atualizado em</dt>
            <dd>{{ formatDate(customer.updated_at) }}</dd>
          </div>
          <div class="customer-detail__field customer-detail__field--full">
            <dt>Metadata</dt>
            <dd class="customer-detail__metadata">{{ formatMetadata(customer.metadata) }}</dd>
          </div>
        </dl>
      </div>

      <!-- Addresses Section -->
      <div class="customer-detail__addresses">
        <div class="customer-detail__addresses-header">
          <h2 class="customer-detail__addresses-title">Endereços</h2>
          <button class="customer-detail__new-address-btn" @click="navigateToNewAddress">
            Novo Endereço
          </button>
        </div>

        <div v-if="addressFetchError" class="customer-detail__address-error">
          <p class="customer-detail__address-error-message">{{ addressFetchError }}</p>
          <button class="customer-detail__address-retry-btn" @click="fetchAddresses">Tentar novamente</button>
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
      <label class="customer-detail__hard-delete">
        <input v-model="hardDelete" type="checkbox" />
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

<style scoped>
.customer-detail {
  padding: 1.5rem;
}

.customer-detail__loading {
  display: flex;
  justify-content: center;
  padding: 3rem;
}

.customer-detail__error {
  text-align: center;
  padding: 2rem;
  background: var(--card);
  border: 1px solid var(--destructive);
  border-radius: 0.5rem;
}

.customer-detail__error-message {
  color: var(--destructive);
  margin-bottom: 1rem;
}

.customer-detail__not-found {
  text-align: center;
  padding: 2rem;
  background: var(--card);
  border: 1px solid var(--destructive);
  border-radius: 0.5rem;
}

.customer-detail__not-found-message {
  color: var(--destructive);
  margin-bottom: 1rem;
  font-size: 1rem;
}

.customer-detail__back-link {
  color: var(--primary);
  text-decoration: none;
  font-size: 0.875rem;
  font-weight: 500;
}

.customer-detail__back-link:hover {
  text-decoration: underline;
}

.customer-detail__retry-btn,
.customer-detail__address-retry-btn {
  padding: 0.5rem 1rem;
  background: var(--primary);
  color: var(--primary-foreground);
  border: none;
  border-radius: 0.375rem;
  cursor: pointer;
  font-size: 0.875rem;
}

.customer-detail__retry-btn:hover,
.customer-detail__address-retry-btn:hover {
  filter: brightness(1.08);
}

.customer-detail__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
}

.customer-detail__title {
  font-size: 1.5rem;
  font-weight: 600;
  color: var(--foreground);
}

.customer-detail__actions {
  display: flex;
  gap: 0.75rem;
}

.customer-detail__edit-btn {
  padding: 0.5rem 1rem;
  background: var(--primary);
  color: var(--primary-foreground);
  border: none;
  border-radius: 0.375rem;
  cursor: pointer;
  font-size: 0.875rem;
  font-weight: 500;
}

.customer-detail__edit-btn:hover {
  filter: brightness(1.08);
}

.customer-detail__delete-btn {
  padding: 0.5rem 1rem;
  background: var(--destructive);
  color: var(--destructive-foreground);
  border: none;
  border-radius: 0.375rem;
  cursor: pointer;
  font-size: 0.875rem;
  font-weight: 500;
}

.customer-detail__delete-btn:hover {
  filter: brightness(1.08);
}

.customer-detail__info {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: 0.5rem;
  padding: 1.5rem;
  margin-bottom: 2rem;
}

.customer-detail__fields {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 1rem;
}

.customer-detail__field dt {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--muted-foreground);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: 0.25rem;
}

.customer-detail__field dd {
  font-size: 0.9375rem;
  color: var(--foreground);
  margin: 0;
}

.customer-detail__field--full {
  grid-column: 1 / -1;
}

.customer-detail__metadata {
  white-space: pre-wrap;
  font-family: monospace;
  font-size: 0.8125rem;
  background: var(--muted);
  padding: 0.75rem;
  border-radius: 0.375rem;
  border: 1px solid var(--border);
}

.customer-detail__addresses {
  margin-top: 2rem;
}

.customer-detail__addresses-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.customer-detail__addresses-title {
  font-size: 1.25rem;
  font-weight: 600;
  color: var(--foreground);
}

.customer-detail__new-address-btn {
  padding: 0.5rem 1rem;
  background: var(--primary);
  color: var(--primary-foreground);
  border: none;
  border-radius: 0.375rem;
  cursor: pointer;
  font-size: 0.875rem;
  font-weight: 500;
}

.customer-detail__new-address-btn:hover {
  filter: brightness(1.08);
}

.customer-detail__address-error {
  text-align: center;
  padding: 2rem;
  background: var(--card);
  border: 1px solid var(--destructive);
  border-radius: 0.5rem;
}

.customer-detail__address-error-message {
  color: var(--destructive);
  margin-bottom: 1rem;
}

.customer-detail__hard-delete {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;
  color: var(--foreground);
  cursor: pointer;
  margin-top: 0.5rem;
}

.customer-detail__hard-delete input[type="checkbox"] {
  width: 1rem;
  height: 1rem;
  cursor: pointer;
}
</style>
