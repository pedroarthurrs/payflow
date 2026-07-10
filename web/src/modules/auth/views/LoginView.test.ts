import { mount, type VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import LoginView from './LoginView.vue'

// Mock vue-router — useAuth navigates on successful login.
const pushMock = vi.fn()
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: pushMock }),
}))

// Mock the HTTP client the auth store depends on (default export, `.post`).
vi.mock('@/core/api/http', () => ({
  default: { post: vi.fn() },
}))

// Mock the RBAC store — login triggers fetchPermissions as a side effect,
// which is out of scope for these tests.
vi.mock('@/core/rbac', () => ({
  useRBACStore: () => ({
    hasRole: vi.fn(),
    can: vi.fn(),
    canAny: vi.fn(),
    canAll: vi.fn(),
    fetchPermissions: vi.fn().mockResolvedValue(undefined),
    $reset: vi.fn(),
  }),
}))

function fakeJwt(payload: Record<string, unknown>): string {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }))
  const body = btoa(JSON.stringify(payload))
  return `${header}.${body}.fake-signature`
}

function makeTokenPair() {
  return {
    access_token: fakeJwt({ sub: 'org-1', role: 'admin', auth_type: 'user', member_id: 'member-1' }),
    refresh_token: 'refresh-abc',
    access_token_expires_at: new Date().toISOString(),
    refresh_token_expires_at: new Date().toISOString(),
  }
}

