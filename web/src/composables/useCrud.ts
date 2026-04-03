import { ref, type Ref } from 'vue'
import http from '@/services/http'
import type { ApiError } from '@/types'

export function useCrud<T>(basePath: string) {
  const loading: Ref<boolean> = ref(false)
  const error: Ref<ApiError | null> = ref(null)

  async function list(): Promise<T[]> {
    loading.value = true
    error.value = null
    try {
      const { data } = await http.get<T[]>(basePath)
      return data
    } catch (e) {
      error.value = e as ApiError
      throw e
    } finally {
      loading.value = false
    }
  }

  async function getById(id: string): Promise<T> {
    loading.value = true
    error.value = null
    try {
      const { data } = await http.get<T>(`${basePath}/${id}`)
      return data
    } catch (e) {
      error.value = e as ApiError
      throw e
    } finally {
      loading.value = false
    }
  }

  async function create(payload: Partial<T>): Promise<T> {
    loading.value = true
    error.value = null
    try {
      const { data } = await http.post<T>(basePath, payload)
      return data
    } catch (e) {
      error.value = e as ApiError
      throw e
    } finally {
      loading.value = false
    }
  }

  async function update(id: string, payload: Partial<T>): Promise<T> {
    loading.value = true
    error.value = null
    try {
      const { data } = await http.put<T>(`${basePath}/${id}`, payload)
      return data
    } catch (e) {
      error.value = e as ApiError
      throw e
    } finally {
      loading.value = false
    }
  }

  async function remove(id: string, hard?: boolean): Promise<void> {
    loading.value = true
    error.value = null
    try {
      const url = hard ? `${basePath}/${id}?hard=true` : `${basePath}/${id}`
      await http.delete(url)
    } catch (e) {
      error.value = e as ApiError
      throw e
    } finally {
      loading.value = false
    }
  }

  return {
    list,
    getById,
    create,
    update,
    remove,
    loading,
    error,
  }
}
