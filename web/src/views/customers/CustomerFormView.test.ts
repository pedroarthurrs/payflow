import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'

// Mock vue-router
const pushMock = vi.fn()
const routeParams = { value: {} as Record<string, string> }
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: pushMock }),
  useRoute: () => ({ params: routeParams.value }),
}))

// Mock http
vi.mock('@/services/http', () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
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

import CustomerFormView from './CustomerFormView.vue'

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
    return mount(CustomerFormView)
  }

  describe('Create mode', () => {
    it('renders title "Novo Cliente" and empty form fields', () => {
      const wrapper = mountView()
      expect(wrapper.find('.customer-form__title').text()).toBe('Novo Cliente')
      expect(wrapper.find('button[type="submit"]').text()).toBe('Criar')
    })

    it('submits POST to /v1/customers with form data', async () => {
      const http = await getHttp()
      const created = makeCustomer({ id: 'new-1' })
      vi.mocked(http.post).mockResolvedValueOnce({ data: created })

      const wrapper = mountView()

      const fields = wrapper.findAllComponents({ name: 'FormField' })
      const nameField = fields.find((f) => f.props('label') === 'Nome')!
      const emailField = fields.find((f) => f.props('label') === 'Email')!
      const phoneField = fields.find((f) => f.props('label') === 'Telefone')!
      const docField = fields.find((f) => f.props('label') === 'Documento')!
      const docTypeField = fields.find((f) => f.props('label') === 'Tipo de Documento')!
      const metadataField = fields.find((f) => f.props('label') === 'Metadata (JSON)')!

      nameField.vm.$emit('update:modelValue', 'Maria')
      emailField.vm.$emit('update:modelValue', 'maria@test.com')
      phoneField.vm.$emit('update:modelValue', '11888888888')
      docField.vm.$emit('update:modelValue', '98765432100')
      docTypeField.vm.$emit('update:modelValue', 'CNPJ')
      metadataField.vm.$emit('update:modelValue', '{"foo":"bar"}')
      await wrapper.vm.$nextTick()

      await wrapper.find('form').trigger('submit')
      await flushPromises()

      expect(http.post).toHaveBeenCalledWith('/v1/customers', {
        name: 'Maria',
        email: 'maria@test.com',
        phone: '11888888888',
        document: '98765432100',
        document_type: 'CNPJ',
        metadata: { foo: 'bar' },
      })
      expect(successMock).toHaveBeenCalledWith('Cliente criado com sucesso')
      expect(pushMock).toHaveBeenCalledWith('/customers/new-1')
    })

    it('submits without metadata when textarea is empty', async () => {
      const http = await getHttp()
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeCustomer({ id: 'c-2' }) })

      const wrapper = mountView()

      const fields = wrapper.findAllComponents({ name: 'FormField' })
      fields.find((f) => f.props('label') === 'Nome')!.vm.$emit('update:modelValue', 'Test')
      fields.find((f) => f.props('label') === 'Email')!.vm.$emit('update:modelValue', 'test@t.com')
      fields.find((f) => f.props('label') === 'Documento')!.vm.$emit('update:modelValue', '111')
      await wrapper.vm.$nextTick()

      await wrapper.find('form').trigger('submit')
      await flushPromises()

      expect(http.post).toHaveBeenCalledWith('/v1/customers', expect.objectContaining({
        metadata: undefined,
      }))
    })

    it('shows validation error on invalid JSON metadata', async () => {
      const wrapper = mountView()

      const fields = wrapper.findAllComponents({ name: 'FormField' })
      fields.find((f) => f.props('label') === 'Nome')!.vm.$emit('update:modelValue', 'Test')
      fields.find((f) => f.props('label') === 'Email')!.vm.$emit('update:modelValue', 'a@b.com')
      fields.find((f) => f.props('label') === 'Documento')!.vm.$emit('update:modelValue', '111')
      fields.find((f) => f.props('label') === 'Metadata (JSON)')!.vm.$emit('update:modelValue', '{invalid}')
      await wrapper.vm.$nextTick()

      await wrapper.find('form').trigger('submit')
      await flushPromises()

      const metadataField = wrapper.findAllComponents({ name: 'FormField' })
        .find((f) => f.props('label') === 'Metadata (JSON)')!
      expect(metadataField.props('error')).toBe('JSON inválido')
    })

    it('displays field-level validation errors from 400 response', async () => {
      const http = await getHttp()
      vi.mocked(http.post).mockRejectedValueOnce({
        message: 'Validation failed',
        status: 400,
        validationErrors: {
          name: 'Nome é obrigatório',
          email: 'Email inválido',
        },
      })

      const wrapper = mountView()

      const fields = wrapper.findAllComponents({ name: 'FormField' })
      fields.find((f) => f.props('label') === 'Nome')!.vm.$emit('update:modelValue', '')
      fields.find((f) => f.props('label') === 'Email')!.vm.$emit('update:modelValue', 'bad')
      fields.find((f) => f.props('label') === 'Documento')!.vm.$emit('update:modelValue', '111')
      await wrapper.vm.$nextTick()

      await wrapper.find('form').trigger('submit')
      await flushPromises()

      const nameField = wrapper.findAllComponents({ name: 'FormField' })
        .find((f) => f.props('label') === 'Nome')!
      const emailField = wrapper.findAllComponents({ name: 'FormField' })
        .find((f) => f.props('label') === 'Email')!

      expect(nameField.props('error')).toBe('Nome é obrigatório')
      expect(emailField.props('error')).toBe('Email inválido')
    })

    it('displays 409 conflict error in the form', async () => {
      const http = await getHttp()
      vi.mocked(http.post).mockRejectedValueOnce({
        message: 'Cliente com este documento já existe',
        status: 409,
      })

      const wrapper = mountView()

      const fields = wrapper.findAllComponents({ name: 'FormField' })
      fields.find((f) => f.props('label') === 'Nome')!.vm.$emit('update:modelValue', 'Test')
      fields.find((f) => f.props('label') === 'Email')!.vm.$emit('update:modelValue', 'a@b.com')
      fields.find((f) => f.props('label') === 'Documento')!.vm.$emit('update:modelValue', '111')
      await wrapper.vm.$nextTick()

      await wrapper.find('form').trigger('submit')
      await flushPromises()

      expect(wrapper.find('.customer-form__error').text()).toBe('Cliente com este documento já existe')
    })

    it('shows generic error for other failures', async () => {
      const http = await getHttp()
      vi.mocked(http.post).mockRejectedValueOnce({
        message: 'Erro de conexão com o servidor',
        status: 0,
      })

      const wrapper = mountView()

      const fields = wrapper.findAllComponents({ name: 'FormField' })
      fields.find((f) => f.props('label') === 'Nome')!.vm.$emit('update:modelValue', 'Test')
      fields.find((f) => f.props('label') === 'Email')!.vm.$emit('update:modelValue', 'a@b.com')
      fields.find((f) => f.props('label') === 'Documento')!.vm.$emit('update:modelValue', '111')
      await wrapper.vm.$nextTick()

      await wrapper.find('form').trigger('submit')
      await flushPromises()

      expect(wrapper.find('.customer-form__error').text()).toBe('Erro de conexão com o servidor')
    })

    it('disables submit button and shows "Salvando..." while loading', async () => {
      const http = await getHttp()
      let resolvePost: (v: any) => void
      vi.mocked(http.post).mockReturnValueOnce(
        new Promise((resolve) => { resolvePost = resolve }) as any,
      )

      const wrapper = mountView()

      const fields = wrapper.findAllComponents({ name: 'FormField' })
      fields.find((f) => f.props('label') === 'Nome')!.vm.$emit('update:modelValue', 'Test')
      fields.find((f) => f.props('label') === 'Email')!.vm.$emit('update:modelValue', 'a@b.com')
      fields.find((f) => f.props('label') === 'Documento')!.vm.$emit('update:modelValue', '111')
      await wrapper.vm.$nextTick()

      await wrapper.find('form').trigger('submit')
      await wrapper.vm.$nextTick()

      const submitBtn = wrapper.find('.customer-form__submit-btn')
      expect(submitBtn.text()).toBe('Salvando...')
      expect(submitBtn.attributes('disabled')).toBeDefined()

      resolvePost!({ data: makeCustomer() })
      await flushPromises()
    })

    it('navigates to /customers when cancel is clicked', async () => {
      const wrapper = mountView()
      await wrapper.find('.customer-form__cancel-btn').trigger('click')
      expect(pushMock).toHaveBeenCalledWith('/customers')
    })
  })

  describe('Edit mode', () => {
    beforeEach(() => {
      routeParams.value = { id: 'cust-1' }
    })

    it('renders title "Editar Cliente" and fetches customer data', async () => {
      const http = await getHttp()
      vi.mocked(http.get).mockResolvedValueOnce({ data: makeCustomer() })

      const wrapper = mountView()
      await flushPromises()

      expect(wrapper.find('.customer-form__title').text()).toBe('Editar Cliente')
      expect(http.get).toHaveBeenCalledWith('/v1/customers/cust-1')
      expect(wrapper.find('button[type="submit"]').text()).toBe('Salvar')
    })

    it('pre-fills form fields with customer data', async () => {
      const http = await getHttp()
      vi.mocked(http.get).mockResolvedValueOnce({ data: makeCustomer() })

      const wrapper = mountView()
      await flushPromises()

      const fields = wrapper.findAllComponents({ name: 'FormField' })
      expect(fields.find((f) => f.props('label') === 'Nome')!.props('modelValue')).toBe('João Silva')
      expect(fields.find((f) => f.props('label') === 'Email')!.props('modelValue')).toBe('joao@test.com')
      expect(fields.find((f) => f.props('label') === 'Telefone')!.props('modelValue')).toBe('11999999999')
      expect(fields.find((f) => f.props('label') === 'Documento')!.props('modelValue')).toBe('12345678901')
      expect(fields.find((f) => f.props('label') === 'Tipo de Documento')!.props('modelValue')).toBe('CPF')
      expect(fields.find((f) => f.props('label') === 'Metadata (JSON)')!.props('modelValue')).toContain('"key"')
    })

    it('pre-fills with empty phone and metadata when not present', async () => {
      const http = await getHttp()
      vi.mocked(http.get).mockResolvedValueOnce({
        data: makeCustomer({ phone: undefined, metadata: undefined }),
      })

      const wrapper = mountView()
      await flushPromises()

      const fields = wrapper.findAllComponents({ name: 'FormField' })
      expect(fields.find((f) => f.props('label') === 'Telefone')!.props('modelValue')).toBe('')
      expect(fields.find((f) => f.props('label') === 'Metadata (JSON)')!.props('modelValue')).toBe('')
    })

    it('submits PUT to /v1/customers/:id with updated data', async () => {
      const http = await getHttp()
      vi.mocked(http.get).mockResolvedValueOnce({ data: makeCustomer() })

      const wrapper = mountView()
      await flushPromises()

      const fields = wrapper.findAllComponents({ name: 'FormField' })
      fields.find((f) => f.props('label') === 'Nome')!.vm.$emit('update:modelValue', 'João Atualizado')
      await wrapper.vm.$nextTick()

      vi.mocked(http.put).mockResolvedValueOnce({ data: makeCustomer({ name: 'João Atualizado' }) })

      await wrapper.find('form').trigger('submit')
      await flushPromises()

      expect(http.put).toHaveBeenCalledWith('/v1/customers/cust-1', expect.objectContaining({
        name: 'João Atualizado',
        email: 'joao@test.com',
      }))
      expect(successMock).toHaveBeenCalledWith('Cliente atualizado com sucesso')
      expect(pushMock).toHaveBeenCalledWith('/customers/cust-1')
    })

    it('shows fetch error with retry button when getById fails', async () => {
      const http = await getHttp()
      vi.mocked(http.get).mockRejectedValueOnce({
        message: 'Erro interno do servidor',
        status: 500,
      })

      const wrapper = mountView()
      await flushPromises()

      expect(wrapper.find('.customer-form__fetch-error').exists()).toBe(true)
      expect(wrapper.find('.customer-form__fetch-error-message').text()).toContain('Erro')
      expect(wrapper.find('.customer-form__retry-btn').exists()).toBe(true)
    })

    it('shows not found state with back link when getById returns 404', async () => {
      const http = await getHttp()
      vi.mocked(http.get).mockRejectedValueOnce({
        message: 'Cliente não encontrado',
        status: 404,
      })

      const wrapper = mountView()
      await flushPromises()

      expect(wrapper.find('.customer-form__not-found').exists()).toBe(true)
      expect(wrapper.find('.customer-form__not-found-message').text()).toBe('Cliente não encontrado')
      expect(wrapper.find('.customer-form__fetch-error').exists()).toBe(false)
    })

    it('retries loading customer when retry button is clicked', async () => {
      const http = await getHttp()
      vi.mocked(http.get).mockRejectedValueOnce({ message: 'Erro', status: 500 })

      const wrapper = mountView()
      await flushPromises()

      vi.mocked(http.get).mockResolvedValueOnce({ data: makeCustomer() })
      await wrapper.find('.customer-form__retry-btn').trigger('click')
      await flushPromises()

      expect(wrapper.find('.customer-form__fetch-error').exists()).toBe(false)
      const fields = wrapper.findAllComponents({ name: 'FormField' })
      expect(fields.find((f) => f.props('label') === 'Nome')!.props('modelValue')).toBe('João Silva')
    })

    it('clears previous errors on new submit', async () => {
      const http = await getHttp()
      vi.mocked(http.get).mockResolvedValueOnce({ data: makeCustomer() })

      const wrapper = mountView()
      await flushPromises()

      // First submit fails with 409
      vi.mocked(http.put).mockRejectedValueOnce({
        message: 'Conflito',
        status: 409,
      })
      await wrapper.find('form').trigger('submit')
      await flushPromises()
      expect(wrapper.find('.customer-form__error').exists()).toBe(true)

      // Second submit succeeds — error should clear
      vi.mocked(http.put).mockResolvedValueOnce({ data: makeCustomer() })
      await wrapper.find('form').trigger('submit')
      await wrapper.vm.$nextTick()
      expect(wrapper.find('.customer-form__error').exists()).toBe(false)
    })
  })
})
