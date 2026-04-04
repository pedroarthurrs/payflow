<script setup lang="ts">
import { useRouter } from 'vue-router'
import { Button } from '@/components/ui/button'

const router = useRouter()
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-background">
    <div class="text-center">
      <span class="block text-8xl font-bold text-muted-foreground/30">404</span>
      <h1 class="mt-2 text-2xl font-semibold text-foreground">Página não encontrada</h1>
      <p class="mt-1 mb-6 text-muted-foreground">
        O endereço que você acessou não existe ou foi removido.
      </p>
      <div class="flex justify-center gap-3">
        <Button variant="outline" @click="router.back()">Voltar</Button>
        <Button @click="router.push('/dashboard')">Ir ao Dashboard</Button>
      </div>
    </div>
  </div>
</template>
