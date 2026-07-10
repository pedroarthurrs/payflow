import type { AxiosError, AxiosResponse, InternalAxiosRequestConfig } from 'axios'
import { AxiosHeaders } from 'axios'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import http, { api, normalizeError, withIdempotencyKey } from './http'

function makeAxiosError(status: number, data: unknown, message = 'Request failed'): AxiosError {
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

  it('truncates overly long messages to 500 chars', () => {
    const longMessage = 'x'.repeat(600)
    const err = makeAxiosError(500, { error: longMessage })
    const result = normalizeError(err)
    expect(result.message).toHaveLength(500)
  })
})

describe('withIdempotencyKey', () => {
  it('adds an X-Idempotency-Key header when missing', () => {
    const config = { headers: new AxiosHeaders() } as InternalAxiosRequestConfig
    const result = withIdempotencyKey(config)
    expect(result.headers.has('X-Idempotency-Key')).toBe(true)
  })

  it('does not overwrite an existing X-Idempotency-Key header', () => {
    const headers = new AxiosHeaders()
    headers.set('X-Idempotency-Key', 'fixed-key')
    const config = { headers } as InternalAxiosRequestConfig
    const result = withIdempotencyKey(config)
    expect(result.headers.get('X-Idempotency-Key')).toBe('fixed-key')
  })
})

describe('http request interceptor', () => {
  function getRequestHandler() {
    const handler = http.interceptors.request.handlers?.[0]
    return handler!.fulfilled! as (
      config: InternalAxiosRequestConfig,
    ) => InternalAxiosRequestConfig | Promise<InternalAxiosRequestConfig>
  }

  afterEach(() => {
    document.cookie = 'csrf_token=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/'
  })

  it('injects an X-Request-ID header when missing', async () => {
    const fulfilled = getRequestHandler()
    const config = await fulfilled({
      method: 'get',
      headers: new AxiosHeaders(),
    } as InternalAxiosRequestConfig)

    expect(config.headers.has('X-Request-ID')).toBe(true)
  })

  it('preserves an existing X-Request-ID header', async () => {
    const fulfilled = getRequestHandler()
    const headers = new AxiosHeaders()
    headers.set('X-Request-ID', 'existing-id')
    const config = await fulfilled({ method: 'get', headers } as InternalAxiosRequestConfig)

    expect(config.headers.get('X-Request-ID')).toBe('existing-id')
  })

  it('echoes the CSRF cookie for state-changing requests', async () => {
    document.cookie = 'csrf_token=abc123'
    const fulfilled = getRequestHandler()
    const config = await fulfilled({
      method: 'post',
      headers: new AxiosHeaders(),
    } as InternalAxiosRequestConfig)

    expect(config.headers.get('X-CSRF-Token')).toBe('abc123')
  })

  it('does not set X-CSRF-Token for GET requests', async () => {
    document.cookie = 'csrf_token=abc123'
    const fulfilled = getRequestHandler()
    const config = await fulfilled({
      method: 'get',
      headers: new AxiosHeaders(),
    } as InternalAxiosRequestConfig)

    expect(config.headers.has('X-CSRF-Token')).toBe(false)
  })

  it('does not set X-CSRF-Token when no cookie is present', async () => {
    const fulfilled = getRequestHandler()
    const config = await fulfilled({
      method: 'post',
      headers: new AxiosHeaders(),
    } as InternalAxiosRequestConfig)

    expect(config.headers.has('X-CSRF-Token')).toBe(false)
  })
})

