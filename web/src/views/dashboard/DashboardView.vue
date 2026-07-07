<script setup lang="ts">
import { api } from '@/core/api/http'
import LoadingSpinner from '@/core/components/LoadingSpinner.vue'
import { Button } from '@/core/components/ui/button'
import { useErrorHandler } from '@/core/composables/useErrorHandler'
import type { ApiError, Customer, PagedResponse, User } from '@/core/types'
import { onMounted, ref } from 'vue'

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
    const [customers, members] = await Promise.all([
      api.get<PagedResponse<Customer>>('/v1/customers'),
      api.get<PagedResponse<User>>('/v1/members'),
    ])

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
  <div class="p-6">
    <h1 class="text-2xl font-semibold text-foreground mb-6">Dashboard</h1>

    <div v-if="loading" class="flex justify-center py-12">
      <LoadingSpinner size="lg" />
    </div>

    <div v-else-if="error" class="text-center p-8 bg-card border border-destructive rounded-lg">
      <p class="text-destructive mb-4">{{ error }}</p>
      <Button @click="fetchData">Tentar novamente</Button>
    </div>

    <div v-else class="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
      <div class="bg-card border border-border rounded-lg p-6 shadow-sm">
        <h2 class="text-sm font-medium text-muted-foreground mb-2">Clientes</h2>
        <p class="text-2xl font-bold text-foreground break-all">{{ customerCount }}</p>
      </div>

      <div class="bg-card border border-border rounded-lg p-6 shadow-sm">
        <h2 class="text-sm font-medium text-muted-foreground mb-2">Membros</h2>
        <p class="text-2xl font-bold text-foreground break-all">{{ userCount }}</p>
      </div>
    </div>
  </div>
</template>
