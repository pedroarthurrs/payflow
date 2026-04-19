<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/services/http'
import type { MeResponse, Customer, User, ApiError, PagedResponse } from '@/types'
import LoadingSpinner from '@/components/LoadingSpinner.vue'
import { useErrorHandler } from '@/composables/useErrorHandler'

const loading = ref(false)
const error = ref<string | null>(null)
const organizationId = ref<string | null>(null)
const customerCount = ref(0)
const userCount = ref(0)

const { handleError } = useErrorHandler()

async function fetchData() {
  loading.value = true
  error.value = null

  try {
    const [me, customers, members] = await Promise.all([
      api.get<MeResponse>('/v1/me'),
      api.get<PagedResponse<Customer>>('/v1/customers'),
      api.get<PagedResponse<User>>('/v1/members'),
    ])

    organizationId.value = me.organization_id
    customerCount.value = customers.data.length
    userCount.value = members.data.length
  } catch (e: any) {
    const apiErr = e as ApiError
    const handled = handleError(apiErr, fetchData)
    if (!handled) {
      error.value = apiErr?.message || 'Erro ao carregar dados do dashboard'
    }
  } finally {
    loading.value = false
  }
}

onMounted(fetchData)
</script>

<template>
  <div class="dashboard">
    <h1 class="dashboard__title">Dashboard</h1>

    <div v-if="loading" class="dashboard__loading">
      <LoadingSpinner size="lg" />
    </div>

    <div v-else-if="error" class="dashboard__error">
      <p class="dashboard__error-message">{{ error }}</p>
      <button class="dashboard__retry-btn" @click="fetchData">Tentar novamente</button>
    </div>

    <div v-else class="dashboard__grid">
      <div class="dashboard__card">
        <h2 class="dashboard__card-title">Organização</h2>
        <p class="dashboard__card-value">{{ organizationId }}</p>
      </div>

      <div class="dashboard__card">
        <h2 class="dashboard__card-title">Clientes</h2>
        <p class="dashboard__card-value">{{ customerCount }}</p>
      </div>

      <div class="dashboard__card">
        <h2 class="dashboard__card-title">Membros</h2>
        <p class="dashboard__card-value">{{ userCount }}</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dashboard {
  padding: 1.5rem;
}

.dashboard__title {
  font-size: 1.5rem;
  font-weight: 600;
  margin-bottom: 1.5rem;
  color: var(--foreground);
}

.dashboard__loading {
  display: flex;
  justify-content: center;
  padding: 3rem 0;
}

.dashboard__error {
  text-align: center;
  padding: 2rem;
  background: var(--card);
  border: 1px solid var(--destructive);
  border-radius: 0.5rem;
}

.dashboard__error-message {
  color: var(--destructive);
  margin-bottom: 1rem;
}

.dashboard__retry-btn {
  padding: 0.5rem 1rem;
  background: var(--primary);
  color: var(--primary-foreground);
  border: none;
  border-radius: 0.375rem;
  cursor: pointer;
  font-size: 0.875rem;
}

.dashboard__retry-btn:hover {
  filter: brightness(1.08);
}

.dashboard__grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 1.5rem;
}

.dashboard__card {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: 0.5rem;
  padding: 1.5rem;
  box-shadow: 0 1px 3px rgb(0 0 0 / 0.2);
}

.dashboard__card-title {
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--muted-foreground);
  margin-bottom: 0.5rem;
}

.dashboard__card-value {
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--foreground);
  word-break: break-all;
}
</style>
