import { mountWithQuery } from '@/core/utils/wrappers'
import { flushPromises } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

// Mock vue-router
const pushMock = vi.fn()
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: pushMock }),
  useRoute: () => ({ params: {} }),
}))

// Mock the API client — useCustomerList calls `api.get` (unwrapped response).
vi.mock('@/core/api/http', () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
  },
  default: {
    get: vi.fn(),
    post: vi.fn(),
  },
}))

// Mock the RBAC store so the "Novo Cliente" button (gated on customers:write)
// renders. Can.vue and useRBAC both resolve to this same './store' module.
vi.mock('@/core/rbac/store', () => ({
  useRBACStore: () => ({
    can: () => true,
    canAny: () => true,
    canAll: () => true,
    hasRole: () => true,
    loaded: true,
    role: 'owner',
    fetchPermissions: vi.fn().mockResolvedValue(undefined),
    $reset: vi.fn(),
  }),
}))

import CustomerListView from './CustomerListView.vue'

async function getApi() {
  return (await import('@/core/api/http')).api
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
    const api = await getApi()
    vi.mocked(api.get).mockResolvedValueOnce({ data: [] })

    const wrapper = mountWithQuery(CustomerListView)
    await flushPromises()

    expect(wrapper.find('h1').text()).toBe('Clientes')
    expect(wrapper.findAll('button').some((b) => b.text() === 'Novo Cliente')).toBe(true)
  })

  it('shows loading state via DataTable while fetching', async () => {
    const api = await getApi()
    let resolve: (v: any) => void
    vi.mocked(api.get).mockReturnValue(new Promise((r) => { resolve = r }) as any)

    const wrapper = mountWithQuery(CustomerListView)
    await wrapper.vm.$nextTick()

    const dataTable = wrapper.findComponent({ name: 'DataTable' })
    expect(dataTable.exists()).toBe(true)
    expect(dataTable.props('loading')).toBe(true)

    resolve!({ data: [] })
  })

  it('fetches customers from /v1/customers and renders them in DataTable', async () => {
    const api = await getApi()
    const customers = [
      makeCustomer({ id: 'c-1', name: 'Alice' }),
      makeCustomer({ id: 'c-2', name: 'Bob' }),
    ]
    vi.mocked(api.get).mockResolvedValueOnce({ data: customers })

    const wrapper = mountWithQuery(CustomerListView)
    await flushPromises()

    expect(api.get).toHaveBeenCalledWith('/v1/customers', { params: undefined })

    const dataTable = wrapper.findComponent({ name: 'DataTable' })
    expect(dataTable.props('data')).toHaveLength(2)
    expect(dataTable.props('loading')).toBe(false)
  })

  it('defines correct columns: nome, email, telefone, documento, status, data de criação', async () => {
    const api = await getApi()
    vi.mocked(api.get).mockResolvedValueOnce({ data: [] })

    const wrapper = mountWithQuery(CustomerListView)
    await flushPromises()

    const dataTable = wrapper.findComponent({ name: 'DataTable' })
    const columns = dataTable.props('columns')

    expect(columns).toHaveLength(6)
    expect(columns[0]).toMatchObject({ key: 'name', label: 'Nome' })
    expect(columns[1]).toMatchObject({ key: 'email', label: 'Email' })
    expect(columns[2]).toMatchObject({ key: 'phone', label: 'Telefone' })
    expect(columns[3]).toMatchObject({ key: 'document', label: 'Documento' })
    expect(columns[4]).toMatchObject({ key: 'status', label: 'Status' })
    expect(columns[5]).toMatchObject({ key: 'created_at', label: 'Data de Criação' })
  })

  it('formats created_at column as localized date', async () => {
    const api = await getApi()
    vi.mocked(api.get).mockResolvedValueOnce({ data: [] })

    const wrapper = mountWithQuery(CustomerListView)
    await flushPromises()

    const dataTable = wrapper.findComponent({ name: 'DataTable' })
    const columns = dataTable.props('columns')
    const createdAtCol = columns.find((c: any) => c.key === 'created_at')

    expect(createdAtCol.format).toBeDefined()
    const formatted = createdAtCol.format('2024-01-15T10:30:00Z')
    expect(formatted).toBe(new Date('2024-01-15T10:30:00Z').toLocaleDateString('pt-BR'))
  })

  it('formats empty created_at as empty string', async () => {
    const api = await getApi()
    vi.mocked(api.get).mockResolvedValueOnce({ data: [] })

    const wrapper = mountWithQuery(CustomerListView)
    await flushPromises()

    const dataTable = wrapper.findComponent({ name: 'DataTable' })
    const columns = dataTable.props('columns')
    const createdAtCol = columns.find((c: any) => c.key === 'created_at')

    expect(createdAtCol.format('')).toBe('')
    expect(createdAtCol.format(null)).toBe('')
    expect(createdAtCol.format(undefined)).toBe('')
  })

  it('formats phone and document columns', async () => {
    const api = await getApi()
    vi.mocked(api.get).mockResolvedValueOnce({ data: [] })

    const wrapper = mountWithQuery(CustomerListView)
    await flushPromises()

    const dataTable = wrapper.findComponent({ name: 'DataTable' })
    const columns = dataTable.props('columns')

    const phoneCol = columns.find((c: any) => c.key === 'phone')
    expect(phoneCol.format('11999999999')).toBe('(11) 99999-9999')
    expect(phoneCol.format(undefined)).toBe('—')

    const documentCol = columns.find((c: any) => c.key === 'document')
    expect(documentCol.format('12345678901', { document_type: 'CPF' })).toBe('123.456.789-01')
  })

  it('opens the create-customer modal when "Novo Cliente" is clicked', async () => {
    const api = await getApi()
    vi.mocked(api.get).mockResolvedValueOnce({ data: [] })

    const wrapper = mountWithQuery(CustomerListView)
    await flushPromises()

    const dialog = wrapper.findComponent({ name: 'Dialog' })
    expect(dialog.props('open')).toBeFalsy()

    const newBtn = wrapper.findAll('button').find((b) => b.text() === 'Novo Cliente')!
    await newBtn.trigger('click')

    expect(wrapper.findComponent({ name: 'Dialog' }).props('open')).toBe(true)
  })

  it('navigates to /customers/:id on row-click', async () => {
    const api = await getApi()
    const customer = makeCustomer({ id: 'cust-42' })
    vi.mocked(api.get).mockResolvedValueOnce({ data: [customer] })

    const wrapper = mountWithQuery(CustomerListView)
    await flushPromises()

    const dataTable = wrapper.findComponent({ name: 'DataTable' })
    dataTable.vm.$emit('row-click', customer)
    await wrapper.vm.$nextTick()

    expect(pushMock).toHaveBeenCalledWith('/customers/cust-42')
  })

  // NOTE: The component declares a local `fetchError` ref but never wires it
  // up to the query's `isError`/`error` state (useCustomerList only
  // destructures `data` and `isPending`). As a result, a failed fetch renders
  // no error UI at all — the DataTable simply displays with empty data. This
  // looks like a genuine regression from the pre-refactor version, which did
  // show an inline error + retry button. Flagged in the report instead of
  // patched here, per instructions not to touch source.
  it('renders an empty DataTable (no error UI) when the fetch fails', async () => {
    const api = await getApi()
    vi.mocked(api.get).mockRejectedValueOnce({ message: 'Erro de conexão com o servidor' })

    const wrapper = mountWithQuery(CustomerListView)
    await flushPromises()

    expect(wrapper.text()).not.toContain('Erro de conexão com o servidor')
    const dataTable = wrapper.findComponent({ name: 'DataTable' })
    expect(dataTable.props('data')).toHaveLength(0)
    expect(dataTable.props('loading')).toBe(false)
  })
})
