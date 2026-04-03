<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import DataTable from '@/components/DataTable.vue'
import type { Column } from '@/components/DataTable.vue'
import type { Customer, ApiError } from '@/types'
import { useCrud } from '@/composables/useCrud'
import { useErrorHandler } from '@/composables/useErrorHandler'

const router = useRouter()
const { list, loading } = useCrud<Customer>('/v1/customers')
const customers = ref<Customer[]>([])
const fetchError = ref<string | null>(null)
const { handleError } = useErrorHandler()

const columns: Column[] = [
  { key: 'name', label: 'Nome' },
  { key: 'email', label: 'Email' },
  { key: 'document', label: 'Documento' },
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

async function fetchCustomers() {
  fetchError.value = null
  try {
    customers.value = await list()
  } catch (e: any) {
    const apiErr = e as ApiError
    const handled = handleError(apiErr, fetchCustomers)
    if (!handled) {
      fetchError.value = apiErr?.message || 'Erro ao carregar clientes'
    }
  }
}

function handleRowClick(row: Customer) {
  router.push(`/customers/${row.id}`)
}

function navigateToNew() {
  router.push('/customers/new')
}

onMounted(fetchCustomers)
</script>

<template>
  <div class="customer-list">
    <div class="customer-list__header">
      <h1 class="customer-list__title">Clientes</h1>
      <button class="customer-list__new-btn" @click="navigateToNew">
        Novo Cliente
      </button>
    </div>

    <div v-if="fetchError" class="customer-list__error">
      <p class="customer-list__error-message">{{ fetchError }}</p>
      <button class="customer-list__retry-btn" @click="fetchCustomers">Tentar novamente</button>
    </div>

    <DataTable
      v-else
      :columns="columns"
      :data="customers"
      :loading="loading"
      @row-click="handleRowClick"
    />
  </div>
</template>

<style scoped>
.customer-list {
  padding: 1.5rem;
}

.customer-list__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
}

.customer-list__title {
  font-size: 1.5rem;
  font-weight: 600;
  color: #1f2937;
}

.customer-list__new-btn {
  padding: 0.5rem 1rem;
  background: #2563eb;
  color: #fff;
  border: none;
  border-radius: 0.375rem;
  cursor: pointer;
  font-size: 0.875rem;
  font-weight: 500;
}

.customer-list__new-btn:hover {
  background: #1d4ed8;
}

.customer-list__error {
  text-align: center;
  padding: 2rem;
  background: #fef2f2;
  border: 1px solid #fecaca;
  border-radius: 0.5rem;
}

.customer-list__error-message {
  color: #dc2626;
  margin-bottom: 1rem;
}

.customer-list__retry-btn {
  padding: 0.5rem 1rem;
  background: #2563eb;
  color: #fff;
  border: none;
  border-radius: 0.375rem;
  cursor: pointer;
  font-size: 0.875rem;
}

.customer-list__retry-btn:hover {
  background: #1d4ed8;
}
</style>
