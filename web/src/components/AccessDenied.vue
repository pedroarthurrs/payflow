<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()
</script>

<template>
  <div class="access-denied">
    <div class="access-denied__content">
      <span class="access-denied__code">403</span>
      <h1 class="access-denied__title">Acesso Negado</h1>
      <p class="access-denied__message">Você não possui permissão para acessar esta página.</p>
      <button class="access-denied__btn" @click="router.push('/dashboard')">
        Voltar ao Dashboard
      </button>
    </div>
  </div>
</template>

<style scoped>
.access-denied {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 60vh;
}

.access-denied__content {
  text-align: center;
}

.access-denied__code {
  font-size: 4rem;
  font-weight: 700;
  color: #dc2626;
}

.access-denied__title {
  font-size: 1.5rem;
  font-weight: 600;
  color: #111827;
  margin: 0.5rem 0;
}

.access-denied__message {
  color: #6b7280;
  margin-bottom: 1.5rem;
}

.access-denied__btn {
  padding: 0.5rem 1.25rem;
  background-color: #2563eb;
  color: #fff;
  border: none;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  cursor: pointer;
  transition: background-color 0.15s;
}

.access-denied__btn:hover {
  background-color: #1d4ed8;
}
</style>
