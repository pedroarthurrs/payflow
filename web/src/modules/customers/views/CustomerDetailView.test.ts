import { mountWithQuery } from '@/core/utils/wrappers'
import { flushPromises } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

// Mock vue-router
const pushMock = vi.fn()
const routeParams = { value: { id: 'cust-1' } as Record<string, string> }
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: pushMock }),
  useRoute: () => ({ params: routeParams.value }),
}))

// Mock the API client — useCustomers composables call `api.get/post/put/delete`
// (unwrapped responses, not the raw axios `default` instance).
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

// Mock useToast
const successMock = vi.fn()
const errorMock = vi.fn()
vi.mock('@/core/composables/useToast', () => ({
  useToast: () => ({
    success: successMock,
    error: errorMock,
    warning: vi.fn(),
  }),
}))

// Mock the RBAC store (Can.vue and useRBAC both import it from './store',
// which resolves to this same module regardless of the relative path used).
// Grant every permission so the Edit/Delete/Novo Endereço buttons render.
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

import CustomerDetailView from './CustomerDetailView.vue'

async function getApi() {
  return (await import('@/core/api/http')).api
}

function makeCustomer(overrides: Record<string, unknown> = {}) {
  return {
    id: 'cust-1',
    name: 'João Silva',
    email: 'joao@test.com',
    phone: '11999999999',
    document: '12345678901',
    document_type: 'CPF',
    status: 'active',
    metadata: { key: 'value' },
    created_at: '2024-01-15T10:30:00Z',
    updated_at: '2024-01-16T14:00:00Z',
    ...overrides,
  }
}

function makeAddress(overrides: Record<string, unknown> = {}) {
  return {
    id: 'addr-1',
    customer_id: 'cust-1',
    type: 'billing',
    postal_code: '01001-000',
    street: 'Rua Exemplo',
    number: '100',
    complement: 'Sala 1',
    neighborhood: 'Centro',
    city: 'São Paulo',
    state: 'SP',
    country: 'BR',
    is_main: true,
    created_at: '2024-01-15T10:30:00Z',
    ...overrides,
  }
}

