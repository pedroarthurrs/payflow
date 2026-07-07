import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import AppLayout from './AppLayout.vue'

const mockHasRole = vi.fn()
const mockLogout = vi.fn()

vi.mock('@/core/composables/useAuth', () => ({
  useAuth: () => ({
    hasRole: mockHasRole,
    logout: mockLogout,
    isAuthenticated: { value: true },
    login: vi.fn(),
  }),
}))

function createTestRouter(initialRoute = '/dashboard') {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/dashboard', component: { template: '<div>Dashboard</div>' } },
      { path: '/customers', component: { template: '<div>Customers</div>' } },
      { path: '/users', component: { template: '<div>Users</div>' } },
    ],
  })
}

async function mountLayout(options: { route?: string; isAdmin?: boolean } = {}) {
  const { route = '/dashboard', isAdmin = false } = options
  const router = createTestRouter(route)
  await router.push(route)
  await router.isReady()

  mockHasRole.mockImplementation((role: string) => {
    if (role === 'admin') return isAdmin
    return false
  })

  const pinia = createPinia()
  setActivePinia(pinia)

  return mount(AppLayout, {
    global: {
      plugins: [router, pinia],
    },
    slots: {
      default: '<div class="test-content">Page Content</div>',
    },
  })
}

describe('AppLayout', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders sidebar with navigation links', async () => {
    const wrapper = await mountLayout()
    expect(wrapper.find('.sidebar').exists()).toBe(true)
    expect(wrapper.text()).toContain('Dashboard')
    expect(wrapper.text()).toContain('Clientes')
    expect(wrapper.text()).toContain('Logout')
  })

  it('renders header with brand name', async () => {
    const wrapper = await mountLayout()
    expect(wrapper.find('.header__brand').text()).toBe('payflow')
  })

  it('renders slot content in main area', async () => {
    const wrapper = await mountLayout()
    expect(wrapper.find('.test-content').exists()).toBe(true)
    expect(wrapper.find('.test-content').text()).toBe('Page Content')
  })

  it('renders ToastContainer', async () => {
    const wrapper = await mountLayout()
    expect(wrapper.findComponent({ name: 'ToastContainer' }).exists()).toBe(true)
  })

  it('shows Usuários link when user has admin role', async () => {
    const wrapper = await mountLayout({ isAdmin: true })
    const links = wrapper.findAll('.sidebar__link')
    const texts = links.map((l) => l.text())
    expect(texts).toContain('Usuários')
  })

  it('hides Usuários link when user does not have admin role', async () => {
    const wrapper = await mountLayout({ isAdmin: false })
    const links = wrapper.findAll('.sidebar__link')
    const texts = links.map((l) => l.text())
    expect(texts).not.toContain('Usuários')
  })

  it('highlights active route in sidebar', async () => {
    const wrapper = await mountLayout({ route: '/dashboard' })
    const dashboardLink = wrapper.findAll('.sidebar__link').find((l) => l.text() === 'Dashboard')
    expect(dashboardLink?.classes()).toContain('sidebar__link--active')
  })

  it('does not highlight inactive routes', async () => {
    const wrapper = await mountLayout({ route: '/dashboard' })
    const customersLink = wrapper.findAll('.sidebar__link').find((l) => l.text() === 'Clientes')
    expect(customersLink?.classes()).not.toContain('sidebar__link--active')
  })

  it('calls logout when logout button is clicked', async () => {
    const wrapper = await mountLayout()
    const logoutBtn = wrapper.find('.sidebar__logout')
    await logoutBtn.trigger('click')
    expect(mockLogout).toHaveBeenCalledOnce()
  })

  it('toggles sidebar open state on hamburger click', async () => {
    const wrapper = await mountLayout()
    const hamburger = wrapper.find('.hamburger')
    expect(wrapper.find('.sidebar--open').exists()).toBe(false)
    await hamburger.trigger('click')
    expect(wrapper.find('.sidebar--open').exists()).toBe(true)
  })

  it('closes sidebar when overlay is clicked', async () => {
    const wrapper = await mountLayout()
    await wrapper.find('.hamburger').trigger('click')
    expect(wrapper.find('.sidebar--open').exists()).toBe(true)
    await wrapper.find('.sidebar-overlay').trigger('click')
    expect(wrapper.find('.sidebar--open').exists()).toBe(false)
  })

  it('closes sidebar when a nav link is clicked', async () => {
    const wrapper = await mountLayout()
    await wrapper.find('.hamburger').trigger('click')
    expect(wrapper.find('.sidebar--open').exists()).toBe(true)
    const dashboardLink = wrapper.findAll('.sidebar__link').find((l) => l.text() === 'Dashboard')
    await dashboardLink?.trigger('click')
    expect(wrapper.find('.sidebar--open').exists()).toBe(false)
  })
})
