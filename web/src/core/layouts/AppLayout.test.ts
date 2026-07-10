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

function createTestRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/dashboard', component: { template: '<div>Dashboard</div>' } },
      { path: '/customers', component: { template: '<div>Customers</div>' } },
      { path: '/members', component: { template: '<div>Members</div>' } },
    ],
  })
}

async function mountLayout(options: { route?: string; isAdmin?: boolean } = {}) {
  const { route = '/dashboard', isAdmin = false } = options
  const router = createTestRouter()
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

// The sidebar <aside> is the element carrying the translate transition classes.
function sidebar(wrapper: Awaited<ReturnType<typeof mountLayout>>) {
  return wrapper.find('aside')
}

// A router-link renders as an <a>; nav links are the ones inside the <nav>.
function navLinks(wrapper: Awaited<ReturnType<typeof mountLayout>>) {
  return wrapper.findAll('nav a')
}

describe('AppLayout', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders the sidebar with brand and navigation', async () => {
    const wrapper = await mountLayout()
    expect(sidebar(wrapper).exists()).toBe(true)
    expect(wrapper.text()).toContain('payflow')
    expect(wrapper.text()).toContain('Dashboard')
    expect(wrapper.text()).toContain('Clientes')
    expect(wrapper.text()).toContain('Logout')
  })

  it('renders slot content in the main area', async () => {
    const wrapper = await mountLayout()
    expect(wrapper.find('.test-content').exists()).toBe(true)
    expect(wrapper.find('main .test-content').text()).toBe('Page Content')
  })

  it('renders ToastContainer', async () => {
    const wrapper = await mountLayout()
    expect(wrapper.findComponent({ name: 'ToastContainer' }).exists()).toBe(true)
  })

  it('shows the Members link when the user has the admin role', async () => {
    const wrapper = await mountLayout({ isAdmin: true })
    const texts = navLinks(wrapper).map((l) => l.text())
    expect(texts).toContain('Members')
  })

  it('hides the Members link when the user lacks the admin role', async () => {
    const wrapper = await mountLayout({ isAdmin: false })
    const texts = navLinks(wrapper).map((l) => l.text())
    expect(texts).not.toContain('Members')
  })

  it('highlights the active route link', async () => {
    const wrapper = await mountLayout({ route: '/dashboard' })
    const dashboardLink = navLinks(wrapper).find((l) => l.text() === 'Dashboard')
    expect(dashboardLink?.classes()).toContain('bg-sidebar-accent')
  })

  it('does not highlight inactive route links', async () => {
    const wrapper = await mountLayout({ route: '/dashboard' })
    const customersLink = navLinks(wrapper).find((l) => l.text() === 'Clientes')
    expect(customersLink?.classes()).not.toContain('bg-sidebar-accent')
  })

  it('calls logout when the logout button is clicked', async () => {
    const wrapper = await mountLayout()
    const logoutBtn = wrapper.findAll('button').find((b) => b.text().includes('Logout'))
    expect(logoutBtn).toBeTruthy()
    await logoutBtn!.trigger('click')
    expect(mockLogout).toHaveBeenCalledOnce()
  })

  it('toggles the mobile sidebar open on the hamburger click', async () => {
    const wrapper = await mountLayout()
    // Closed by default: the sidebar is translated off-screen.
    expect(sidebar(wrapper).classes()).toContain('-translate-x-full')

    await wrapper.find('[aria-label="Abrir menu"]').trigger('click')

    expect(sidebar(wrapper).classes()).toContain('translate-x-0')
    expect(sidebar(wrapper).classes()).not.toContain('-translate-x-full')
  })

  it('closes the sidebar when the overlay is clicked', async () => {
    const wrapper = await mountLayout()
    await wrapper.find('[aria-label="Abrir menu"]').trigger('click')
    expect(sidebar(wrapper).classes()).toContain('translate-x-0')

    // The overlay only renders while the sidebar is open.
    const overlay = wrapper.findAll('div').find((d) => d.classes().includes('bg-black/40'))
    expect(overlay).toBeTruthy()
    await overlay!.trigger('click')

    expect(sidebar(wrapper).classes()).toContain('-translate-x-full')
  })

  it('closes the sidebar when a nav link is clicked', async () => {
    const wrapper = await mountLayout()
    await wrapper.find('[aria-label="Abrir menu"]').trigger('click')
    expect(sidebar(wrapper).classes()).toContain('translate-x-0')

    const dashboardLink = navLinks(wrapper).find((l) => l.text() === 'Dashboard')
    await dashboardLink?.trigger('click')

    expect(sidebar(wrapper).classes()).toContain('-translate-x-full')
  })
})