describe('CustomerDetailView', () => {
  beforeEach(() => {
    routeParams.value = { id: 'cust-1' }
    vi.clearAllMocks()
  })

  // `api.get` unwraps the axios response, so queries resolve with the raw
  // payload directly (no `{ data }` envelope) — except for the paginated
  // addresses endpoint, which really does return `{ data: [...] }`.
  async function mountWithData(
    customerData = makeCustomer(),
    addressData: any[] = [makeAddress()],
  ) {
    const api = await getApi()
    vi.mocked(api.get).mockImplementation((url: string) => {
      if (url === '/v1/customers/cust-1') {
        return Promise.resolve(customerData) as any
      }
      if (url === '/v1/customers/cust-1/addresses') {
        return Promise.resolve({ data: addressData }) as any
      }
      return Promise.reject(new Error('unexpected url'))
    })
    const wrapper = mountWithQuery(CustomerDetailView)
    await flushPromises()
    return wrapper
  }

  // The customer-level Editar/Excluir buttons render first in DOM order,
  // before the address DataTable's per-row action buttons of the same name.
  function findTopButton(wrapper: ReturnType<typeof mountWithQuery>, text: string) {
    return wrapper.findAll('button').find((b) => b.text() === text)!
  }

  it('fetches customer and addresses on mount', async () => {
    const api = await getApi()
    await mountWithData()
    expect(api.get).toHaveBeenCalledWith('/v1/customers/cust-1')
    expect(api.get).toHaveBeenCalledWith('/v1/customers/cust-1/addresses')
  })

  it('displays customer information', async () => {
    const wrapper = await mountWithData()
    expect(wrapper.find('h1').text()).toBe('João Silva')
    expect(wrapper.text()).toContain('joao@test.com')
    expect(wrapper.text()).toContain('(11) 99999-9999')
    expect(wrapper.text()).toContain('123.456.789-01')
    expect(wrapper.text()).toContain('CPF')
    expect(wrapper.text()).toContain('active')
  })

  it('displays formatted dates', async () => {
    const wrapper = await mountWithData()
    const createdDate = new Date('2024-01-15T10:30:00Z').toLocaleDateString('pt-BR')
    const updatedDate = new Date('2024-01-16T14:00:00Z').toLocaleDateString('pt-BR')
    expect(wrapper.text()).toContain(createdDate)
    expect(wrapper.text()).toContain(updatedDate)
  })

  it('displays metadata as key/value pairs', async () => {
    const wrapper = await mountWithData()
    expect(wrapper.text()).toContain('key:')
    expect(wrapper.text()).toContain('value')
  })

  it('shows dash when phone is missing', async () => {
    const wrapper = await mountWithData(makeCustomer({ phone: undefined }))
    const phoneDt = wrapper.findAll('dt').find((d) => d.text() === 'Telefone')
    const phoneDd = phoneDt?.element.nextElementSibling
    expect(phoneDd?.textContent).toBe('—')
  })

  it('shows dash when metadata is missing', async () => {
    const wrapper = await mountWithData(makeCustomer({ metadata: undefined }))
    const metaDt = wrapper.findAll('dt').find((d) => d.text() === 'Metadata')
    const metaDd = metaDt?.element.nextElementSibling
    expect(metaDd?.textContent).toBe('—')
  })

  it('renders Editar and Excluir buttons', async () => {
    const wrapper = await mountWithData()
    expect(findTopButton(wrapper, 'Editar').exists()).toBe(true)
    expect(findTopButton(wrapper, 'Excluir').exists()).toBe(true)
  })

  it('navigates to edit page when Editar is clicked', async () => {
    const wrapper = await mountWithData()
    await findTopButton(wrapper, 'Editar').trigger('click')
    expect(pushMock).toHaveBeenCalledWith('/customers/cust-1/edit')
  })

  it('renders addresses in DataTable with correct columns', async () => {
    const wrapper = await mountWithData()
    const dataTable = wrapper.findComponent({ name: 'DataTable' })
    expect(dataTable.exists()).toBe(true)
    const columns = dataTable.props('columns')
    expect(columns).toHaveLength(6)
    expect(columns.map((c: any) => c.key)).toEqual([
      'type', 'street', 'city', 'state', 'postal_code', 'is_main',
    ])
  })

  it('formats is_main column as Sim/Não', async () => {
    const wrapper = await mountWithData()
    const dataTable = wrapper.findComponent({ name: 'DataTable' })
    const columns = dataTable.props('columns')
    const isMainCol = columns.find((c: any) => c.key === 'is_main')
    expect(isMainCol.format(true)).toBe('Sim')
    expect(isMainCol.format(false)).toBe('Não')
  })

  it('renders "Novo Endereço" button', async () => {
    const wrapper = await mountWithData()
    expect(wrapper.findAll('button').some((b) => b.text() === 'Novo Endereço')).toBe(true)
  })

  it('navigates to new address page when "Novo Endereço" is clicked', async () => {
    const wrapper = await mountWithData()
    const btn = wrapper.findAll('button').find((b) => b.text() === 'Novo Endereço')!
    await btn.trigger('click')
    expect(pushMock).toHaveBeenCalledWith('/customers/cust-1/addresses/new')
  })

  it('navigates to edit address on DataTable edit event', async () => {
    const wrapper = await mountWithData()
    const dataTable = wrapper.findComponent({ name: 'DataTable' })
    dataTable.vm.$emit('edit', makeAddress({ id: 'addr-42' }))
    await wrapper.vm.$nextTick()
    expect(pushMock).toHaveBeenCalledWith('/customers/cust-1/addresses/addr-42/edit')
  })

  describe('Customer deletion', () => {
    it('opens ConfirmDialog when Excluir is clicked', async () => {
      const wrapper = await mountWithData()
      const dialogs = wrapper.findAllComponents({ name: 'ConfirmDialog' })
      expect(dialogs[0].props('visible')).toBe(false)

      await findTopButton(wrapper, 'Excluir').trigger('click')

      const dialogsAfter = wrapper.findAllComponents({ name: 'ConfirmDialog' })
      expect(dialogsAfter[0].props('visible')).toBe(true)
      expect(dialogsAfter[0].props('title')).toBe('Excluir Cliente')
    })

    it('renders hard delete checkbox in the dialog slot', async () => {
      const wrapper = await mountWithData()
      await findTopButton(wrapper, 'Excluir').trigger('click')
      await wrapper.vm.$nextTick()

      const checkbox = document.body.querySelector('input[type="checkbox"]')
      expect(checkbox).toBeTruthy()
      expect(document.body.textContent).toContain('Exclusão permanente')
    })

    it('calls soft delete and navigates to /customers on confirm', async () => {
      const api = await getApi()
      const wrapper = await mountWithData()

      await findTopButton(wrapper, 'Excluir').trigger('click')

      vi.mocked(api.delete).mockResolvedValueOnce(undefined as any)

      const dialogs = wrapper.findAllComponents({ name: 'ConfirmDialog' })
      dialogs[0].vm.$emit('confirm')
      await flushPromises()

      expect(api.delete).toHaveBeenCalledWith('/v1/customers/cust-1')
      expect(successMock).toHaveBeenCalledWith('Cliente excluído com sucesso')
      expect(pushMock).toHaveBeenCalledWith('/customers')
    })

    it('shows error toast on delete failure', async () => {
      const api = await getApi()
      const wrapper = await mountWithData()

      await findTopButton(wrapper, 'Excluir').trigger('click')

      vi.mocked(api.delete).mockRejectedValueOnce({ message: 'Erro ao excluir' })

      const dialogs = wrapper.findAllComponents({ name: 'ConfirmDialog' })
      dialogs[0].vm.$emit('confirm')
      await flushPromises()

      expect(errorMock).toHaveBeenCalledWith('Erro ao excluir')
    })

    it('closes dialog on cancel', async () => {
      const wrapper = await mountWithData()
      await findTopButton(wrapper, 'Excluir').trigger('click')

      const dialogs = wrapper.findAllComponents({ name: 'ConfirmDialog' })
      expect(dialogs[0].props('visible')).toBe(true)

      dialogs[0].vm.$emit('cancel')
      await wrapper.vm.$nextTick()

      expect(wrapper.findAllComponents({ name: 'ConfirmDialog' })[0].props('visible')).toBe(false)
    })
  })

  describe('Address deletion', () => {
    it('opens address ConfirmDialog on DataTable delete event', async () => {
      const wrapper = await mountWithData()
      const dataTable = wrapper.findComponent({ name: 'DataTable' })
      dataTable.vm.$emit('delete', makeAddress())
      await wrapper.vm.$nextTick()

      const dialogs = wrapper.findAllComponents({ name: 'ConfirmDialog' })
      const addressDialog = dialogs[1]
      expect(addressDialog.props('visible')).toBe(true)
      expect(addressDialog.props('title')).toBe('Excluir Endereço')
    })

    it('deletes address and refreshes list on confirm', async () => {
      const api = await getApi()
      const wrapper = await mountWithData()

      const dataTable = wrapper.findComponent({ name: 'DataTable' })
      dataTable.vm.$emit('delete', makeAddress({ id: 'addr-1' }))
      await wrapper.vm.$nextTick()

      vi.mocked(api.delete).mockResolvedValueOnce(undefined as any)

      const dialogs = wrapper.findAllComponents({ name: 'ConfirmDialog' })
      dialogs[1].vm.$emit('confirm')
      await flushPromises()

      expect(api.delete).toHaveBeenCalledWith('/v1/customers/cust-1/addresses/addr-1')
      expect(successMock).toHaveBeenCalledWith('Endereço excluído com sucesso')
    })

    it('shows error toast on address delete failure', async () => {
      const api = await getApi()
      const wrapper = await mountWithData()

      const dataTable = wrapper.findComponent({ name: 'DataTable' })
      dataTable.vm.$emit('delete', makeAddress())
      await wrapper.vm.$nextTick()

      vi.mocked(api.delete).mockRejectedValueOnce({ message: 'Erro ao excluir endereço' })

      const dialogs = wrapper.findAllComponents({ name: 'ConfirmDialog' })
      dialogs[1].vm.$emit('confirm')
      await flushPromises()

      expect(errorMock).toHaveBeenCalledWith('Erro ao excluir endereço')
    })
  })

  describe('Error states', () => {
    it('shows an error message when customer fetch fails', async () => {
      const api = await getApi()
      vi.mocked(api.get).mockImplementation((url: string) => {
        if (url === '/v1/customers/cust-1') {
          return Promise.reject({ message: 'Cliente não encontrado' })
        }
        return Promise.resolve({ data: [] }) as any
      })

      const wrapper = mountWithQuery(CustomerDetailView)
      await flushPromises()

      expect(wrapper.text()).toContain('Cliente não encontrado')
      // No customer data available yet, so the detail section is not shown.
      expect(wrapper.find('h1').exists()).toBe(false)
    })
  })

  it('shows loading spinner while fetching customer', async () => {
    const api = await getApi()
    let resolveCustomer: (v: any) => void
    vi.mocked(api.get).mockImplementation((url: string) => {
      if (url === '/v1/customers/cust-1') {
        return new Promise((r) => { resolveCustomer = r }) as any
      }
      return Promise.resolve({ data: [] }) as any
    })

    const wrapper = mountWithQuery(CustomerDetailView)
    await wrapper.vm.$nextTick()

    expect(wrapper.findComponent({ name: 'LoadingSpinner' }).exists()).toBe(true)

    resolveCustomer!(makeCustomer())
    await flushPromises()

    expect(wrapper.find('h1').text()).toBe('João Silva')
  })
})
