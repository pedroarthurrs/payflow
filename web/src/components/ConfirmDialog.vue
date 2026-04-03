<script setup lang="ts">
import { onMounted, onUnmounted, watch } from 'vue'

const props = withDefaults(defineProps<{
  visible: boolean
  title: string
  message: string
  confirmLabel?: string
  variant?: 'danger' | 'warning'
}>(), {
  confirmLabel: 'Confirmar',
  variant: 'danger',
})

const emit = defineEmits<{
  confirm: []
  cancel: []
}>()

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    emit('cancel')
  }
}

function onBackdropClick(e: MouseEvent) {
  if ((e.target as HTMLElement).classList.contains('confirm-dialog__overlay')) {
    emit('cancel')
  }
}

watch(() => props.visible, (val) => {
  if (val) {
    document.addEventListener('keydown', onKeydown)
  } else {
    document.removeEventListener('keydown', onKeydown)
  }
})

onMounted(() => {
  if (props.visible) {
    document.addEventListener('keydown', onKeydown)
  }
})

onUnmounted(() => {
  document.removeEventListener('keydown', onKeydown)
})
</script>

<template>
  <div v-if="visible" class="confirm-dialog__overlay" @click="onBackdropClick">
    <div class="confirm-dialog" role="dialog" aria-modal="true">
      <h3 class="confirm-dialog__title">{{ title }}</h3>
      <p class="confirm-dialog__message">{{ message }}</p>
      <slot />
      <div class="confirm-dialog__actions">
        <button class="confirm-dialog__btn confirm-dialog__btn--cancel" @click="emit('cancel')">
          Cancelar
        </button>
        <button
          class="confirm-dialog__btn"
          :class="`confirm-dialog__btn--${variant}`"
          @click="emit('confirm')"
        >
          {{ confirmLabel }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.confirm-dialog__overlay {
  position: fixed;
  inset: 0;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9000;
}

.confirm-dialog {
  background-color: #fff;
  border-radius: 0.5rem;
  padding: 1.5rem;
  max-width: 28rem;
  width: 90%;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.2);
}

.confirm-dialog__title {
  margin: 0 0 0.5rem;
  font-size: 1.125rem;
  font-weight: 600;
  color: #111827;
}

.confirm-dialog__message {
  margin: 0 0 1rem;
  font-size: 0.875rem;
  color: #4b5563;
  line-height: 1.5;
}

.confirm-dialog__actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 1rem;
}

.confirm-dialog__btn {
  padding: 0.5rem 1rem;
  border: 1px solid transparent;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 0.15s;
}

.confirm-dialog__btn--cancel {
  background-color: #f3f4f6;
  color: #374151;
  border-color: #d1d5db;
}

.confirm-dialog__btn--cancel:hover {
  background-color: #e5e7eb;
}

.confirm-dialog__btn--danger {
  background-color: #dc2626;
  color: #fff;
}

.confirm-dialog__btn--danger:hover {
  background-color: #b91c1c;
}

.confirm-dialog__btn--warning {
  background-color: #d97706;
  color: #fff;
}

.confirm-dialog__btn--warning:hover {
  background-color: #b45309;
}
</style>
