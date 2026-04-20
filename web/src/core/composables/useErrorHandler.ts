import { useToast } from '@/core/composables/useToast'
import type { ApiError } from '@/core/types'

export interface ErrorHandlerOptions {
  /** Called when a 404 error occurs. If not provided, a toast is shown instead. */
  onNotFound?: (err: ApiError) => void
}

/**
 * Shared error handler for HTTP errors across views.
 * Handles 403, 404, 429 errors with appropriate toasts/callbacks.
 * 400, 409, 500/network errors are NOT handled here — views handle them inline.
 */
export function useErrorHandler(options: ErrorHandlerOptions = {}) {
  const toast = useToast()

  /**
   * Handle an API error with consistent behavior per status code.
   * Returns true if the error was handled (caller doesn't need to show inline error).
   */
  function handleError(err: ApiError, _retryFn?: () => void): boolean {
    switch (err.status) {
      case 403:
        toast.error(err.message || 'Permissão negada')
        return true

      case 404:
        if (options.onNotFound) {
          options.onNotFound(err)
        } else {
          toast.error(err.message || 'Recurso não encontrado')
        }
        return true

      case 429:
        toast.warning('Limite de requisições excedido. Aguarde antes de tentar novamente.')
        return true

      default:
        return false
    }
  }

  return { handleError }
}