async function getHttp() {
  return (await import('@/core/api/http')).default
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

function fieldError(wrapper: VueWrapper<any>) {
  return wrapper.find('.form-field__error')
}

describe('LoginView', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  function mountView() {
    return mount(LoginView)
  }

  it('renders brand name, email field, password field and submit button', () => {
    const wrapper = mountView()
    expect(wrapper.text()).toContain('payflow')
    expect(wrapper.find('input[type="email"]').exists()).toBe(true)
    expect(wrapper.find('input[type="password"]').exists()).toBe(true)
    expect(wrapper.find('button[type="submit"]').exists()).toBe(true)
    expect(wrapper.find('button[type="submit"]').text()).toBe('Entrar')
  })

  it('calls login and redirects on success', async () => {
    const http = await getHttp()
    vi.mocked(http.post).mockResolvedValueOnce({ data: makeTokenPair() })

    const wrapper = mountView()
    await wrapper.find('input[type="email"]').setValue('user@test.com')
    await wrapper.find('input[type="password"]').setValue('password123')
    await wrapper.find('form').trigger('submit')
    await waitUntil(() => pushMock.mock.calls.length > 0)

    expect(http.post).toHaveBeenCalledWith('/v1/auth/login', {
      email: 'user@test.com',
      password: 'password123',
    })
    expect(pushMock).toHaveBeenCalledWith('/dashboard')
  })

  // NOTE (real bug found in src/modules/auth/store/auth.ts `login()`):
  // the store re-wraps any rejected error into `new AuthError(err instanceof
  // Error ? err.message : 'Login failed', 'LOGIN_ERROR')`. The HTTP client's
  // `normalizeError()` rejects with a *plain* `{ message, status }` object,
  // not an `Error` instance, so `err instanceof Error` is always false for
  // real API failures. The original status/message are discarded and the
  // thrown AuthError always carries the literal message "Login failed" with
  // no `status` at all. As a result LoginView's 401/429/generic branches
  // (which key off `apiError.status`) are unreachable in practice — every
  // failed login shows the same "Login failed" text. These tests assert the
  // current (buggy) behavior rather than the never-hit branches.
  it('shows an error message when login fails (401)', async () => {
    const http = await getHttp()
    vi.mocked(http.post).mockRejectedValueOnce({
      message: 'credenciais inválidas',
      status: 401,
    })

    const wrapper = mountView()
    await wrapper.find('input[type="email"]').setValue('bad@test.com')
    await wrapper.find('input[type="password"]').setValue('wrongpass')
    await wrapper.find('form').trigger('submit')
    await waitUntil(() => wrapper.text().includes('Login failed'))

    expect(wrapper.text()).toContain('Login failed')
  })

  it('shows an error message when login fails (429)', async () => {
    const http = await getHttp()
    vi.mocked(http.post).mockRejectedValueOnce({
      message: 'too many requests',
      status: 429,
    })

    const wrapper = mountView()
    await wrapper.find('input[type="email"]').setValue('user@test.com')
    await wrapper.find('input[type="password"]').setValue('somepass')
    await wrapper.find('form').trigger('submit')
    await waitUntil(() => wrapper.text().includes('Login failed'))

    expect(wrapper.text()).toContain('Login failed')
  })

  it('shows an error message for other failures', async () => {
    const http = await getHttp()
    vi.mocked(http.post).mockRejectedValueOnce({
      message: 'Erro de conexão com o servidor',
      status: 0,
    })

    const wrapper = mountView()
    await wrapper.find('input[type="email"]').setValue('user@test.com')
    await wrapper.find('input[type="password"]').setValue('somepass')
    await wrapper.find('form').trigger('submit')
    await waitUntil(() => wrapper.text().includes('Login failed'))

    expect(wrapper.text()).toContain('Login failed')
  })

  it('disables button and shows loading text while submitting', async () => {
    const http = await getHttp()
    let resolveLogin: (v: any) => void
    vi.mocked(http.post).mockReturnValueOnce(
      new Promise((resolve) => {
        resolveLogin = resolve
      }),
    )

    const wrapper = mountView()
    await wrapper.find('input[type="email"]').setValue('user@test.com')
    await wrapper.find('input[type="password"]').setValue('somepass')
    await wrapper.find('form').trigger('submit')
    await waitUntil(() => wrapper.find('button[type="submit"]').text() === 'Entrando...')

    const button = wrapper.find('button[type="submit"]')
    expect(button.text()).toBe('Entrando...')
    expect(button.attributes('disabled')).toBeDefined()

    resolveLogin!({ data: makeTokenPair() })
    await waitUntil(() => wrapper.find('button[type="submit"]').text() === 'Entrar')

    expect(button.text()).toBe('Entrar')
    expect(button.attributes('disabled')).toBeUndefined()
  })

  it('clears error message on new submit', async () => {
    const http = await getHttp()
    vi.mocked(http.post).mockRejectedValueOnce({
      message: 'credenciais inválidas',
      status: 401,
    })

    const wrapper = mountView()
    await wrapper.find('input[type="email"]').setValue('bad@test.com')
    await wrapper.find('input[type="password"]').setValue('wrongpass')
    await wrapper.find('form').trigger('submit')
    await waitUntil(() => wrapper.text().includes('Login failed'))

    expect(wrapper.text()).toContain('Login failed')

    // Second submit — error should clear as soon as loading starts, before
    // the new attempt resolves.
    let resolveLogin: (v: any) => void
    vi.mocked(http.post).mockReturnValueOnce(
      new Promise((resolve) => {
        resolveLogin = resolve
      }),
    )
    await wrapper.find('form').trigger('submit')
    await waitUntil(() => wrapper.find('button[type="submit"]').text() === 'Entrando...')

    expect(wrapper.text()).not.toContain('Login failed')

    resolveLogin!({ data: makeTokenPair() })
  })

  it('shows a validation error when email is left empty', async () => {
    const wrapper = mountView()
    await wrapper.find('input[type="password"]').setValue('mypassword')
    await wrapper.find('form').trigger('submit')
    await waitUntil(() => fieldError(wrapper).exists())

    // The field was never touched (value stays `undefined`), so zod's base
    // string-type check fails before the custom email refinement runs.
    expect(fieldError(wrapper).text()).toBe('Required')
  })

  it('shows a validation error when email format is wrong', async () => {
    const wrapper = mountView()
    await wrapper.find('input[type="email"]').setValue('not-an-email')
    await wrapper.find('input[type="password"]').setValue('mypassword')
    await wrapper.find('form').trigger('submit')
    await waitUntil(() => fieldError(wrapper).exists())

    expect(fieldError(wrapper).text()).toBe('E-mail inválido')
  })

  it('shows a validation error when password is too short', async () => {
    const wrapper = mountView()
    await wrapper.find('input[type="email"]').setValue('user@test.com')
    await wrapper.find('input[type="password"]').setValue('short')
    await wrapper.find('form').trigger('submit')
    await waitUntil(() => fieldError(wrapper).exists())

    expect(fieldError(wrapper).text()).toBe('Senha deve ter no mínimo 8 caracteres')
  })

  it('does not call the API when validation fails', async () => {
    const http = await getHttp()

    const wrapper = mountView()
    await wrapper.find('form').trigger('submit')
    await waitUntil(() => fieldError(wrapper).exists())

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

  it('clears the email validation error once a valid email is typed after submit', async () => {
    const wrapper = mountView()
    await wrapper.find('input[type="email"]').setValue('not-an-email')
    await wrapper.find('input[type="password"]').setValue('mypassword')
    await wrapper.find('form').trigger('submit')
    await waitUntil(() => fieldError(wrapper).exists())

    expect(fieldError(wrapper).exists()).toBe(true)

    await wrapper.find('input[type="email"]').setValue('valid@test.com')
    await waitUntil(() => !fieldError(wrapper).exists())

    expect(fieldError(wrapper).exists()).toBe(false)
  })
})
