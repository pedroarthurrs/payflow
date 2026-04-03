<script setup lang="ts">
import { ref } from 'vue'
import FormField from '@/components/FormField.vue'
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
  <div class="login-page">
    <div class="login-card">
      <h1 class="login-brand">i9pay</h1>
      <form class="login-form" @submit.prevent="handleSubmit">
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
        <p v-if="errorMessage" class="login-error">{{ errorMessage }}</p>
        <button
          type="submit"
          class="login-button"
          :disabled="loading"
        >
          {{ loading ? 'Entrando...' : 'Entrar' }}
        </button>
      </form>
    </div>
  </div>
</template>

<style scoped>
.login-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background-color: #f3f4f6;
}

.login-card {
  background: #fff;
  border-radius: 0.5rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1), 0 1px 2px rgba(0, 0, 0, 0.06);
  padding: 2rem;
  width: 100%;
  max-width: 24rem;
}

.login-brand {
  text-align: center;
  font-size: 1.75rem;
  font-weight: 700;
  color: #2563eb;
  margin: 0 0 1.5rem;
}

.login-form {
  display: flex;
  flex-direction: column;
}

.login-error {
  color: #dc2626;
  font-size: 0.875rem;
  margin: 0 0 0.75rem;
  text-align: center;
}

.login-button {
  padding: 0.625rem 1rem;
  background-color: #2563eb;
  color: #fff;
  border: none;
  border-radius: 0.375rem;
  font-size: 0.9375rem;
  font-weight: 600;
  cursor: pointer;
  transition: background-color 0.15s;
}

.login-button:hover:not(:disabled) {
  background-color: #1d4ed8;
}

.login-button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
