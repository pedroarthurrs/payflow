import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'

// Mock vue-router
const pushMock = vi.fn()
const routeParams = { value: { id: 'cust-1' } as Record<string, string> }
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: pushMock }),
  useRoute: () => ({ params: routeParams.value }),
}))

// Mock http
vi.mock('@/services/http', () => ({
  default: {
    get: vi.fn(),
    delete: vi.fn(),
  },
}))

// Mock useToast
const successMock = vi.fn()
const errorMock = vi.fn()
vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    success: successMock,
    error: errorMock,
  }),
}))

import CustomerDetailView from './CustomerDetailView.vue'

async function getHttp() {
  return (await import('@/services/http')).default
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

  async function mountWithData(
    customerData = makeCustomer(),
    addressData: any[] = [makeAddress()],
  ) {
    const http = await getHttp()
    vi.mocked(http.get).mockImplementation((url: string) => {
      if (url === '/v1/customers/cust-1') {
        return Promise.resolve({ data: customerData }) as any
      }
      if (url === '/v1/customers/cust-1/addresses') {
        return Promise.resolve({ data: addressData }) as any
      }
      return Promise.reject(new Error('unexpected url'))
    })
    const wrapper = mount(CustomerDetailView)
    await flushPromises()
    return wrapper
  }

  it('fetches customer and addresses on mount', async () => {
    const http = await getHttp()
    await mountWithData()
    expect(http.get).toHaveBeenCalledWith('/v1/customers/cust-1')
    expect(http.get).toHaveBeenCalledWith('/v1/customers/cust-1/addresses')
  })

  it('displays customer information', async () => {
    const wrapper = await mountWithData()
    expect(wrapper.find('.customer-detail__title').text()).toBe('João Silva')
    expect(wrapper.text()).toContain('joao@test.com')
    expect(wrapper.text()).toContain('11999999999')
    expect(wrapper.text()).toContain('12345678901')
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

  it('displays metadata as formatted JSON', async () => {
    const wrapper = await mountWithData()
    expect(wrapper.text()).toContain('"key"')
    expect(wrapper.text()).toContain('"value"')
  })

  it('shows dash when phone is missing', async () => {
    const wrapper = await mountWithData(makeCustomer({ phone: undefined }))
    const fields = wrapper.findAll('.customer-detail__field')
    const phoneField = fields.find((f) => f.find('dt').text() === 'Telefone')
    expect(phoneField?.find('dd').text()).toBe('—')
  })

  it('shows dash when metadata is missing', async () => {
    const wrapper = await mountWithData(makeCustomer({ metadata: undefined }))
    const fields = wrapper.findAll('.customer-detail__field')
    const metaField = fields.find((f) => f.find('dt').text() === 'Metadata')
    expect(metaField?.find('dd').text()).toBe('—')
  })

  it('renders Edit and Delete buttons', async () => {
    const wrapper = await mountWithData()
    expect(wrapper.find('.customer-detail__edit-btn').text()).toBe('Editar')
    expect(wrapper.find('.customer-detail__delete-btn').text()).toBe('Excluir')
  })

  it('navigates to edit page when Edit is clicked', async () => {
    const wrapper = await mountWithData()
    await wrapper.find('.customer-detail__edit-btn').trigger('click')
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
    expect(wrapper.find('.customer-detail__new-address-btn').text()).toBe('Novo Endereço')
  })

  it('navigates to new address page when "Novo Endereço" is clicked', async () => {
    const wrapper = await mountWithData()
    await wrapper.find('.customer-detail__new-address-btn').trigger('click')
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
    it('opens ConfirmDialog when Delete is clicked', async () => {
      const wrapper = await mountWithData()
      expect(wrapper.findComponent({ name: 'ConfirmDialog' }).props('visible')).toBe(false)
      await wrapper.find('.customer-detail__delete-btn').trigger('click')
      // First ConfirmDialog is for customer deletion
      const dialogs = wrapper.findAllComponents({ name: 'ConfirmDialog' })
      const customerDialog = dialogs[0]
      expect(customerDialog.props('visible')).toBe(true)
      expect(customerDialog.props('title')).toBe('Excluir Cliente')
    })

    it('renders hard delete checkbox in the dialog slot', async () => {
      const wrapper = await mountWithData()
      await wrapper.find('.customer-detail__delete-btn').trigger('click')
      expect(wrapper.find('.customer-detail__hard-delete').exists()).toBe(true)
      expect(wrapper.find('.customer-detail__hard-delete input[type="checkbox"]').exists()).toBe(true)
    })

    it('calls soft delete and navigates to /customers on confirm', async () => {
      const http = await getHttp()
      const wrapper = await mountWithData()

      await wrapper.find('.customer-detail__delete-btn').trigger('click')

      vi.mocked(http.delete).mockResolvedValueOnce({} as any)

      const dialogs = wrapper.findAllComponents({ name: 'ConfirmDialog' })
      dialogs[0].vm.$emit('confirm')
      await flushPromises()

      expect(http.delete).toHaveBeenCalledWith('/v1/customers/cust-1')
      expect(successMock).toHaveBeenCalledWith('Cliente excluído com sucesso')
      expect(pushMock).toHaveBeenCalledWith('/customers')
    })

    it('calls hard delete with ?hard=true when checkbox is checked', async () => {
      const http = await getHttp()
      const wrapper = await mountWithData()

      await wrapper.find('.customer-detail__delete-btn').trigger('click')

      // Check the hard delete checkbox
      await wrapper.find('.customer-detail__hard-delete input[type="checkbox"]').setValue(true)

      vi.mocked(http.delete).mockResolvedValueOnce({} as any)

      const dialogs = wrapper.findAllComponents({ name: 'ConfirmDialog' })
      dialogs[0].vm.$emit('confirm')
      await flushPromises()

      expect(http.delete).toHaveBeenCalledWith('/v1/customers/cust-1?hard=true')
      expect(successMock).toHaveBeenCalledWith('Cliente excluído com sucesso')
    })

    it('shows error toast on delete failure', async () => {
      const http = await getHttp()
      const wrapper = await mountWithData()

      await wrapper.find('.customer-detail__delete-btn').trigger('click')

      vi.mocked(http.delete).mockRejectedValueOnce({ message: 'Erro ao excluir' })

      const dialogs = wrapper.findAllComponents({ name: 'ConfirmDialog' })
      dialogs[0].vm.$emit('confirm')
      await flushPromises()

      expect(errorMock).toHaveBeenCalledWith('Erro ao excluir')
    })

    it('closes dialog on cancel', async () => {
      const wrapper = await mountWithData()
      await wrapper.find('.customer-detail__delete-btn').trigger('click')

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
      const http = await getHttp()
      const wrapper = await mountWithData()

      const dataTable = wrapper.findComponent({ name: 'DataTable' })
      dataTable.vm.$emit('delete', makeAddress({ id: 'addr-1' }))
      await wrapper.vm.$nextTick()

      vi.mocked(http.delete).mockResolvedValueOnce({} as any)
      // Mock the address list refresh
      vi.mocked(http.get).mockImplementation((url: string) => {
        if (url === '/v1/customers/cust-1/addresses') {
          return Promise.resolve({ data: [] }) as any
        }
        return Promise.reject(new Error('unexpected'))
      })

      const dialogs = wrapper.findAllComponents({ name: 'ConfirmDialog' })
      dialogs[1].vm.$emit('confirm')
      await flushPromises()

      expect(http.delete).toHaveBeenCalledWith('/v1/customers/cust-1/addresses/addr-1')
      expect(successMock).toHaveBeenCalledWith('Endereço excluído com sucesso')
    })

    it('shows error toast on address delete failure', async () => {
      const http = await getHttp()
      const wrapper = await mountWithData()

      const dataTable = wrapper.findComponent({ name: 'DataTable' })
      dataTable.vm.$emit('delete', makeAddress())
      await wrapper.vm.$nextTick()

      vi.mocked(http.delete).mockRejectedValueOnce({ message: 'Erro ao excluir endereço' })

      const dialogs = wrapper.findAllComponents({ name: 'ConfirmDialog' })
      dialogs[1].vm.$emit('confirm')
      await flushPromises()

      expect(errorMock).toHaveBeenCalledWith('Erro ao excluir endereço')
    })
  })

  describe('Error states', () => {
    it('shows error and retry button when customer fetch fails', async () => {
      const http = await getHttp()
      vi.mocked(http.get).mockImplementation((url: string) => {
        if (url === '/v1/customers/cust-1') {
          return Promise.reject({ message: 'Cliente não encontrado' })
        }
        return Promise.resolve({ data: [] }) as any
      })

      const wrapper = mount(CustomerDetailView)
      await flushPromises()

      expect(wrapper.find('.customer-detail__error').exists()).toBe(true)
      expect(wrapper.find('.customer-detail__error-message').text()).toBe('Cliente não encontrado')
      expect(wrapper.find('.customer-detail__retry-btn').exists()).toBe(true)
    })

    it('retries customer fetch when retry button is clicked', async () => {
      const http = await getHttp()
      vi.mocked(http.get).mockImplementation((url: string) => {
        if (url === '/v1/customers/cust-1') {
          return Promise.reject({ message: 'Erro' })
        }
        return Promise.resolve({ data: [] }) as any
      })

      const wrapper = mount(CustomerDetailView)
      await flushPromises()

      expect(wrapper.find('.customer-detail__error').exists()).toBe(true)

      // Mock success for retry
      vi.mocked(http.get).mockImplementation((url: string) => {
        if (url === '/v1/customers/cust-1') {
          return Promise.resolve({ data: makeCustomer() }) as any
        }
        return Promise.resolve({ data: [] }) as any
      })

      await wrapper.find('.customer-detail__retry-btn').trigger('click')
      await flushPromises()

      expect(wrapper.find('.customer-detail__error').exists()).toBe(false)
      expect(wrapper.find('.customer-detail__title').text()).toBe('João Silva')
    })

    it('shows address error when address fetch fails', async () => {
      const http = await getHttp()
      vi.mocked(http.get).mockImplementation((url: string) => {
        if (url === '/v1/customers/cust-1') {
          return Promise.resolve({ data: makeCustomer() }) as any
        }
        if (url === '/v1/customers/cust-1/addresses') {
          return Promise.reject({ message: 'Erro ao carregar endereços' })
        }
        return Promise.reject(new Error('unexpected'))
      })

      const wrapper = mount(CustomerDetailView)
      await flushPromises()

      expect(wrapper.find('.customer-detail__address-error').exists()).toBe(true)
      expect(wrapper.find('.customer-detail__address-error-message').text()).toBe('Erro ao carregar endereços')
    })

    it('retries address fetch when address retry button is clicked', async () => {
      const http = await getHttp()
      let addressCallCount = 0
      vi.mocked(http.get).mockImplementation((url: string) => {
        if (url === '/v1/customers/cust-1') {
          return Promise.resolve({ data: makeCustomer() }) as any
        }
        if (url === '/v1/customers/cust-1/addresses') {
          addressCallCount++
          if (addressCallCount === 1) {
            return Promise.reject({ message: 'Erro' })
          }
          return Promise.resolve({ data: [makeAddress()] }) as any
        }
        return Promise.reject(new Error('unexpected'))
      })

      const wrapper = mount(CustomerDetailView)
      await flushPromises()

      expect(wrapper.find('.customer-detail__address-error').exists()).toBe(true)

      await wrapper.find('.customer-detail__address-retry-btn').trigger('click')
      await flushPromises()

      expect(wrapper.find('.customer-detail__address-error').exists()).toBe(false)
      const dataTable = wrapper.findComponent({ name: 'DataTable' })
      expect(dataTable.props('data')).toHaveLength(1)
    })
  })

  it('shows loading spinner while fetching customer', async () => {
    const http = await getHttp()
    let resolveCustomer: (v: any) => void
    vi.mocked(http.get).mockImplementation((url: string) => {
      if (url === '/v1/customers/cust-1') {
        return new Promise((r) => { resolveCustomer = r }) as any
      }
      return Promise.resolve({ data: [] }) as any
    })

    const wrapper = mount(CustomerDetailView)
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.customer-detail__loading').exists()).toBe(true)
    expect(wrapper.findComponent({ name: 'LoadingSpinner' }).exists()).toBe(true)

    resolveCustomer!({ data: makeCustomer() })
    await flushPromises()

    expect(wrapper.find('.customer-detail__loading').exists()).toBe(false)
  })
})
