import { describe, it, expect, beforeEach } from 'vitest'
import type { AxiosError, AxiosResponse } from 'axios'
import { normalizeError } from './http'

function makeAxiosError(
  status: number,
  data: unknown,
  message = 'Request failed',
): AxiosError {
  const error = new Error(message) as AxiosError
  error.isAxiosError = true
  error.response = {
    status,
    data,
    headers: {},
    statusText: '',
    config: {} as any,
  } as AxiosResponse
  error.config = {} as any
  error.toJSON = () => ({})
  return error
}

function makeNetworkError(): AxiosError {
  const error = new Error('Network Error') as AxiosError
  error.isAxiosError = true
  error.response = undefined
  error.config = {} as any
  error.toJSON = () => ({})
  return error
}

describe('normalizeError', () => {
  it('extracts message and status from API error response', () => {
    const err = makeAxiosError(403, { error: 'Forbidden' })
    const result = normalizeError(err)
    expect(result).toEqual({ message: 'Forbidden', status: 403 })
  })

  it('maps validation errors for 400 responses', () => {
    const err = makeAxiosError(400, {
      error: 'Validation failed',
      errors: [
        { field: 'email', rejected_value: '', message: 'Email é obrigatório' },
        { field: 'name', rejected_value: '', message: 'Nome é obrigatório' },
      ],
    })
    const result = normalizeError(err)
    expect(result.status).toBe(400)
    expect(result.validationErrors).toEqual({
      email: 'Email é obrigatório',
      name: 'Nome é obrigatório',
    })
  })

  it('does not include validationErrors for non-400 errors', () => {
    const err = makeAxiosError(500, { error: 'Internal server error' })
    const result = normalizeError(err)
    expect(result.status).toBe(500)
    expect(result.validationErrors).toBeUndefined()
  })

  it('does not include validationErrors for 400 without errors array', () => {
    const err = makeAxiosError(400, { error: 'Bad request' })
    const result = normalizeError(err)
    expect(result.status).toBe(400)
    expect(result.validationErrors).toBeUndefined()
  })

  it('returns generic error for network errors', () => {
    const err = makeNetworkError()
    const result = normalizeError(err)
    expect(result.status).toBe(0)
    expect(result.message).toBe('Erro de conexão com o servidor')
  })

  it('falls back to error.message when data.error is missing', () => {
    const err = makeAxiosError(422, {}, 'Unprocessable Entity')
    const result = normalizeError(err)
    expect(result.status).toBe(422)
    expect(result.message).toBe('Unprocessable Entity')
  })
})

describe('http instance', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('injects Authorization header when access_token exists', async () => {
    localStorage.setItem('access_token', 'test-token-123')

    // Import fresh to get the interceptors
    const { default: http } = await import('./http')

    // Use interceptor manager to test request config transformation
    const handler = http.interceptors.request.handlers?.[0]
    const config = await handler!.fulfilled!({
      headers: new (await import('axios')).AxiosHeaders(),
    } as any)

    expect(config.headers.Authorization).toBe('Bearer test-token-123')
  })

  it('does not inject Authorization header when no token', async () => {
    const { default: http } = await import('./http')

    const handler = http.interceptors.request.handlers?.[0]
    const config = await handler!.fulfilled!({
      headers: new (await import('axios')).AxiosHeaders(),
    } as any)

    expect(config.headers.Authorization).toBeUndefined()
  })
})
