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

import CustomerFormView from './CustomerFormView.vue'

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

function makeCustomer(overrides: Record<string, unknown> = {}) {
  return {
    id: 'cust-1',
    name: 'João Silva',
    email: 'joao@test.com',
    phone: '11999999999',
    document: '98765432100',
    document_type: 'cpf',
    status: 'active',
    metadata: { key: 'value' },
    created_at: '2024-01-15T10:30:00Z',
    updated_at: '2024-01-15T10:30:00Z',
    ...overrides,
  }
}

describe('CustomerFormView', () => {
  beforeEach(() => {
    routeParams.value = {}
    vi.clearAllMocks()
  })

  function mountView() {
    return mountWithQuery(CustomerFormView, {
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
      Nome: 'Test',
      Email: 'test@test.com',
      Documento: '52998224725',
      'Tipo de Documento': 'cpf',
      ...overrides,
    }
    for (const [label, value] of Object.entries(values)) {
      findField(wrapper, label).vm.$emit('update:modelValue', value)
    }
    await wrapper.vm.$nextTick()
  }

  describe('Create mode', () => {
    it('renders title "Novo Cliente" and empty form fields', () => {
      const wrapper = mountView()
      expect(wrapper.find('h1').text()).toBe('Novo Cliente')
      expect(wrapper.find('button[type="submit"]').text()).toBe('Criar')
    })

    it('submits POST to /v1/customers with form data', async () => {
      const api = await getApi()
      const created = makeCustomer({ id: 'new-1' })
      vi.mocked(api.post).mockResolvedValueOnce(created)

      const wrapper = mountView()
      await fillRequiredFields(wrapper, {
        Nome: 'Maria',
        Email: 'maria@test.com',
        Telefone: '11888888888',
        Documento: '98765432100',
        'Tipo de Documento': 'cnpj',
      })
      findField(wrapper, 'Metadata (JSON)').vm.$emit('update:modelValue', '{"foo":"bar"}')
      await wrapper.vm.$nextTick()

      await wrapper.find('form').trigger('submit')
      await waitUntil(() => pushMock.mock.calls.length > 0)

      expect(api.post).toHaveBeenCalledWith('/v1/customers', {
        name: 'Maria',
        email: 'maria@test.com',
        phone: '11888888888',
        document: '98765432100',
        document_type: 'cnpj',
        metadata: { foo: 'bar' },
      })
      expect(successMock).toHaveBeenCalledWith('Cliente criado com sucesso')
      expect(pushMock).toHaveBeenCalledWith('/customers/new-1')
    })

    it('submits without metadata when textarea is empty', async () => {
      const api = await getApi()
      vi.mocked(api.post).mockResolvedValueOnce(makeCustomer({ id: 'c-2' }))

      const wrapper = mountView()
      await fillRequiredFields(wrapper)

      await wrapper.find('form').trigger('submit')
      await waitUntil(() => vi.mocked(api.post).mock.calls.length > 0)

      expect(api.post).toHaveBeenCalledWith('/v1/customers', expect.objectContaining({
        metadata: undefined,
      }))
    })

    it('shows validation error on invalid JSON metadata and does not submit', async () => {
      const api = await getApi()
      const wrapper = mountView()
      await fillRequiredFields(wrapper)
      findField(wrapper, 'Metadata (JSON)').vm.$emit('update:modelValue', '{invalid}')
      await wrapper.vm.$nextTick()

      await wrapper.find('form').trigger('submit')
      await waitUntil(() => findField(wrapper, 'Metadata (JSON)').props('error') === 'JSON inválido')

      expect(findField(wrapper, 'Metadata (JSON)').props('error')).toBe('JSON inválido')
      expect(api.post).not.toHaveBeenCalled()
    })

    it('displays field-level validation errors from 400 response', async () => {
      const api = await getApi()
      vi.mocked(api.post).mockRejectedValueOnce({
        message: 'Validation failed',
        status: 400,
        validationErrors: {
          name: 'Nome já cadastrado',
          email: 'Email já cadastrado',
        },
      })

      const wrapper = mountView()
      await fillRequiredFields(wrapper)

      await wrapper.find('form').trigger('submit')
      await waitUntil(() => !!findField(wrapper, 'Nome').props('error'))

      expect(findField(wrapper, 'Nome').props('error')).toBe('Nome já cadastrado')
      expect(findField(wrapper, 'Email').props('error')).toBe('Email já cadastrado')
    })

    it('displays 409 conflict error in the form', async () => {
      const api = await getApi()
      vi.mocked(api.post).mockRejectedValueOnce({
        message: 'Cliente com este documento já existe',
        status: 409,
      })

      const wrapper = mountView()
      await fillRequiredFields(wrapper)

      await wrapper.find('form').trigger('submit')
      await waitUntil(() => wrapper.text().includes('Cliente com este documento já existe'))

      expect(wrapper.text()).toContain('Cliente com este documento já existe')
    })

    it('shows generic error for other failures', async () => {
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

      resolvePost!(makeCustomer())
      await flushPromises()
    })

    it('navigates to /customers when cancel is clicked', async () => {
      const wrapper = mountView()
      await findButton(wrapper, 'Cancelar').trigger('click')
      expect(pushMock).toHaveBeenCalledWith('/customers')
    })
  })

  describe('Edit mode', () => {
    beforeEach(() => {
      routeParams.value = { id: 'cust-1' }
    })

    it('renders title "Editar Cliente" and fetches customer data', async () => {
      const api = await getApi()
      vi.mocked(api.get).mockResolvedValueOnce(makeCustomer())

      const wrapper = mountView()
      await waitUntil(() => findField(wrapper, 'Nome').props('modelValue') === 'João Silva')

      expect(wrapper.find('h1').text()).toBe('Editar Cliente')
      expect(api.get).toHaveBeenCalledWith('/v1/customers/cust-1')
      expect(wrapper.find('button[type="submit"]').text()).toBe('Salvar')
    })

    it('pre-fills form fields with customer data', async () => {
      const api = await getApi()
      vi.mocked(api.get).mockResolvedValueOnce(makeCustomer())

      const wrapper = mountView()
      await waitUntil(() => findField(wrapper, 'Nome').props('modelValue') === 'João Silva')

      expect(findField(wrapper, 'Nome').props('modelValue')).toBe('João Silva')
      expect(findField(wrapper, 'Email').props('modelValue')).toBe('joao@test.com')
      expect(findField(wrapper, 'Telefone').props('modelValue')).toBe('11999999999')
      expect(findField(wrapper, 'Documento').props('modelValue')).toBe('98765432100')
      expect(findField(wrapper, 'Tipo de Documento').props('modelValue')).toBe('cpf')
      expect(findField(wrapper, 'Metadata (JSON)').props('modelValue')).toContain('"key"')
    })

    it('pre-fills with empty phone and metadata when not present', async () => {
      const api = await getApi()
      vi.mocked(api.get).mockResolvedValueOnce(
        makeCustomer({ phone: undefined, metadata: undefined }),
      )

      const wrapper = mountView()
      await waitUntil(() => findField(wrapper, 'Nome').props('modelValue') === 'João Silva')

      expect(findField(wrapper, 'Telefone').props('modelValue')).toBe('')
      expect(findField(wrapper, 'Metadata (JSON)').props('modelValue')).toBe('')
    })

    it('submits PUT to /v1/customers/:id with updated data', async () => {
      const api = await getApi()
      vi.mocked(api.get).mockResolvedValueOnce(makeCustomer())

      const wrapper = mountView()
      await waitUntil(() => findField(wrapper, 'Nome').props('modelValue') === 'João Silva')

      findField(wrapper, 'Nome').vm.$emit('update:modelValue', 'João Atualizado')
      await wrapper.vm.$nextTick()

      vi.mocked(api.put).mockResolvedValueOnce(makeCustomer({ name: 'João Atualizado' }))

      await wrapper.find('form').trigger('submit')
      await waitUntil(() => pushMock.mock.calls.length > 0)

      expect(api.put).toHaveBeenCalledWith('/v1/customers/cust-1', expect.objectContaining({
        name: 'João Atualizado',
        email: 'joao@test.com',
      }))
      expect(successMock).toHaveBeenCalledWith('Cliente atualizado com sucesso')
      expect(pushMock).toHaveBeenCalledWith('/customers/cust-1')
    })

    it('shows fetch error with retry button when getById fails', async () => {
      const api = await getApi()
      vi.mocked(api.get).mockRejectedValueOnce({
        message: 'Erro interno do servidor',
        status: 500,
      })

      const wrapper = mountView()
      await waitUntil(() => wrapper.text().includes('Erro interno do servidor'))

      expect(wrapper.text()).toContain('Erro interno do servidor')
      expect(wrapper.findAll('button').some((b) => b.text() === 'Tentar novamente')).toBe(true)
    })

    it('shows not found state with back link when getById returns 404', async () => {
      const api = await getApi()
      vi.mocked(api.get).mockRejectedValueOnce({
        message: 'Cliente não encontrado',
        status: 404,
      })

      const wrapper = mountView()
      await waitUntil(() => wrapper.text().includes('Cliente não encontrado'))

      expect(wrapper.text()).toContain('Cliente não encontrado')
      expect(wrapper.find('a').exists()).toBe(true)
      expect(wrapper.findAll('button').some((b) => b.text() === 'Tentar novamente')).toBe(false)
    })

    it('retries loading customer when retry button is clicked', async () => {
      const api = await getApi()
      vi.mocked(api.get).mockRejectedValueOnce({ message: 'Erro', status: 500 })

      const wrapper = mountView()
      await waitUntil(() => wrapper.findAll('button').some((b) => b.text() === 'Tentar novamente'))

      vi.mocked(api.get).mockResolvedValueOnce(makeCustomer())
      await findButton(wrapper, 'Tentar novamente').trigger('click')
      await waitUntil(() => findField(wrapper, 'Nome').props('modelValue') === 'João Silva')

      expect(wrapper.findAll('button').some((b) => b.text() === 'Tentar novamente')).toBe(false)
      expect(findField(wrapper, 'Nome').props('modelValue')).toBe('João Silva')
    })

    it('clears previous errors on new submit', async () => {
      const api = await getApi()
      vi.mocked(api.get).mockResolvedValueOnce(makeCustomer())

      const wrapper = mountView()
      await waitUntil(() => findField(wrapper, 'Nome').props('modelValue') === 'João Silva')

      // First submit fails with 409
      vi.mocked(api.put).mockRejectedValueOnce({
        message: 'Conflito de dados',
        status: 409,
      })
      await wrapper.find('form').trigger('submit')
      await waitUntil(() => wrapper.text().includes('Conflito de dados'))
      expect(wrapper.text()).toContain('Conflito de dados')

      // Second submit succeeds — error should clear
      vi.mocked(api.put).mockResolvedValueOnce(makeCustomer())
      await wrapper.find('form').trigger('submit')
      await waitUntil(() => pushMock.mock.calls.length > 0)
      expect(wrapper.text()).not.toContain('Conflito de dados')
    })
  })
})
