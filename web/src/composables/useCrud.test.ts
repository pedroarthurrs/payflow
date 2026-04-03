import { describe, it, expect, beforeEach, vi } from 'vitest'
import { useCrud } from './useCrud'

vi.mock('@/services/http', () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}))

interface TestEntity {
  id: string
  name: string
}

const basePath = '/v1/customers'

describe('useCrud', () => {
  let http: any

  beforeEach(async () => {
    vi.clearAllMocks()
    http = (await import('@/services/http')).default
  })

  describe('list', () => {
    it('calls GET on basePath and returns data', async () => {
      const items: TestEntity[] = [{ id: '1', name: 'Alice' }]
      http.get.mockResolvedValueOnce({ data: items })

      const { list, loading, error } = useCrud<TestEntity>(basePath)
      const result = await list()

      expect(http.get).toHaveBeenCalledWith(basePath)
      expect(result).toEqual(items)
      expect(loading.value).toBe(false)
      expect(error.value).toBeNull()
    })

    it('sets error on failure', async () => {
      const apiError = { message: 'Server error', status: 500 }
      http.get.mockRejectedValueOnce(apiError)

      const { list, error, loading } = useCrud<TestEntity>(basePath)
      await expect(list()).rejects.toEqual(apiError)

      expect(error.value).toEqual(apiError)
      expect(loading.value).toBe(false)
    })
  })

  describe('getById', () => {
    it('calls GET on basePath/:id and returns data', async () => {
      const item: TestEntity = { id: '42', name: 'Bob' }
      http.get.mockResolvedValueOnce({ data: item })

      const { getById } = useCrud<TestEntity>(basePath)
      const result = await getById('42')

      expect(http.get).toHaveBeenCalledWith(`${basePath}/42`)
      expect(result).toEqual(item)
    })

    it('sets error on failure', async () => {
      const apiError = { message: 'Not found', status: 404 }
      http.get.mockRejectedValueOnce(apiError)

      const { getById, error } = useCrud<TestEntity>(basePath)
      await expect(getById('999')).rejects.toEqual(apiError)

      expect(error.value).toEqual(apiError)
    })
  })

  describe('create', () => {
    it('calls POST on basePath with payload and returns data', async () => {
      const created: TestEntity = { id: '3', name: 'Carol' }
      http.post.mockResolvedValueOnce({ data: created })

      const { create } = useCrud<TestEntity>(basePath)
      const result = await create({ name: 'Carol' })

      expect(http.post).toHaveBeenCalledWith(basePath, { name: 'Carol' })
      expect(result).toEqual(created)
    })

    it('sets error on validation failure', async () => {
      const apiError = { message: 'Validation failed', status: 400, validationErrors: { name: 'required' } }
      http.post.mockRejectedValueOnce(apiError)

      const { create, error } = useCrud<TestEntity>(basePath)
      await expect(create({})).rejects.toEqual(apiError)

      expect(error.value).toEqual(apiError)
    })
  })

  describe('update', () => {
    it('calls PUT on basePath/:id with payload and returns data', async () => {
      const updated: TestEntity = { id: '1', name: 'Alice Updated' }
      http.put.mockResolvedValueOnce({ data: updated })

      const { update } = useCrud<TestEntity>(basePath)
      const result = await update('1', { name: 'Alice Updated' })

      expect(http.put).toHaveBeenCalledWith(`${basePath}/1`, { name: 'Alice Updated' })
      expect(result).toEqual(updated)
    })

    it('sets error on failure', async () => {
      const apiError = { message: 'Forbidden', status: 403 }
      http.put.mockRejectedValueOnce(apiError)

      const { update, error } = useCrud<TestEntity>(basePath)
      await expect(update('1', { name: 'x' })).rejects.toEqual(apiError)

      expect(error.value).toEqual(apiError)
    })
  })

  describe('remove', () => {
    it('calls DELETE on basePath/:id (soft delete)', async () => {
      http.delete.mockResolvedValueOnce({})

      const { remove } = useCrud<TestEntity>(basePath)
      await remove('5')

      expect(http.delete).toHaveBeenCalledWith(`${basePath}/5`)
    })

    it('calls DELETE with ?hard=true when hard=true', async () => {
      http.delete.mockResolvedValueOnce({})

      const { remove } = useCrud<TestEntity>(basePath)
      await remove('5', true)

      expect(http.delete).toHaveBeenCalledWith(`${basePath}/5?hard=true`)
    })

    it('does not append ?hard=true when hard=false', async () => {
      http.delete.mockResolvedValueOnce({})

      const { remove } = useCrud<TestEntity>(basePath)
      await remove('5', false)

      expect(http.delete).toHaveBeenCalledWith(`${basePath}/5`)
    })

    it('sets error on failure', async () => {
      const apiError = { message: 'Server error', status: 500 }
      http.delete.mockRejectedValueOnce(apiError)

      const { remove, error } = useCrud<TestEntity>(basePath)
      await expect(remove('5')).rejects.toEqual(apiError)

      expect(error.value).toEqual(apiError)
    })
  })

  describe('loading state', () => {
    it('is true during a request and false after', async () => {
      let resolvePromise: (v: any) => void
      http.get.mockReturnValueOnce(new Promise((r) => { resolvePromise = r }))

      const { list, loading } = useCrud<TestEntity>(basePath)
      expect(loading.value).toBe(false)

      const promise = list()
      expect(loading.value).toBe(true)

      resolvePromise!({ data: [] })
      await promise

      expect(loading.value).toBe(false)
    })
  })

  describe('error reset', () => {
    it('clears previous error on new request', async () => {
      const apiError = { message: 'fail', status: 500 }
      http.get.mockRejectedValueOnce(apiError)

      const { list, error } = useCrud<TestEntity>(basePath)
      await expect(list()).rejects.toEqual(apiError)
      expect(error.value).toEqual(apiError)

      http.get.mockResolvedValueOnce({ data: [] })
      await list()
      expect(error.value).toBeNull()
    })
  })
})
