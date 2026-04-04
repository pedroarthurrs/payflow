<script setup lang="ts">
import { ref, watch } from 'vue'
import FormField from '@/components/FormField.vue'
import { Button } from '@/components/ui/button'
import { useAuth } from '@/composables/useAuth'
import type { ApiError } from '@/types'

const { login } = useAuth()

const email = ref('')
const password = ref('')
const loading = ref(false)
const errorMessage = ref('')
const emailError = ref('')
const passwordError = ref('')
const submitted = ref(false)

const EMAIL_REGEX = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

function validateEmail(value: string): string {
  if (!value.trim()) return 'Email é obrigatório.'
  if (!EMAIL_REGEX.test(value.trim())) return 'Informe um email válido.'
  return ''
}

function validatePassword(value: string): string {
  if (!value) return 'Senha é obrigatória.'
  return ''
}

watch(email, (val) => {
  if (submitted.value) emailError.value = validateEmail(val)
})

watch(password, (val) => {
  if (submitted.value) passwordError.value = validatePassword(val)
})

function validate(): boolean {
  emailError.value = validateEmail(email.value)
  passwordError.value = validatePassword(password.value)
  return !emailError.value && !passwordError.value
}

async function handleSubmit() {
  submitted.value = true
  if (!validate()) return
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
      <form class="flex flex-col" novalidate @submit.prevent="handleSubmit">
        <FormField
          label="Email"
          v-model="email"
          type="email"
          :error="emailError"
          placeholder="seu@email.com"
        />
        <FormField
          label="Senha"
          v-model="password"
          type="password"
          :error="passwordError"
          :show-password-toggle="true"
          placeholder="Sua senha"
        />
        <p v-if="errorMessage" class="login-error mb-3 text-center text-sm text-destructive">{{ errorMessage }}</p>
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
