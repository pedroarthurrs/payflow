import { describe, it, expect, beforeEach, vi } from 'vitest'

// Mock useToast
const successMock = vi.fn()
const errorMock = vi.fn()
const warningMock = vi.fn()
vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    success: successMock,
    error: errorMock,
    warning: warningMock,
  }),
}))

import { useErrorHandler } from './useErrorHandler'
import type { ApiError } from '@/types'

describe('useErrorHandler', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('handles 403 with error toast', () => {
    const { handleError } = useErrorHandler()
    const err: ApiError = { message: 'Acesso negado', status: 403 }

    const handled = handleError(err)

    expect(handled).toBe(true)
    expect(errorMock).toHaveBeenCalledWith('Acesso negado')
  })

  it('uses default message for 403 without message', () => {
    const { handleError } = useErrorHandler()
    const err: ApiError = { message: '', status: 403 }

    handleError(err)

    expect(errorMock).toHaveBeenCalledWith('Permissão negada')
  })

  it('handles 404 with onNotFound callback when provided', () => {
    const onNotFound = vi.fn()
    const { handleError } = useErrorHandler({ onNotFound })
    const err: ApiError = { message: 'Não encontrado', status: 404 }

    const handled = handleError(err)

    expect(handled).toBe(true)
    expect(onNotFound).toHaveBeenCalledWith(err)
    expect(errorMock).not.toHaveBeenCalled()
  })

  it('handles 404 with error toast when no onNotFound callback', () => {
    const { handleError } = useErrorHandler()
    const err: ApiError = { message: 'Recurso não existe', status: 404 }

    const handled = handleError(err)

    expect(handled).toBe(true)
    expect(errorMock).toHaveBeenCalledWith('Recurso não existe')
  })

  it('handles 429 with warning toast', () => {
    const { handleError } = useErrorHandler()
    const err: ApiError = { message: 'too many requests', status: 429 }

    const handled = handleError(err)

    expect(handled).toBe(true)
    expect(warningMock).toHaveBeenCalledWith(
      'Limite de requisições excedido. Aguarde antes de tentar novamente.',
    )
  })

  it('does not handle 400 errors', () => {
    const { handleError } = useErrorHandler()
    const err: ApiError = { message: 'Validation failed', status: 400 }

    const handled = handleError(err)

    expect(handled).toBe(false)
    expect(errorMock).not.toHaveBeenCalled()
    expect(warningMock).not.toHaveBeenCalled()
  })

  it('does not handle 409 errors', () => {
    const { handleError } = useErrorHandler()
    const err: ApiError = { message: 'Conflict', status: 409 }

    const handled = handleError(err)

    expect(handled).toBe(false)
  })

  it('does not handle 500 errors', () => {
    const { handleError } = useErrorHandler()
    const err: ApiError = { message: 'Internal server error', status: 500 }

    const handled = handleError(err)

    expect(handled).toBe(false)
  })

  it('does not handle network errors (status 0)', () => {
    const { handleError } = useErrorHandler()
    const err: ApiError = { message: 'Erro de conexão', status: 0 }

    const handled = handleError(err)

    expect(handled).toBe(false)
  })
})
