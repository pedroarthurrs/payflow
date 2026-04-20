import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

// Mock http
vi.mock('@/api/http', () => ({
  default: { get: vi.fn() },
}))

import DashboardView from './DashboardView.vue'

async function getHttp() {
  return (await import('@/core/api/http')).default
}

describe('DashboardView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  async function mockSuccess(orgId = 'org-123', customers = 3, users = 2) {
    const http = await getHttp()
    vi.mocked(http.get)
      .mockResolvedValueOnce({ data: { organization_id: orgId } })
      .mockResolvedValueOnce({ data: Array.from({ length: customers }, (_, i) => ({ id: `c-${i}` })) })
      .mockResolvedValueOnce({ data: Array.from({ length: users }, (_, i) => ({ id: `u-${i}` })) })
  }

  async function mockError(message = 'Erro de conexão com o servidor') {
    const http = await getHttp()
    vi.mocked(http.get).mockRejectedValue({ message })
  }

  it('shows loading spinner while fetching', async () => {
    const http = await getHttp()
    let resolve: (v: any) => void
    vi.mocked(http.get).mockReturnValue(new Promise((r) => { resolve = r }) as any)

    const wrapper = mount(DashboardView)
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.dashboard__loading').exists()).toBe(true)
    expect(wrapper.findComponent({ name: 'LoadingSpinner' }).exists()).toBe(true)

    resolve!({ data: { organization_id: 'org-1' } })
  })

  it('renders organization, customer count and user count cards on success', async () => {
    await mockSuccess('org-abc', 5, 3)

    const wrapper = mount(DashboardView)
    await flushPromises()

    const cards = wrapper.findAll('.dashboard__card')
    expect(cards).toHaveLength(3)

    expect(cards[0].find('.dashboard__card-title').text()).toBe('Organização')
    expect(cards[0].find('.dashboard__card-value').text()).toBe('org-abc')

    expect(cards[1].find('.dashboard__card-title').text()).toBe('Clientes')
    expect(cards[1].find('.dashboard__card-value').text()).toBe('5')

    expect(cards[2].find('.dashboard__card-title').text()).toBe('Usuários')
    expect(cards[2].find('.dashboard__card-value').text()).toBe('3')
  })

  it('shows error message and retry button on failure', async () => {
    await mockError('Falha na rede')

    const wrapper = mount(DashboardView)
    await flushPromises()

    expect(wrapper.find('.dashboard__error').exists()).toBe(true)
    expect(wrapper.find('.dashboard__error-message').text()).toBe('Falha na rede')
    expect(wrapper.find('.dashboard__retry-btn').text()).toBe('Tentar novamente')
  })

  it('re-fetches data when retry button is clicked', async () => {
    await mockError('Erro')

    const wrapper = mount(DashboardView)
    await flushPromises()

    expect(wrapper.find('.dashboard__error').exists()).toBe(true)

    // Now mock success for retry
    await mockSuccess('org-retry', 1, 1)
    await wrapper.find('.dashboard__retry-btn').trigger('click')
    await flushPromises()

    expect(wrapper.find('.dashboard__error').exists()).toBe(false)
    const cards = wrapper.findAll('.dashboard__card')
    expect(cards).toHaveLength(3)
    expect(cards[0].find('.dashboard__card-value').text()).toBe('org-retry')
  })

  it('fetches from correct endpoints', async () => {
    await mockSuccess()

    mount(DashboardView)
    await flushPromises()

    const http = await getHttp()
    expect(http.get).toHaveBeenCalledWith('/v1/me')
    expect(http.get).toHaveBeenCalledWith('/v1/customers')
    expect(http.get).toHaveBeenCalledWith('/v1/users')
  })
})
