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

import UserFormView from './UserFormView.vue'

async function getHttp() {
  return (await import('@/services/http')).default
}

function makeUser(overrides: Record<string, unknown> = {}) {
  return {
    id: 'user-1',
    name: 'Carlos Admin',
    email: 'carlos@test.com',
    role: 'admin',
    created_at: '2024-01-15T10:30:00Z',
    ...overrides,
  }
}

describe('UserFormView', () => {
  beforeEach(() => {
    routeParams.value = {}
    vi.clearAllMocks()
  })

  function mountView() {
    return mount(UserFormView)
  }

  describe('Create mode', () => {
    it('renders title "Novo Usuário" and all create fields', () => {
      const wrapper = mountView()
      expect(wrapper.find('.user-form__title').text()).toBe('Novo Usuário')
      expect(wrapper.find('button[type="submit"]').text()).toBe('Criar')

      const fields = wrapper.findAllComponents({ name: 'FormField' })
      const labels = fields.map((f) => f.props('label'))
      expect(labels).toContain('Nome')
      expect(labels).toContain('Email')
      expect(labels).toContain('Senha')
      expect(labels).toContain('Role')
    })

    it('submits POST to /v1/users with form data', async () => {
      const http = await getHttp()
      const created = makeUser({ id: 'new-1' })
      vi.mocked(http.post).mockResolvedValueOnce({ data: created })

      const wrapper = mountView()

      const fields = wrapper.findAllComponents({ name: 'FormField' })
      fields.find((f) => f.props('label') === 'Nome')!.vm.$emit('update:modelValue', 'Ana Maria')
      fields.find((f) => f.props('label') === 'Email')!.vm.$emit('update:modelValue', 'ana@test.com')
      fields.find((f) => f.props('label') === 'Senha')!.vm.$emit('update:modelValue', 'senha12345')
      fields.find((f) => f.props('label') === 'Role')!.vm.$emit('update:modelValue', 'admin')
      await wrapper.vm.$nextTick()

      await wrapper.find('form').trigger('submit')
      await flushPromises()

      expect(http.post).toHaveBeenCalledWith('/v1/users', {
        name: 'Ana Maria',
        email: 'ana@test.com',
        password: 'senha12345',
        role: 'admin',
      })
      expect(successMock).toHaveBeenCalledWith('Usuário criado com sucesso')
      expect(pushMock).toHaveBeenCalledWith('/users')
    })

    it('displays field-level validation errors from 400 response', async () => {
      const http = await getHttp()
      vi.mocked(http.post).mockRejectedValueOnce({
        message: 'Validation failed',
        status: 400,
        validationErrors: {
          name: 'Nome é obrigatório',
          email: 'Email inválido',
          password: 'Senha deve ter no mínimo 8 caracteres',
        },
      })

      const wrapper = mountView()

      const fields = wrapper.findAllComponents({ name: 'FormField' })
      fields.find((f) => f.props('label') === 'Nome')!.vm.$emit('update:modelValue', '')
      fields.find((f) => f.props('label') === 'Email')!.vm.$emit('update:modelValue', 'bad')
      fields.find((f) => f.props('label') === 'Senha')!.vm.$emit('update:modelValue', '123')
      await wrapper.vm.$nextTick()

      await wrapper.find('form').trigger('submit')
      await flushPromises()

      const updatedFields = wrapper.findAllComponents({ name: 'FormField' })
      expect(updatedFields.find((f) => f.props('label') === 'Nome')!.props('error')).toBe('Nome é obrigatório')
      expect(updatedFields.find((f) => f.props('label') === 'Email')!.props('error')).toBe('Email inválido')
      expect(updatedFields.find((f) => f.props('label') === 'Senha')!.props('error')).toBe('Senha deve ter no mínimo 8 caracteres')
    })

    it('displays 403 error in the form', async () => {
      const http = await getHttp()
      vi.mocked(http.post).mockRejectedValueOnce({
        message: 'Apenas owners podem criar usuários',
        status: 403,
      })

      const wrapper = mountView()

      const fields = wrapper.findAllComponents({ name: 'FormField' })
      fields.find((f) => f.props('label') === 'Nome')!.vm.$emit('update:modelValue', 'Test')
      fields.find((f) => f.props('label') === 'Email')!.vm.$emit('update:modelValue', 'a@b.com')
      fields.find((f) => f.props('label') === 'Senha')!.vm.$emit('update:modelValue', '12345678')
      await wrapper.vm.$nextTick()

      await wrapper.find('form').trigger('submit')
      await flushPromises()

      expect(wrapper.find('.user-form__error').text()).toBe('Apenas owners podem criar usuários')
    })

    it('displays 409 conflict error in the form', async () => {
      const http = await getHttp()
      vi.mocked(http.post).mockRejectedValueOnce({
        message: 'Email já cadastrado',
        status: 409,
      })

      const wrapper = mountView()

      const fields = wrapper.findAllComponents({ name: 'FormField' })
      fields.find((f) => f.props('label') === 'Nome')!.vm.$emit('update:modelValue', 'Test')
      fields.find((f) => f.props('label') === 'Email')!.vm.$emit('update:modelValue', 'dup@test.com')
      fields.find((f) => f.props('label') === 'Senha')!.vm.$emit('update:modelValue', '12345678')
      await wrapper.vm.$nextTick()

      await wrapper.find('form').trigger('submit')
      await flushPromises()

      expect(wrapper.find('.user-form__error').text()).toBe('Email já cadastrado')
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
      fields.find((f) => f.props('label') === 'Senha')!.vm.$emit('update:modelValue', '12345678')
      await wrapper.vm.$nextTick()

      await wrapper.find('form').trigger('submit')
      await flushPromises()

      expect(wrapper.find('.user-form__error').text()).toBe('Erro de conexão com o servidor')
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
      fields.find((f) => f.props('label') === 'Senha')!.vm.$emit('update:modelValue', '12345678')
      await wrapper.vm.$nextTick()

      await wrapper.find('form').trigger('submit')
      await wrapper.vm.$nextTick()

      const submitBtn = wrapper.find('.user-form__submit-btn')
      expect(submitBtn.text()).toBe('Salvando...')
      expect(submitBtn.attributes('disabled')).toBeDefined()

      resolvePost!({ data: makeUser() })
      await flushPromises()
    })

    it('navigates to /users when cancel is clicked', async () => {
      const wrapper = mountView()
      await wrapper.find('.user-form__cancel-btn').trigger('click')
      expect(pushMock).toHaveBeenCalledWith('/users')
    })
  })

  describe('Edit mode', () => {
    beforeEach(() => {
      routeParams.value = { id: 'user-1' }
    })

    it('renders title "Editar Usuário" and fetches user data', async () => {
      const http = await getHttp()
      vi.mocked(http.get).mockResolvedValueOnce({ data: makeUser() })

      const wrapper = mountView()
      await flushPromises()

      expect(wrapper.find('.user-form__title').text()).toBe('Editar Usuário')
      expect(http.get).toHaveBeenCalledWith('/v1/users/user-1')
      expect(wrapper.find('button[type="submit"]').text()).toBe('Salvar')
    })

    it('pre-fills name and role, hides email and password fields', async () => {
      const http = await getHttp()
      vi.mocked(http.get).mockResolvedValueOnce({ data: makeUser() })

      const wrapper = mountView()
      await flushPromises()

      const fields = wrapper.findAllComponents({ name: 'FormField' })
      const labels = fields.map((f) => f.props('label'))
      expect(labels).toContain('Nome')
      expect(labels).toContain('Role')
      expect(labels).not.toContain('Email')
      expect(labels).not.toContain('Senha')

      expect(fields.find((f) => f.props('label') === 'Nome')!.props('modelValue')).toBe('Carlos Admin')
      expect(fields.find((f) => f.props('label') === 'Role')!.props('modelValue')).toBe('admin')
    })

    it('submits PUT to /v1/users/:id with name and role only', async () => {
      const http = await getHttp()
      vi.mocked(http.get).mockResolvedValueOnce({ data: makeUser() })

      const wrapper = mountView()
      await flushPromises()

      const fields = wrapper.findAllComponents({ name: 'FormField' })
      fields.find((f) => f.props('label') === 'Nome')!.vm.$emit('update:modelValue', 'Carlos Atualizado')
      fields.find((f) => f.props('label') === 'Role')!.vm.$emit('update:modelValue', 'member')
      await wrapper.vm.$nextTick()

      vi.mocked(http.put).mockResolvedValueOnce({ data: makeUser({ name: 'Carlos Atualizado', role: 'member' }) })

      await wrapper.find('form').trigger('submit')
      await flushPromises()

      expect(http.put).toHaveBeenCalledWith('/v1/users/user-1', {
        name: 'Carlos Atualizado',
        role: 'member',
      })
      expect(successMock).toHaveBeenCalledWith('Usuário atualizado com sucesso')
      expect(pushMock).toHaveBeenCalledWith('/users')
    })

    it('shows fetch error with retry button when getById fails', async () => {
      const http = await getHttp()
      vi.mocked(http.get).mockRejectedValueOnce({
        message: 'Erro interno do servidor',
        status: 500,
      })

      const wrapper = mountView()
      await flushPromises()

      expect(wrapper.find('.user-form__fetch-error').exists()).toBe(true)
      expect(wrapper.find('.user-form__fetch-error-message').text()).toContain('Erro')
      expect(wrapper.find('.user-form__retry-btn').exists()).toBe(true)
    })

    it('shows not found state with back link when getById returns 404', async () => {
      const http = await getHttp()
      vi.mocked(http.get).mockRejectedValueOnce({
        message: 'Usuário não encontrado',
        status: 404,
      })

      const wrapper = mountView()
      await flushPromises()

      expect(wrapper.find('.user-form__not-found').exists()).toBe(true)
      expect(wrapper.find('.user-form__not-found-message').text()).toBe('Usuário não encontrado')
      expect(wrapper.find('.user-form__fetch-error').exists()).toBe(false)
    })

    it('retries loading user when retry button is clicked', async () => {
      const http = await getHttp()
      vi.mocked(http.get).mockRejectedValueOnce({ message: 'Erro', status: 500 })

      const wrapper = mountView()
      await flushPromises()

      vi.mocked(http.get).mockResolvedValueOnce({ data: makeUser() })
      await wrapper.find('.user-form__retry-btn').trigger('click')
      await flushPromises()

      expect(wrapper.find('.user-form__fetch-error').exists()).toBe(false)
      const fields = wrapper.findAllComponents({ name: 'FormField' })
      expect(fields.find((f) => f.props('label') === 'Nome')!.props('modelValue')).toBe('Carlos Admin')
    })

    it('clears previous errors on new submit', async () => {
      const http = await getHttp()
      vi.mocked(http.get).mockResolvedValueOnce({ data: makeUser() })

      const wrapper = mountView()
      await flushPromises()

      // First submit fails with 403
      vi.mocked(http.put).mockRejectedValueOnce({
        message: 'Operação não permitida',
        status: 403,
      })
      await wrapper.find('form').trigger('submit')
      await flushPromises()
      expect(wrapper.find('.user-form__error').exists()).toBe(true)

      // Second submit succeeds — error should clear
      vi.mocked(http.put).mockResolvedValueOnce({ data: makeUser() })
      await wrapper.find('form').trigger('submit')
      await wrapper.vm.$nextTick()
      expect(wrapper.find('.user-form__error').exists()).toBe(false)
    })
  })
})
