<script setup lang="ts">
import { ref } from 'vue'
import FormField from '@/components/FormField.vue'
import { Button } from '@/components/ui/button'
import { useAuth } from '@/composables/useAuth'
import type { ApiError } from '@/types'

const { login } = useAuth()

const email = ref('')
const password = ref('')
const loading = ref(false)
const errorMessage = ref('')

async function handleSubmit() {
  errorMessage.value = ''
  loading.value = true
  try {
    await login(email.value, password.value)
  } catch (err) {
    const apiError = err as ApiError
    if (apiError.status === 401) {
      errorMessage.value = apiError.message
    } else if (apiError.status === 429) {
      errorMessage.value = 'Limite de tentativas excedido. Aguarde antes de tentar novamente.'
    } else {
      errorMessage.value = apiError.message || 'Erro ao realizar login. Tente novamente.'
    }
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="flex min-h-svh items-center justify-center bg-background px-4">
    <div class="w-full max-w-sm rounded-xl border bg-card p-8 shadow-sm">
      <h1 class="mb-6 text-center text-3xl font-bold text-foreground">i9pay</h1>
      <form class="flex flex-col" @submit.prevent="handleSubmit">
        <FormField
          label="Email"
          v-model="email"
          type="email"
          :required="true"
          placeholder="seu@email.com"
        />
        <FormField
          label="Senha"
          v-model="password"
          type="password"
          :required="true"
          placeholder="Sua senha"
        />
        <p v-if="errorMessage" class="mb-3 text-center text-sm text-destructive">{{ errorMessage }}</p>
        <Button
          type="submit"
          class="w-full"
          :disabled="loading"
        >
          {{ loading ? 'Entrando...' : 'Entrar' }}
        </Button>
      </form>
    </div>
  </div>
</template>
