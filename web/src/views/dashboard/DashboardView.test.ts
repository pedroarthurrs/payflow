import { flushPromises } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mountWithQuery } from '@/core/utils/wrappers'
import DashboardView from './DashboardView.vue'

// The view fetches through the `api` named export (which unwraps the axios
// response's `.data` for the caller), not the default `http` instance.
vi.mock('@/core/api/http', () => ({
  api: { get: vi.fn() },
}))

async function getApi() {
  return (await import('@/core/api/http')).api
}

describe('DashboardView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  async function mockSuccess(customers = 3, users = 2) {
    const api = await getApi()
    vi.mocked(api.get)
      .mockResolvedValueOnce({ data: Array.from({ length: customers }, (_, i) => ({ id: `c-${i}` })) })
      .mockResolvedValueOnce({ data: Array.from({ length: users }, (_, i) => ({ id: `u-${i}` })) })
  }

  async function mockError(message = 'Erro de conexão com o servidor') {
    const api = await getApi()
    vi.mocked(api.get).mockRejectedValue({ message })
  }

  function mountView() {
    return mountWithQuery(DashboardView)
  }

  it('shows loading spinner while fetching', async () => {
    const api = await getApi()
    let resolve: (v: any) => void
    vi.mocked(api.get).mockReturnValue(new Promise((r) => { resolve = r }) as any)

    const wrapper = mountView()
    await wrapper.vm.$nextTick()

    expect(wrapper.findComponent({ name: 'LoadingSpinner' }).exists()).toBe(true)

    resolve!({ data: [] })
    await flushPromises()
  })

  it('renders customer count and user count cards on success', async () => {
    await mockSuccess(5, 3)

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('Clientes')
    expect(wrapper.text()).toContain('5')
    expect(wrapper.text()).toContain('Membros')
    expect(wrapper.text()).toContain('3')
    expect(wrapper.findComponent({ name: 'LoadingSpinner' }).exists()).toBe(false)
  })

  it('shows error message and retry button on failure', async () => {
    await mockError('Falha na rede')

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('Falha na rede')
    const retryButton = wrapper.findAll('button').find((b) => b.text() === 'Tentar novamente')
    expect(retryButton).toBeTruthy()
  })

  it('re-fetches data when retry button is clicked', async () => {
    await mockError('Erro')

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('Erro')

    // Now mock success for the retry.
    await mockSuccess(1, 1)
    const retryButton = wrapper.findAll('button').find((b) => b.text() === 'Tentar novamente')
    await retryButton!.trigger('click')
    await flushPromises()

    expect(wrapper.text()).not.toContain('Erro')
    expect(wrapper.text()).toContain('Clientes')
    expect(wrapper.text()).toContain('Membros')
  })

  it('fetches from the correct endpoints', async () => {
    await mockSuccess()

    mountView()
    await flushPromises()

    const api = await getApi()
    expect(api.get).toHaveBeenCalledWith('/v1/customers')
    expect(api.get).toHaveBeenCalledWith('/v1/members')
  })
})
