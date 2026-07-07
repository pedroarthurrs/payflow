<script setup lang="ts">
import { Button } from '@/core/components/ui/button'
import { useAuth } from '@/core/composables/useAuth'
import type { ApiError } from '@/core/types'
import { ref } from 'vue'

const { pendingOrganizations, selectOrganization } = useAuth()

const loading = ref(false)
const errorMessage = ref('')

async function handleSelect(orgId: string) {
  errorMessage.value = ''
  loading.value = true
  try {
    await selectOrganization(orgId)
  } catch (err) {
    const apiError = err as ApiError
    if (apiError.status === 401 || apiError.status === 403) {
      errorMessage.value = 'Sessão expirada. Faça login novamente.'
    } else {
      errorMessage.value = apiError.message || 'Erro ao selecionar organização. Tente novamente.'
    }
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="flex min-h-svh items-center justify-center bg-background px-4">
    <div class="w-full max-w-sm rounded-xl border bg-card p-8 shadow-sm">
      <h1 class="mb-2 text-center text-3xl font-bold text-foreground">payflow</h1>
      <p class="mb-6 text-center text-sm text-muted-foreground">Selecione a organização para continuar</p>

      <div class="flex flex-col gap-3">
        <Button
          v-for="org in pendingOrganizations"
          :key="org.organization_id"
          variant="outline"
          class="w-full justify-between"
          :disabled="loading"
          @click="handleSelect(org.organization_id)"
        >
          <span>{{ org.name }}</span>
          <span class="text-xs text-muted-foreground capitalize">{{ org.role }}</span>
        </Button>
      </div>

      <p v-if="errorMessage" class="mt-4 text-center text-sm text-destructive">{{ errorMessage }}</p>
    </div>
  </div>
</template>