describe('http response interceptor', () => {
  function getResponseRejectedHandler() {
    const handler = http.interceptors.response.handlers?.[0]
    return handler!.rejected! as (error: AxiosError) => Promise<unknown>
  }

  let originalLocation: Location

  beforeEach(() => {
    originalLocation = window.location
    // @ts-expect-error -- replacing jsdom's location with a stub to observe redirects
    delete window.location
    window.location = { href: '' } as Location
  })

  afterEach(() => {
    window.location = originalLocation
    vi.restoreAllMocks()
  })

  it('redirects to /maintenance and rejects the raw error on 503', async () => {
    const rejected = getResponseRejectedHandler()
    const err = makeAxiosError(503, { error: 'Service unavailable' })

    await expect(rejected(err)).rejects.toBe(err)
    expect(window.location.href).toBe('/maintenance')
  })

  it('rejects with a normalized error for non-401/503 responses', async () => {
    const rejected = getResponseRejectedHandler()
    const err = makeAxiosError(500, { error: 'Boom' })
    err.config = { url: '/v1/customers' } as any

    await expect(rejected(err)).rejects.toEqual({ message: 'Boom', status: 500 })
  })

  it('does not attempt a silent refresh for 401s on auth endpoints', async () => {
    const postSpy = vi.spyOn(http, 'post')
    const rejected = getResponseRejectedHandler()
    const err = makeAxiosError(401, { error: 'Invalid credentials' })
    err.config = { url: '/v1/auth/login' } as any

    await expect(rejected(err)).rejects.toEqual({ message: 'Invalid credentials', status: 401 })
    expect(postSpy).not.toHaveBeenCalled()
    expect(window.location.href).toBe('')
  })

  it('redirects to /login when the silent refresh fails on a 401', async () => {
    const postSpy = vi.spyOn(http, 'post').mockRejectedValueOnce(new Error('refresh failed'))
    const rejected = getResponseRejectedHandler()
    const err = makeAxiosError(401, { error: 'Unauthorized' })
    err.config = { url: '/v1/customers' } as any

    await expect(rejected(err)).rejects.toEqual({ message: 'Unauthorized', status: 401 })
    expect(postSpy).toHaveBeenCalledWith('/v1/auth/refresh', {})
    expect(window.location.href).toBe('/login')
  })
})

describe('api helpers', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('api.get unwraps the response data', async () => {
    vi.spyOn(http, 'get').mockResolvedValueOnce({ data: { id: '1' } } as AxiosResponse)

    const result = await api.get<{ id: string }>('/v1/customers/1')

    expect(http.get).toHaveBeenCalledWith('/v1/customers/1', undefined)
    expect(result).toEqual({ id: '1' })
  })

  it('api.post unwraps the response data', async () => {
    vi.spyOn(http, 'post').mockResolvedValueOnce({ data: { id: '2' } } as AxiosResponse)

    const result = await api.post<{ id: string }>('/v1/customers', { name: 'Jane' })

    expect(http.post).toHaveBeenCalledWith('/v1/customers', { name: 'Jane' }, undefined)
    expect(result).toEqual({ id: '2' })
  })

  it('api.put unwraps the response data', async () => {
    vi.spyOn(http, 'put').mockResolvedValueOnce({ data: { updated: true } } as AxiosResponse)

    const result = await api.put<{ updated: boolean }>('/v1/customers/1', { name: 'Jane' })

    expect(http.put).toHaveBeenCalledWith('/v1/customers/1', { name: 'Jane' }, undefined)
    expect(result).toEqual({ updated: true })
  })

  it('api.patch unwraps the response data', async () => {
    vi.spyOn(http, 'patch').mockResolvedValueOnce({ data: { patched: true } } as AxiosResponse)

    const result = await api.patch<{ patched: boolean }>('/v1/customers/1', { name: 'Jane' })

    expect(http.patch).toHaveBeenCalledWith('/v1/customers/1', { name: 'Jane' }, undefined)
    expect(result).toEqual({ patched: true })
  })

  it('api.delete unwraps the response data', async () => {
    vi.spyOn(http, 'delete').mockResolvedValueOnce({ data: undefined } as AxiosResponse)

    const result = await api.delete('/v1/customers/1')

    expect(http.delete).toHaveBeenCalledWith('/v1/customers/1', undefined)
    expect(result).toBeUndefined()
  })
})
