<script setup lang="ts">
import { Button } from '@/core/components/ui/button';
import { useRouter } from 'vue-router';

const router = useRouter()
</script>

<template>
  <div class="flex min-h-[60vh] items-center justify-center">
    <div class="text-center">
      <span class="block text-6xl font-bold text-destructive">403</span>
      <h1 class="mt-2 text-2xl font-semibold text-foreground">Acesso Negado</h1>
      <p class="mt-1 mb-6 text-muted-foreground">Você não possui permissão para acessar esta página.</p>
      <Button @click="router.push('/dashboard')">Voltar ao Dashboard</Button>
    </div>
  </div>
</template>
