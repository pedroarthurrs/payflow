import { flushPromises } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mountWithQuery } from '@/core/utils/wrappers'

// Mock vue-router
const pushMock = vi.fn()
const routeParams = { value: {} as Record<string, string> }
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: pushMock }),
  useRoute: () => ({ params: routeParams.value }),
}))

// Mock the HTTP client — the view and its composables import the named `api`
// helper (already unwraps `.data`), not the default axios instance.
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
const warningMock = vi.fn()
vi.mock('@/core/composables/useToast', () => ({
  useToast: () => ({
    success: successMock,
    error: errorMock,
    warning: warningMock,
  }),
}))

import AddressFormView from './AddressFormView.vue'

async function getApi() {
  return (await import('@/core/api/http')).api
}

// vee-validate's typed-schema (zod) validation resolves a couple of event-loop
// turns after the `submit` event — a single `flushPromises()` (a microtask
// flush) is not enough to observe the result. `vi.waitFor` polls until the
// expected side effect (an HTTP call or a rendered DOM node) shows up.
async function waitUntil(predicate: () => boolean): Promise<void> {
  await vi.waitFor(() => {
    if (!predicate()) throw new Error('condition not met yet')
  })
}

function makeAddress(overrides: Record<string, unknown> = {}) {
  return {
    id: 'addr-1',
    customer_id: 'cust-1',
    type: 'shipping',
    postal_code: '01001-000',
    street: 'Rua Teste',
    number: '123',
    complement: 'Apto 4',
    neighborhood: 'Centro',
    city: 'São Paulo',
    state: 'SP',
    country: 'BR',
    is_main: true,
    created_at: '2024-01-15T10:30:00Z',
    ...overrides,
  }
}

