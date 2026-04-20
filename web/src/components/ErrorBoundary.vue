<script setup lang="ts">
import { ref, onErrorCaptured } from 'vue'
import { useAuthStore, AuthError } from '@/stores/auth'
import { useRouter } from 'vue-router'

const authStore = useAuthStore()
const router = useRouter()

const authError = ref<AuthError | null>(null)

onErrorCaptured((err) => {
  if (err instanceof AuthError) {
    authError.value = err
    return false // prevent further propagation
  }
  // Let non-auth errors propagate to parent boundaries / global handler
  return true
})

function handleLogout() {
  authError.value = null
  authStore.logout()
  router.push('/login')
}

function handleRetry() {
  authError.value = null
}
</script>

<template>
  <div v-if="authError" class="flex min-h-[200px] flex-col items-center justify-center gap-4 p-6 text-center">
    <h2 class="text-lg font-semibold text-destructive">Erro de Autenticação</h2>
    <p class="max-w-sm text-sm text-muted-foreground">{{ authError.message }}</p>
    <div class="flex gap-2">
      <button
        class="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
        @click="handleRetry"
      >
        Tentar novamente
      </button>
      <button
        class="rounded-md border px-4 py-2 text-sm font-medium hover:bg-accent"
        @click="handleLogout"
      >
        Voltar ao login
      </button>
    </div>
  </div>
  <slot v-else />
</template>
