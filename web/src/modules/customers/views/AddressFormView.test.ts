import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

// Mock vue-router
const pushMock = vi.fn()
const routeParams = { value: {} as Record<string, string> }
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: pushMock }),
  useRoute: () => ({ params: routeParams.value }),
}))

// Mock http
vi.mock('@/api/http', () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
  },
}))

// Mock useToast
const successMock = vi.fn()
const errorMock = vi.fn()
vi.mock('@/core/composables/useToast', () => ({
  useToast: () => ({
    success: successMock,
    error: errorMock,
  }),
}))

import AddressFormView from './AddressFormView.vue'

async function getHttp() {
  return (await import('@/core/api/http')).default
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
    return mount(AddressFormView)
  }

  describe('Create mode', () => {
    it('renders title "Novo Endereço" and empty form fields', () => {
      const wrapper = mountView()
      expect(wrapper.find('.address-form__title').text()).toBe('Novo Endereço')
      expect(wrapper.find('button[type="submit"]').text()).toBe('Criar')
    })

    it('defaults country to "BR" and type to "shipping"', () => {
      const wrapper = mountView()
      const fields = wrapper.findAllComponents({ name: 'FormField' })
      expect(fields.find((f) => f.props('label') === 'País')!.props('modelValue')).toBe('BR')
      expect(fields.find((f) => f.props('label') === 'Tipo')!.props('modelValue')).toBe('shipping')
    })

    it('submits POST to /v1/customers/:id/addresses with form data', async () => {
      const http = await getHttp()
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeAddress({ id: 'new-addr' }) })

      const wrapper = mountView()

      const fields = wrapper.findAllComponents({ name: 'FormField' })
      fields.find((f) => f.props('label') === 'Tipo')!.vm.$emit('update:modelValue', 'billing')
      fields.find((f) => f.props('label') === 'CEP')!.vm.$emit('update:modelValue', '01001-000')
      fields.find((f) => f.props('label') === 'Logradouro')!.vm.$emit('update:modelValue', 'Rua Nova')
      fields.find((f) => f.props('label') === 'Número')!.vm.$emit('update:modelValue', '456')
      fields.find((f) => f.props('label') === 'Complemento')!.vm.$emit('update:modelValue', 'Sala 2')
      fields.find((f) => f.props('label') === 'Bairro')!.vm.$emit('update:modelValue', 'Centro')
      fields.find((f) => f.props('label') === 'Cidade')!.vm.$emit('update:modelValue', 'Rio de Janeiro')
      fields.find((f) => f.props('label') === 'Estado')!.vm.$emit('update:modelValue', 'RJ')
      fields.find((f) => f.props('label') === 'Endereço principal')!.vm.$emit('update:modelValue', true)
      await wrapper.vm.$nextTick()

      await wrapper.find('form').trigger('submit')
      await flushPromises()

      expect(http.post).toHaveBeenCalledWith('/v1/customers/cust-1/addresses', {
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
      const http = await getHttp()
      vi.mocked(http.post).mockResolvedValueOnce({ data: makeAddress() })

      const wrapper = mountView()

      const fields = wrapper.findAllComponents({ name: 'FormField' })
      fields.find((f) => f.props('label') === 'CEP')!.vm.$emit('update:modelValue', '01001-000')
      fields.find((f) => f.props('label') === 'Logradouro')!.vm.$emit('update:modelValue', 'Rua X')
      fields.find((f) => f.props('label') === 'Cidade')!.vm.$emit('update:modelValue', 'SP')
      fields.find((f) => f.props('label') === 'Estado')!.vm.$emit('update:modelValue', 'SP')
      await wrapper.vm.$nextTick()

      await wrapper.find('form').trigger('submit')
      await flushPromises()

      expect(http.post).toHaveBeenCalledWith('/v1/customers/cust-1/addresses', expect.objectContaining({
        number: undefined,
        complement: undefined,
        neighborhood: undefined,
      }))
    })

    it('displays field-level validation errors from 400 response', async () => {
      const http = await getHttp()
      vi.mocked(http.post).mockRejectedValueOnce({
        message: 'Validation failed',
        status: 400,
        validationErrors: {
          postal_code: 'CEP é obrigatório',
          street: 'Logradouro é obrigatório',
        },
      })

      const wrapper = mountView()

      const fields = wrapper.findAllComponents({ name: 'FormField' })
      fields.find((f) => f.props('label') === 'CEP')!.vm.$emit('update:modelValue', '')
      fields.find((f) => f.props('label') === 'Logradouro')!.vm.$emit('update:modelValue', '')
      fields.find((f) => f.props('label') === 'Cidade')!.vm.$emit('update:modelValue', 'SP')
      fields.find((f) => f.props('label') === 'Estado')!.vm.$emit('update:modelValue', 'SP')
      await wrapper.vm.$nextTick()

      await wrapper.find('form').trigger('submit')
      await flushPromises()

      const updatedFields = wrapper.findAllComponents({ name: 'FormField' })
      expect(updatedFields.find((f) => f.props('label') === 'CEP')!.props('error')).toBe('CEP é obrigatório')
      expect(updatedFields.find((f) => f.props('label') === 'Logradouro')!.props('error')).toBe('Logradouro é obrigatório')
    })

    it('shows generic error for non-validation failures', async () => {
      const http = await getHttp()
      vi.mocked(http.post).mockRejectedValueOnce({
        message: 'Erro de conexão com o servidor',
        status: 0,
      })

      const wrapper = mountView()

      const fields = wrapper.findAllComponents({ name: 'FormField' })
      fields.find((f) => f.props('label') === 'CEP')!.vm.$emit('update:modelValue', '01001-000')
      fields.find((f) => f.props('label') === 'Logradouro')!.vm.$emit('update:modelValue', 'Rua X')
      fields.find((f) => f.props('label') === 'Cidade')!.vm.$emit('update:modelValue', 'SP')
      fields.find((f) => f.props('label') === 'Estado')!.vm.$emit('update:modelValue', 'SP')
      await wrapper.vm.$nextTick()

      await wrapper.find('form').trigger('submit')
      await flushPromises()

      expect(wrapper.find('.address-form__error').text()).toBe('Erro de conexão com o servidor')
    })

    it('disables submit button and shows "Salvando..." while loading', async () => {
      const http = await getHttp()
      let resolvePost: (v: any) => void
      vi.mocked(http.post).mockReturnValueOnce(
        new Promise((resolve) => { resolvePost = resolve }) as any,
      )

      const wrapper = mountView()

      const fields = wrapper.findAllComponents({ name: 'FormField' })
      fields.find((f) => f.props('label') === 'CEP')!.vm.$emit('update:modelValue', '01001-000')
      fields.find((f) => f.props('label') === 'Logradouro')!.vm.$emit('update:modelValue', 'Rua X')
      fields.find((f) => f.props('label') === 'Cidade')!.vm.$emit('update:modelValue', 'SP')
      fields.find((f) => f.props('label') === 'Estado')!.vm.$emit('update:modelValue', 'SP')
      await wrapper.vm.$nextTick()

      await wrapper.find('form').trigger('submit')
      await wrapper.vm.$nextTick()

      const submitBtn = wrapper.find('.address-form__submit-btn')
      expect(submitBtn.text()).toBe('Salvando...')
      expect(submitBtn.attributes('disabled')).toBeDefined()

      resolvePost!({ data: makeAddress() })
      await flushPromises()
    })

    it('navigates to customer detail when cancel is clicked', async () => {
      const wrapper = mountView()
      await wrapper.find('.address-form__cancel-btn').trigger('click')
      expect(pushMock).toHaveBeenCalledWith('/customers/cust-1')
    })
  })

  describe('Edit mode', () => {
    beforeEach(() => {
      routeParams.value = { id: 'cust-1', addressId: 'addr-1' }
    })

    it('renders title "Editar Endereço" and fetches address data', async () => {
      const http = await getHttp()
      vi.mocked(http.get).mockResolvedValueOnce({ data: makeAddress() })

      const wrapper = mountView()
      await flushPromises()

      expect(wrapper.find('.address-form__title').text()).toBe('Editar Endereço')
      expect(http.get).toHaveBeenCalledWith('/v1/customers/cust-1/addresses/addr-1')
      expect(wrapper.find('button[type="submit"]').text()).toBe('Salvar')
    })

    it('pre-fills form fields with address data', async () => {
      const http = await getHttp()
      vi.mocked(http.get).mockResolvedValueOnce({ data: makeAddress() })

      const wrapper = mountView()
      await flushPromises()

      const fields = wrapper.findAllComponents({ name: 'FormField' })
      expect(fields.find((f) => f.props('label') === 'Tipo')!.props('modelValue')).toBe('shipping')
      expect(fields.find((f) => f.props('label') === 'CEP')!.props('modelValue')).toBe('01001-000')
      expect(fields.find((f) => f.props('label') === 'Logradouro')!.props('modelValue')).toBe('Rua Teste')
      expect(fields.find((f) => f.props('label') === 'Número')!.props('modelValue')).toBe('123')
      expect(fields.find((f) => f.props('label') === 'Complemento')!.props('modelValue')).toBe('Apto 4')
      expect(fields.find((f) => f.props('label') === 'Bairro')!.props('modelValue')).toBe('Centro')
      expect(fields.find((f) => f.props('label') === 'Cidade')!.props('modelValue')).toBe('São Paulo')
      expect(fields.find((f) => f.props('label') === 'Estado')!.props('modelValue')).toBe('SP')
      expect(fields.find((f) => f.props('label') === 'País')!.props('modelValue')).toBe('BR')
      expect(fields.find((f) => f.props('label') === 'Endereço principal')!.props('modelValue')).toBe(true)
    })

    it('pre-fills with empty optional fields when not present', async () => {
      const http = await getHttp()
      vi.mocked(http.get).mockResolvedValueOnce({
        data: makeAddress({ number: undefined, complement: undefined, neighborhood: undefined }),
      })

      const wrapper = mountView()
      await flushPromises()

      const fields = wrapper.findAllComponents({ name: 'FormField' })
      expect(fields.find((f) => f.props('label') === 'Número')!.props('modelValue')).toBe('')
      expect(fields.find((f) => f.props('label') === 'Complemento')!.props('modelValue')).toBe('')
      expect(fields.find((f) => f.props('label') === 'Bairro')!.props('modelValue')).toBe('')
    })

    it('submits PUT to /v1/customers/:id/addresses/:addressId with updated data', async () => {
      const http = await getHttp()
      vi.mocked(http.get).mockResolvedValueOnce({ data: makeAddress() })

      const wrapper = mountView()
      await flushPromises()

      const fields = wrapper.findAllComponents({ name: 'FormField' })
      fields.find((f) => f.props('label') === 'Logradouro')!.vm.$emit('update:modelValue', 'Rua Atualizada')
      await wrapper.vm.$nextTick()

      vi.mocked(http.put).mockResolvedValueOnce({ data: makeAddress({ street: 'Rua Atualizada' }) })

      await wrapper.find('form').trigger('submit')
      await flushPromises()

      expect(http.put).toHaveBeenCalledWith('/v1/customers/cust-1/addresses/addr-1', expect.objectContaining({
        street: 'Rua Atualizada',
        postal_code: '01001-000',
      }))
      expect(successMock).toHaveBeenCalledWith('Endereço atualizado com sucesso')
      expect(pushMock).toHaveBeenCalledWith('/customers/cust-1')
    })

    it('shows fetch error with retry button when getById fails', async () => {
      const http = await getHttp()
      vi.mocked(http.get).mockRejectedValueOnce({
        message: 'Erro interno',
        status: 500,
      })

      const wrapper = mountView()
      await flushPromises()

      expect(wrapper.find('.address-form__fetch-error').exists()).toBe(true)
      expect(wrapper.find('.address-form__fetch-error-message').text()).toContain('Erro')
      expect(wrapper.find('.address-form__retry-btn').exists()).toBe(true)
    })

    it('shows not found state with back link when getById returns 404', async () => {
      const http = await getHttp()
      vi.mocked(http.get).mockRejectedValueOnce({
        message: 'Endereço não encontrado',
        status: 404,
      })

      const wrapper = mountView()
      await flushPromises()

      expect(wrapper.find('.address-form__not-found').exists()).toBe(true)
      expect(wrapper.find('.address-form__not-found-message').text()).toBe('Endereço não encontrado')
      expect(wrapper.find('.address-form__fetch-error').exists()).toBe(false)
    })

    it('retries loading address when retry button is clicked', async () => {
      const http = await getHttp()
      vi.mocked(http.get).mockRejectedValueOnce({ message: 'Erro', status: 500 })

      const wrapper = mountView()
      await flushPromises()

      vi.mocked(http.get).mockResolvedValueOnce({ data: makeAddress() })
      await wrapper.find('.address-form__retry-btn').trigger('click')
      await flushPromises()

      expect(wrapper.find('.address-form__fetch-error').exists()).toBe(false)
      const fields = wrapper.findAllComponents({ name: 'FormField' })
      expect(fields.find((f) => f.props('label') === 'Logradouro')!.props('modelValue')).toBe('Rua Teste')
    })
  })
})
