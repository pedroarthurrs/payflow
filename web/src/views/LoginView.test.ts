import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import LoginView from './LoginView.vue'

// Mock vue-router
const pushMock = vi.fn()
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: pushMock }),
}))

// Mock http
vi.mock('@/services/http', () => ({
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
    const http = (await import('@/services/http')).default
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
    const http = (await import('@/services/http')).default
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
    const http = (await import('@/services/http')).default
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
    const http = (await import('@/services/http')).default
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
    const http = (await import('@/services/http')).default
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
    const http = (await import('@/services/http')).default
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
})
