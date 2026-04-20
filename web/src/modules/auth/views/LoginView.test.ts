import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import LoginView from './LoginView.vue'

// Mock vue-router
const pushMock = vi.fn()
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: pushMock }),
}))

// Mock http
vi.mock('@/api/http', () => ({
  default: { post: vi.fn() },
}))

function fakeJwt(payload: Record<string, unknown>): string {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }))
  const body = btoa(JSON.stringify(payload))
  return `${header}.${body}.fake-signature`
}

function makeTokenPair() {
  return {
    access_token: fakeJwt({ sub: 'org-1', role: 'admin', auth_type: 'user' }),
    refresh_token: 'refresh-abc',
    access_token_expires_at: new Date().toISOString(),
    refresh_token_expires_at: new Date().toISOString(),
  }
}

describe('LoginView', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    vi.clearAllMocks()
  })

  function mountView() {
    return mount(LoginView, {
      global: {
        plugins: [],
      },
    })
  }

  it('renders brand name, email field, password field and submit button', () => {
    const wrapper = mountView()
    expect(wrapper.text()).toContain('i9pay')
    expect(wrapper.find('input[type="email"]').exists()).toBe(true)
    expect(wrapper.find('input[type="password"]').exists()).toBe(true)
    expect(wrapper.find('button[type="submit"]').exists()).toBe(true)
    expect(wrapper.find('button[type="submit"]').text()).toBe('Entrar')
  })

  it('calls login and redirects on success', async () => {
    const http = (await import('@/core/api/http')).default
    vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair() })

    const wrapper = mountView()
    await wrapper.find('input[type="email"]').setValue('user@test.com')
    await wrapper.find('input[type="password"]').setValue('password123')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(http.post).toHaveBeenCalledWith('/v1/auth/login', {
      email: 'user@test.com',
      password: 'password123',
    })
    expect(pushMock).toHaveBeenCalledWith('/dashboard')
  })

  it('shows API error message on 401', async () => {
    const http = (await import('@/core/api/http')).default
    vi.mocked(http.post).mockRejectedValueOnce({
      message: 'credenciais inválidas',
      status: 401,
    })

    const wrapper = mountView()
    await wrapper.find('input[type="email"]').setValue('bad@test.com')
    await wrapper.find('input[type="password"]').setValue('wrong')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(wrapper.find('.login-error').text()).toBe('credenciais inválidas')
  })

  it('shows rate limit message on 429', async () => {
    const http = (await import('@/core/api/http')).default
    vi.mocked(http.post).mockRejectedValueOnce({
      message: 'too many requests',
      status: 429,
    })

    const wrapper = mountView()
    await wrapper.find('input[type="email"]').setValue('user@test.com')
    await wrapper.find('input[type="password"]').setValue('pass')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(wrapper.find('.login-error').text()).toBe(
      'Limite de tentativas excedido. Aguarde antes de tentar novamente.',
    )
  })

  it('shows generic error for other failures', async () => {
    const http = (await import('@/core/api/http')).default
    vi.mocked(http.post).mockRejectedValueOnce({
      message: 'Erro de conexão com o servidor',
      status: 0,
    })

    const wrapper = mountView()
    await wrapper.find('input[type="email"]').setValue('user@test.com')
    await wrapper.find('input[type="password"]').setValue('pass')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(wrapper.find('.login-error').text()).toBe('Erro de conexão com o servidor')
  })

  it('disables button and shows loading text while submitting', async () => {
    const http = (await import('@/core/api/http')).default
    let resolveLogin: (v: any) => void
    vi.mocked(http.post).mockReturnValueOnce(
      new Promise((resolve) => {
        resolveLogin = resolve
      }),
    )

    const wrapper = mountView()
    await wrapper.find('input[type="email"]').setValue('user@test.com')
    await wrapper.find('input[type="password"]').setValue('pass')
    await wrapper.find('form').trigger('submit')
    await wrapper.vm.$nextTick()

    const button = wrapper.find('button[type="submit"]')
    expect(button.text()).toBe('Entrando...')
    expect(button.attributes('disabled')).toBeDefined()

    resolveLogin!({ data: makeTokenPair() })
    await flushPromises()

    expect(button.text()).toBe('Entrar')
    expect(button.attributes('disabled')).toBeUndefined()
  })

  it('clears error message on new submit', async () => {
    const http = (await import('@/core/api/http')).default
    vi.mocked(http.post).mockRejectedValueOnce({
      message: 'credenciais inválidas',
      status: 401,
    })

    const wrapper = mountView()
    await wrapper.find('input[type="email"]').setValue('bad@test.com')
    await wrapper.find('input[type="password"]').setValue('wrong')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(wrapper.find('.login-error').exists()).toBe(true)

    // Second submit — error should clear before new attempt
    vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair() })
    await wrapper.find('form').trigger('submit')
    await wrapper.vm.$nextTick()

    // During loading, old error should be gone
    expect(wrapper.find('.login-error').exists()).toBe(false)
  })

  it('shows email required error when email is empty', async () => {
    const wrapper = mountView()
    await wrapper.find('input[type="password"]').setValue('mypassword')
    await wrapper.find('form').trigger('submit')
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.form-field__error').text()).toBe('Email é obrigatório.')
  })

  it('shows invalid email error when email format is wrong', async () => {
    const wrapper = mountView()
    await wrapper.find('input[type="email"]').setValue('not-an-email')
    await wrapper.find('input[type="password"]').setValue('mypassword')
    await wrapper.find('form').trigger('submit')
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.form-field__error').text()).toBe('Informe um email válido.')
  })

  it('shows password required error when password is empty', async () => {
    const wrapper = mountView()
    await wrapper.find('input[type="email"]').setValue('user@test.com')
    await wrapper.find('form').trigger('submit')
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.form-field__error').text()).toBe('Senha é obrigatória.')
  })

  it('does not call API when validation fails', async () => {
    const http = (await import('@/core/api/http')).default

    const wrapper = mountView()
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(http.post).not.toHaveBeenCalled()
  })

  it('renders password toggle button', () => {
    const wrapper = mountView()
    expect(wrapper.find('button[aria-label="Mostrar senha"]').exists()).toBe(true)
  })

  it('toggles password visibility on button click', async () => {
    const wrapper = mountView()

    expect(wrapper.find('input[type="password"]').exists()).toBe(true)

    await wrapper.find('button[aria-label="Mostrar senha"]').trigger('click')
    await wrapper.vm.$nextTick()

    expect(wrapper.find('input[type="text"]').exists()).toBe(true)
    expect(wrapper.find('button[aria-label="Ocultar senha"]').exists()).toBe(true)
  })

  it('clears email error dynamically when a valid email is typed after submit', async () => {
    const wrapper = mountView()
    await wrapper.find('input[type="email"]').setValue('not-an-email')
    await wrapper.find('input[type="password"]').setValue('mypassword')
    await wrapper.find('form').trigger('submit')
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.form-field__error').exists()).toBe(true)

    await wrapper.find('input[type="email"]').setValue('valid@test.com')
    await flushPromises()

    expect(wrapper.find('.form-field__error').exists()).toBe(false)
  })

  it('does not show field errors before first submit attempt', async () => {
    const wrapper = mountView()

    await wrapper.find('input[type="email"]').setValue('bad-email')
    await flushPromises()

    expect(wrapper.find('.form-field__error').exists()).toBe(false)
  })
})
