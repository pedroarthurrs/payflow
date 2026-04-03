import { ref, type Ref } from 'vue'

export interface Toast {
  id: string
  type: 'success' | 'error' | 'warning' | 'info'
  message: string
  duration: number
}

// Singleton: module-level ref shared across all components
const toasts: Ref<Toast[]> = ref([])

let counter = 0

function addToast(type: Toast['type'], message: string, duration = 3000): void {
  const id = `toast-${++counter}`
  const toast: Toast = { id, type, message, duration }
  toasts.value.push(toast)

  setTimeout(() => {
    removeToast(id)
  }, duration)
}

function removeToast(id: string): void {
  toasts.value = toasts.value.filter((t) => t.id !== id)
}

export function useToast() {
  function success(message: string): void {
    addToast('success', message)
  }

  function error(message: string): void {
    addToast('error', message)
  }

  function warning(message: string): void {
    addToast('warning', message)
  }

  return {
    toasts,
    success,
    error,
    warning,
    removeToast,
  }
}
