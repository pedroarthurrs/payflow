import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'

// Mock vue-router
const pushMock = vi.fn()
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: pushMock }),
}))

// Mock http (used by useCrud internally)
vi.mock('@/services/http', () => ({
  default: { get: vi.fn() },
}))

import CustomerListView from './CustomerListView.vue'

async function getHttp() {
  return (await import('@/services/http')).default
}

function makeCustomer(overrides: Record<string, unknown> = {}) {
  return {
    id: 'cust-1',
    name: 'João Silva',
    email: 'joao@test.com',
    document: '12345678901',
    document_type: 'CPF',
    status: 'active',
    created_at: '2024-01-15T10:30:00Z',
    updated_at: '2024-01-15T10:30:00Z',
    ...overrides,
  }
}

describe('CustomerListView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders page title and "Novo Cliente" button', async () => {
    const http = await getHttp()
    vi.mocked(http.get).mockResolvedValueOnce({ data: [] })

    const wrapper = mount(CustomerListView)
    await flushPromises()

    expect(wrapper.find('.customer-list__title').text()).toBe('Clientes')
    expect(wrapper.find('.customer-list__new-btn').text()).toBe('Novo Cliente')
  })

  it('shows loading state via DataTable while fetching', async () => {
    const http = await getHttp()
    let resolve: (v: any) => void
    vi.mocked(http.get).mockReturnValue(new Promise((r) => { resolve = r }) as any)

    const wrapper = mount(CustomerListView)
    await wrapper.vm.$nextTick()

    const dataTable = wrapper.findComponent({ name: 'DataTable' })
    expect(dataTable.exists()).toBe(true)
    expect(dataTable.props('loading')).toBe(true)

    resolve!({ data: [] })
  })

  it('fetches customers from /v1/customers and renders them in DataTable', async () => {
    const http = await getHttp()
    const customers = [
      makeCustomer({ id: 'c-1', name: 'Alice' }),
      makeCustomer({ id: 'c-2', name: 'Bob' }),
    ]
    vi.mocked(http.get).mockResolvedValueOnce({ data: customers })

    const wrapper = mount(CustomerListView)
    await flushPromises()

    expect(http.get).toHaveBeenCalledWith('/v1/customers')

    const dataTable = wrapper.findComponent({ name: 'DataTable' })
    expect(dataTable.props('data')).toHaveLength(2)
    expect(dataTable.props('loading')).toBe(false)
  })

  it('defines correct columns: nome, email, documento, status, data de criação', async () => {
    const http = await getHttp()
    vi.mocked(http.get).mockResolvedValueOnce({ data: [] })

    const wrapper = mount(CustomerListView)
    await flushPromises()

    const dataTable = wrapper.findComponent({ name: 'DataTable' })
    const columns = dataTable.props('columns')

    expect(columns).toHaveLength(5)
    expect(columns[0]).toMatchObject({ key: 'name', label: 'Nome' })
    expect(columns[1]).toMatchObject({ key: 'email', label: 'Email' })
    expect(columns[2]).toMatchObject({ key: 'document', label: 'Documento' })
    expect(columns[3]).toMatchObject({ key: 'status', label: 'Status' })
    expect(columns[4]).toMatchObject({ key: 'created_at', label: 'Data de Criação' })
  })

  it('formats created_at column as localized date', async () => {
    const http = await getHttp()
    vi.mocked(http.get).mockResolvedValueOnce({ data: [] })

    const wrapper = mount(CustomerListView)
    await flushPromises()

    const dataTable = wrapper.findComponent({ name: 'DataTable' })
    const columns = dataTable.props('columns')
    const createdAtCol = columns.find((c: any) => c.key === 'created_at')

    expect(createdAtCol.format).toBeDefined()
    const formatted = createdAtCol.format('2024-01-15T10:30:00Z')
    // Should produce a localized date string (pt-BR: 15/01/2024)
    expect(formatted).toBe(new Date('2024-01-15T10:30:00Z').toLocaleDateString('pt-BR'))
  })

  it('formats empty created_at as empty string', async () => {
    const http = await getHttp()
    vi.mocked(http.get).mockResolvedValueOnce({ data: [] })

    const wrapper = mount(CustomerListView)
    await flushPromises()

    const dataTable = wrapper.findComponent({ name: 'DataTable' })
    const columns = dataTable.props('columns')
    const createdAtCol = columns.find((c: any) => c.key === 'created_at')

    expect(createdAtCol.format('')).toBe('')
    expect(createdAtCol.format(null)).toBe('')
    expect(createdAtCol.format(undefined)).toBe('')
  })

  it('navigates to /customers/new when "Novo Cliente" is clicked', async () => {
    const http = await getHttp()
    vi.mocked(http.get).mockResolvedValueOnce({ data: [] })

    const wrapper = mount(CustomerListView)
    await flushPromises()

    await wrapper.find('.customer-list__new-btn').trigger('click')
    expect(pushMock).toHaveBeenCalledWith('/customers/new')
  })

  it('navigates to /customers/:id on row-click', async () => {
    const http = await getHttp()
    const customer = makeCustomer({ id: 'cust-42' })
    vi.mocked(http.get).mockResolvedValueOnce({ data: [customer] })

    const wrapper = mount(CustomerListView)
    await flushPromises()

    const dataTable = wrapper.findComponent({ name: 'DataTable' })
    dataTable.vm.$emit('row-click', customer)
    await wrapper.vm.$nextTick()

    expect(pushMock).toHaveBeenCalledWith('/customers/cust-42')
  })

  it('shows error message and retry button on fetch failure', async () => {
    const http = await getHttp()
    vi.mocked(http.get).mockRejectedValueOnce({ message: 'Erro de conexão com o servidor' })

    const wrapper = mount(CustomerListView)
    await flushPromises()

    expect(wrapper.find('.customer-list__error').exists()).toBe(true)
    expect(wrapper.find('.customer-list__error-message').text()).toBe('Erro de conexão com o servidor')
    expect(wrapper.find('.customer-list__retry-btn').exists()).toBe(true)
  })

  it('retries fetch when retry button is clicked', async () => {
    const http = await getHttp()
    vi.mocked(http.get).mockRejectedValueOnce({ message: 'Erro' })

    const wrapper = mount(CustomerListView)
    await flushPromises()

    expect(wrapper.find('.customer-list__error').exists()).toBe(true)

    // Mock success for retry
    vi.mocked(http.get).mockResolvedValueOnce({ data: [makeCustomer()] })
    await wrapper.find('.customer-list__retry-btn').trigger('click')
    await flushPromises()

    expect(wrapper.find('.customer-list__error').exists()).toBe(false)
    const dataTable = wrapper.findComponent({ name: 'DataTable' })
    expect(dataTable.props('data')).toHaveLength(1)
  })
})
