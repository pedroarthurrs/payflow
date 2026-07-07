<script setup lang="ts">
import FormField from '@/core/components/FormField.vue'
import PixelBlast from '@/core/components/PixelBlast.vue'
import { Button } from '@/core/components/ui/button'
import { useAuth } from '@/core/composables/useAuth'
import type { ApiError } from '@/core/types'
import { loginSchema } from '@/modules/auth/schemas'
import { toTypedSchema } from '@vee-validate/zod'
import { useForm } from 'vee-validate'
import { ref } from 'vue'

const { login } = useAuth()

const { handleSubmit, errors, defineField } = useForm({
  validationSchema: toTypedSchema(loginSchema),
})

const [email] = defineField('email')
const [password] = defineField('password')

const loading = ref(false)
const errorMessage = ref('')

const onSubmit = handleSubmit(async (values) => {
  errorMessage.value = ''
  loading.value = true
  try {
    await login(values.email, values.password)
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
})
</script>

<template>
  <div class="relative flex min-h-svh items-center justify-center bg-background px-4">
    <!-- PixelBlast Background -->
    <div class="fixed inset-0 z-0">
      <PixelBlast
        variant="square"
        :pixel-size="10"
        color="#bfbebd"
        :pattern-scale="0.5"
        :pattern-density="1"
        :speed="0.25"
        :pixel-size-jitter="2"
        :enable-ripples="false"
        :ripple-intensity-scale="0.8"
        :edge-fade="0"
        transparent
      />
    </div>

    <!-- Login Card -->
    <div class="relative z-10 w-full max-w-sm rounded-xl border backdrop-blur-md p-8 shadow-lg">
      <h1 class="mb-6 text-center text-3xl font-bold text-foreground">payflow</h1>
      <form class="flex flex-col" novalidate @submit.prevent="onSubmit">
        <FormField
          label="Email"
          :modelValue="email ?? ''"
          @update:modelValue="email = $event as string"
          type="email"
          :error="errors.email"
          placeholder="seu@email.com"
        />
        <FormField
          label="Senha"
          :modelValue="password ?? ''"
          @update:modelValue="password = $event as string"
          type="password"
          :error="errors.password"
          :show-password-toggle="true"
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
