<script setup lang="ts">
withDefaults(defineProps<{
  size?: 'sm' | 'md' | 'lg'
}>(), {
  size: 'md',
})
</script>

<template>
  <div class="spinner-wrapper" role="status">
    <div class="spinner" :class="`spinner--${size}`" />
    <span class="sr-only">Carregando...</span>
  </div>
</template>

<style scoped>
.spinner-wrapper {
  display: flex;
  justify-content: center;
  align-items: center;
}

.spinner {
  border-radius: 50%;
  border: 3px solid #e5e7eb;
  border-top-color: #2563eb;
  animation: spin 0.6s linear infinite;
}

.spinner--sm {
  width: 1rem;
  height: 1rem;
  border-width: 2px;
}

.spinner--md {
  width: 2rem;
  height: 2rem;
  border-width: 3px;
}

.spinner--lg {
  width: 3rem;
  height: 3rem;
  border-width: 4px;
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