describe('AddressFormView', () => {
  beforeEach(() => {
    routeParams.value = { id: 'cust-1' }
    vi.clearAllMocks()
  })

  function mountView() {
    return mountWithQuery(AddressFormView, {
      global: { stubs: { RouterLink: { template: '<a><slot /></a>' } } },
    })
  }

  function findField(wrapper: ReturnType<typeof mountView>, label: string) {
    const field = wrapper
      .findAllComponents({ name: 'FormField' })
      .find((f) => f.props('label') === label)
    if (!field) throw new Error(`FormField with label "${label}" not found`)
    return field
  }

  function findButton(wrapper: ReturnType<typeof mountView>, text: string) {
    const button = wrapper.findAll('button').find((b) => b.text() === text)
    if (!button) throw new Error(`Button with text "${text}" not found`)
    return button
  }

  async function fillRequiredFields(
    wrapper: ReturnType<typeof mountView>,
    overrides: Record<string, unknown> = {},
  ) {
    const values: Record<string, unknown> = {
      CEP: '01001-000',
      Logradouro: 'Rua X',
      Cidade: 'São Paulo',
      Estado: 'SP',
      ...overrides,
    }
    for (const [label, value] of Object.entries(values)) {
      findField(wrapper, label).vm.$emit('update:modelValue', value)
    }
    await wrapper.vm.$nextTick()
  }

  describe('Create mode', () => {
    it('renders title "Novo Endereço" and empty form fields', () => {
      const wrapper = mountView()
      expect(wrapper.find('h1').text()).toBe('Novo Endereço')
      expect(wrapper.find('button[type="submit"]').text()).toBe('Criar')
    })

    it('defaults country to "BR" and type to "shipping"', () => {
      const wrapper = mountView()
      expect(findField(wrapper, 'País').props('modelValue')).toBe('BR')
      expect(findField(wrapper, 'Tipo').props('modelValue')).toBe('shipping')
    })

    it('submits POST to /v1/customers/:id/addresses with form data', async () => {
      const api = await getApi()
      vi.mocked(api.post).mockResolvedValueOnce(makeAddress({ id: 'new-addr' }))

      const wrapper = mountView()
      await fillRequiredFields(wrapper, {
        Tipo: 'billing',
        CEP: '01001-000',
        Logradouro: 'Rua Nova',
        Número: '456',
        Complemento: 'Sala 2',
        Bairro: 'Centro',
        Cidade: 'Rio de Janeiro',
        Estado: 'RJ',
        'Endereço principal': true,
      })

      await wrapper.find('form').trigger('submit')
      await waitUntil(() => pushMock.mock.calls.length > 0)

      expect(api.post).toHaveBeenCalledWith('/v1/customers/cust-1/addresses', {
        type: 'billing',
        postal_code: '01001-000',
        street: 'Rua Nova',
        number: '456',
        complement: 'Sala 2',
        neighborhood: 'Centro',
        city: 'Rio de Janeiro',
        state: 'RJ',
        country: 'BR',
        is_main: true,
      })
      expect(successMock).toHaveBeenCalledWith('Endereço criado com sucesso')
      expect(pushMock).toHaveBeenCalledWith('/customers/cust-1')
    })

    it('submits without optional fields when they are empty', async () => {
      const api = await getApi()
      vi.mocked(api.post).mockResolvedValueOnce(makeAddress())

      const wrapper = mountView()
      await fillRequiredFields(wrapper)

      await wrapper.find('form').trigger('submit')
      await waitUntil(() => vi.mocked(api.post).mock.calls.length > 0)

      expect(api.post).toHaveBeenCalledWith('/v1/customers/cust-1/addresses', expect.objectContaining({
        number: undefined,
        complement: undefined,
        neighborhood: undefined,
      }))
    })

    it('displays field-level validation errors from 400 response', async () => {
      const api = await getApi()
      vi.mocked(api.post).mockRejectedValueOnce({
        message: 'Validation failed',
        status: 400,
        validationErrors: {
          postal_code: 'CEP já cadastrado para este cliente',
          street: 'Logradouro inválido',
        },
      })

      const wrapper = mountView()
      await fillRequiredFields(wrapper)

      await wrapper.find('form').trigger('submit')
      await waitUntil(() => !!findField(wrapper, 'CEP').props('error'))

      expect(findField(wrapper, 'CEP').props('error')).toBe('CEP já cadastrado para este cliente')
      expect(findField(wrapper, 'Logradouro').props('error')).toBe('Logradouro inválido')
    })

    it('shows generic error for non-validation failures', async () => {
      const api = await getApi()
      vi.mocked(api.post).mockRejectedValueOnce({
        message: 'Erro de conexão com o servidor',
        status: 0,
      })

      const wrapper = mountView()
      await fillRequiredFields(wrapper)

      await wrapper.find('form').trigger('submit')
      await waitUntil(() => wrapper.text().includes('Erro de conexão com o servidor'))

      expect(wrapper.text()).toContain('Erro de conexão com o servidor')
    })

    it('disables submit button and shows "Salvando..." while loading', async () => {
      const api = await getApi()
      let resolvePost: (v: unknown) => void
      vi.mocked(api.post).mockReturnValueOnce(
        new Promise((resolve) => { resolvePost = resolve }),
      )

      const wrapper = mountView()
      await fillRequiredFields(wrapper)

      await wrapper.find('form').trigger('submit')
      await waitUntil(() => wrapper.find('button[type="submit"]').text() === 'Salvando...')

      const submitBtn = wrapper.find('button[type="submit"]')
      expect(submitBtn.text()).toBe('Salvando...')
      expect(submitBtn.attributes('disabled')).toBeDefined()

      resolvePost!(makeAddress())
      await flushPromises()
    })

    it('navigates to customer detail when cancel is clicked', async () => {
      const wrapper = mountView()
      await findButton(wrapper, 'Cancelar').trigger('click')
      expect(pushMock).toHaveBeenCalledWith('/customers/cust-1')
    })
  })

  describe('Edit mode', () => {
    beforeEach(() => {
      routeParams.value = { id: 'cust-1', addressId: 'addr-1' }
    })

    it('renders title "Editar Endereço" and fetches address data', async () => {
      const api = await getApi()
      vi.mocked(api.get).mockResolvedValueOnce(makeAddress())

      const wrapper = mountView()
      await waitUntil(() => findField(wrapper, 'Logradouro').props('modelValue') === 'Rua Teste')

      expect(wrapper.find('h1').text()).toBe('Editar Endereço')
      expect(api.get).toHaveBeenCalledWith('/v1/customers/cust-1/addresses/addr-1')
      expect(wrapper.find('button[type="submit"]').text()).toBe('Salvar')
    })

    it('pre-fills form fields with address data', async () => {
      const api = await getApi()
      vi.mocked(api.get).mockResolvedValueOnce(makeAddress())

      const wrapper = mountView()
      await waitUntil(() => findField(wrapper, 'Logradouro').props('modelValue') === 'Rua Teste')

      expect(findField(wrapper, 'Tipo').props('modelValue')).toBe('shipping')
      expect(findField(wrapper, 'CEP').props('modelValue')).toBe('01001-000')
      expect(findField(wrapper, 'Logradouro').props('modelValue')).toBe('Rua Teste')
      expect(findField(wrapper, 'Número').props('modelValue')).toBe('123')
      expect(findField(wrapper, 'Complemento').props('modelValue')).toBe('Apto 4')
      expect(findField(wrapper, 'Bairro').props('modelValue')).toBe('Centro')
      expect(findField(wrapper, 'Cidade').props('modelValue')).toBe('São Paulo')
      expect(findField(wrapper, 'Estado').props('modelValue')).toBe('SP')
      expect(findField(wrapper, 'País').props('modelValue')).toBe('BR')
      expect(findField(wrapper, 'Endereço principal').props('modelValue')).toBe(true)
    })

    it('pre-fills with empty optional fields when not present', async () => {
      const api = await getApi()
      vi.mocked(api.get).mockResolvedValueOnce(
        makeAddress({ number: undefined, complement: undefined, neighborhood: undefined }),
      )

      const wrapper = mountView()
      await waitUntil(() => findField(wrapper, 'Logradouro').props('modelValue') === 'Rua Teste')

      expect(findField(wrapper, 'Número').props('modelValue')).toBe('')
      expect(findField(wrapper, 'Complemento').props('modelValue')).toBe('')
      expect(findField(wrapper, 'Bairro').props('modelValue')).toBe('')
    })

    it('submits PUT to /v1/customers/:id/addresses/:addressId with updated data', async () => {
      const api = await getApi()
      vi.mocked(api.get).mockResolvedValueOnce(makeAddress())

      const wrapper = mountView()
      await waitUntil(() => findField(wrapper, 'Logradouro').props('modelValue') === 'Rua Teste')

      findField(wrapper, 'Logradouro').vm.$emit('update:modelValue', 'Rua Atualizada')
      await wrapper.vm.$nextTick()

      vi.mocked(api.put).mockResolvedValueOnce(makeAddress({ street: 'Rua Atualizada' }))

      await wrapper.find('form').trigger('submit')
      await waitUntil(() => pushMock.mock.calls.length > 0)

      expect(api.put).toHaveBeenCalledWith('/v1/customers/cust-1/addresses/addr-1', expect.objectContaining({
        street: 'Rua Atualizada',
        postal_code: '01001-000',
      }))
      expect(successMock).toHaveBeenCalledWith('Endereço atualizado com sucesso')
      expect(pushMock).toHaveBeenCalledWith('/customers/cust-1')
    })

    it('shows fetch error with retry button when getById fails', async () => {
      const api = await getApi()
      vi.mocked(api.get).mockRejectedValueOnce({
        message: 'Erro interno',
        status: 500,
      })

      const wrapper = mountView()
      await waitUntil(() => wrapper.text().includes('Erro interno'))

      expect(wrapper.text()).toContain('Erro interno')
      expect(wrapper.findAll('button').some((b) => b.text() === 'Tentar novamente')).toBe(true)
    })

    it('shows not found state with back link when getById returns 404', async () => {
      const api = await getApi()
      vi.mocked(api.get).mockRejectedValueOnce({
        message: 'Endereço não encontrado',
        status: 404,
      })

      const wrapper = mountView()
      await waitUntil(() => wrapper.text().includes('Endereço não encontrado'))

      expect(wrapper.text()).toContain('Endereço não encontrado')
      expect(wrapper.find('a').exists()).toBe(true)
      expect(wrapper.findAll('button').some((b) => b.text() === 'Tentar novamente')).toBe(false)
    })

    it('retries loading address when retry button is clicked', async () => {
      const api = await getApi()
      vi.mocked(api.get).mockRejectedValueOnce({ message: 'Erro', status: 500 })

      const wrapper = mountView()
      await waitUntil(() => wrapper.findAll('button').some((b) => b.text() === 'Tentar novamente'))

      vi.mocked(api.get).mockResolvedValueOnce(makeAddress())
      await findButton(wrapper, 'Tentar novamente').trigger('click')
      await waitUntil(() => findField(wrapper, 'Logradouro').props('modelValue') === 'Rua Teste')

      expect(wrapper.findAll('button').some((b) => b.text() === 'Tentar novamente')).toBe(false)
      expect(findField(wrapper, 'Logradouro').props('modelValue')).toBe('Rua Teste')
    })
  })
})
