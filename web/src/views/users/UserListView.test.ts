import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'

// Mock vue-router
const pushMock = vi.fn()
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: pushMock }),
}))

// Mock http (used by useCrud internally)
vi.mock('@/services/http', () => ({
  default: { get: vi.fn(), delete: vi.fn() },
}))

// Mock useAuth
let mockHasRole: (role: string) => boolean = () => true
vi.mock('@/composables/useAuth', () => ({
  useAuth: () => ({
    hasRole: (role: string) => mockHasRole(role),
    isAuthenticated: { value: true },
  }),
}))

import UserListView from './UserListView.vue'

async function getHttp() {
  return (await import('@/services/http')).default
}

function makeUser(overrides: Record<string, unknown> = {}) {
  return {
    id: 'user-1',
    name: 'Maria Owner',
    email: 'maria@test.com',
    role: 'admin',
    created_at: '2024-03-10T14:00:00Z',
    ...overrides,
  }
}

describe('UserListView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockHasRole = () => true // default: owner
  })

  it('renders page title "Usuários"', async () => {
    const http = await getHttp()
    vi.mocked(http.get).mockResolvedValueOnce({ data: [] })

    const wrapper = mount(UserListView)
    await flushPromises()

    expect(wrapper.find('.user-list__title').text()).toBe('Usuários')
  })

  it('shows "Novo Usuário" button for owner', async () => {
    mockHasRole = (role: string) => role === 'owner'
    const http = await getHttp()
    vi.mocked(http.get).mockResolvedValueOnce({ data: [] })

    const wrapper = mount(UserListView)
    await flushPromises()

    expect(wrapper.find('.user-list__new-btn').exists()).toBe(true)
    expect(wrapper.find('.user-list__new-btn').text()).toBe('Novo Usuário')
  })

  it('hides "Novo Usuário" button for admin (non-owner)', async () => {
    mockHasRole = (role: string) => role !== 'owner'
    const http = await getHttp()
    vi.mocked(http.get).mockResolvedValueOnce({ data: [] })

    const wrapper = mount(UserListView)
    await flushPromises()

    expect(wrapper.find('.user-list__new-btn').exists()).toBe(false)
  })

  it('fetches users from /v1/users and renders them in DataTable', async () => {
    const http = await getHttp()
    const users = [
      makeUser({ id: 'u-1', name: 'Alice' }),
      makeUser({ id: 'u-2', name: 'Bob' }),
    ]
    vi.mocked(http.get).mockResolvedValueOnce({ data: users })

    const wrapper = mount(UserListView)
    await flushPromises()

    expect(http.get).toHaveBeenCalledWith('/v1/users')

    const dataTable = wrapper.findComponent({ name: 'DataTable' })
    expect(dataTable.props('data')).toHaveLength(2)
    expect(dataTable.props('loading')).toBe(false)
  })

  it('defines correct columns: nome, email, role, data de criação', async () => {
    const http = await getHttp()
    vi.mocked(http.get).mockResolvedValueOnce({ data: [] })

    const wrapper = mount(UserListView)
    await flushPromises()

    const dataTable = wrapper.findComponent({ name: 'DataTable' })
    const columns = dataTable.props('columns')

    expect(columns).toHaveLength(4)
    expect(columns[0]).toMatchObject({ key: 'name', label: 'Nome' })
    expect(columns[1]).toMatchObject({ key: 'email', label: 'Email' })
    expect(columns[2]).toMatchObject({ key: 'role', label: 'Role' })
    expect(columns[3]).toMatchObject({ key: 'created_at', label: 'Data de Criação' })
  })

  it('formats created_at column as localized date', async () => {
    const http = await getHttp()
    vi.mocked(http.get).mockResolvedValueOnce({ data: [] })

    const wrapper = mount(UserListView)
    await flushPromises()

    const dataTable = wrapper.findComponent({ name: 'DataTable' })
    const columns = dataTable.props('columns')
    const createdAtCol = columns.find((c: any) => c.key === 'created_at')

    expect(createdAtCol.format).toBeDefined()
    const formatted = createdAtCol.format('2024-03-10T14:00:00Z')
    expect(formatted).toBe(new Date('2024-03-10T14:00:00Z').toLocaleDateString('pt-BR'))
  })

  it('formats empty created_at as empty string', async () => {
    const http = await getHttp()
    vi.mocked(http.get).mockResolvedValueOnce({ data: [] })

    const wrapper = mount(UserListView)
    await flushPromises()

    const dataTable = wrapper.findComponent({ name: 'DataTable' })
    const columns = dataTable.props('columns')
    const createdAtCol = columns.find((c: any) => c.key === 'created_at')

    expect(createdAtCol.format('')).toBe('')
    expect(createdAtCol.format(null)).toBe('')
    expect(createdAtCol.format(undefined)).toBe('')
  })

  it('shows actions in DataTable for owner', async () => {
    mockHasRole = (role: string) => role === 'owner'
    const http = await getHttp()
    vi.mocked(http.get).mockResolvedValueOnce({ data: [] })

    const wrapper = mount(UserListView)
    await flushPromises()

    const dataTable = wrapper.findComponent({ name: 'DataTable' })
    expect(dataTable.props('showActions')).toBe(true)
  })

  it('hides actions in DataTable for admin (non-owner)', async () => {
    mockHasRole = (role: string) => role !== 'owner'
    const http = await getHttp()
    vi.mocked(http.get).mockResolvedValueOnce({ data: [] })

    const wrapper = mount(UserListView)
    await flushPromises()

    const dataTable = wrapper.findComponent({ name: 'DataTable' })
    expect(dataTable.props('showActions')).toBe(false)
  })

  it('navigates to /users/new when "Novo Usuário" is clicked', async () => {
    mockHasRole = (role: string) => role === 'owner'
    const http = await getHttp()
    vi.mocked(http.get).mockResolvedValueOnce({ data: [] })

    const wrapper = mount(UserListView)
    await flushPromises()

    await wrapper.find('.user-list__new-btn').trigger('click')
    expect(pushMock).toHaveBeenCalledWith('/users/new')
  })

  it('navigates to /users/:id/edit on edit event', async () => {
    mockHasRole = (role: string) => role === 'owner'
    const http = await getHttp()
    const user = makeUser({ id: 'user-42' })
    vi.mocked(http.get).mockResolvedValueOnce({ data: [user] })

    const wrapper = mount(UserListView)
    await flushPromises()

    const dataTable = wrapper.findComponent({ name: 'DataTable' })
    dataTable.vm.$emit('edit', user)
    await wrapper.vm.$nextTick()

    expect(pushMock).toHaveBeenCalledWith('/users/user-42/edit')
  })

  it('shows ConfirmDialog on delete event', async () => {
    mockHasRole = (role: string) => role === 'owner'
    const http = await getHttp()
    const user = makeUser({ id: 'user-99', name: 'Carlos' })
    vi.mocked(http.get).mockResolvedValueOnce({ data: [user] })

    const wrapper = mount(UserListView)
    await flushPromises()

    const dataTable = wrapper.findComponent({ name: 'DataTable' })
    dataTable.vm.$emit('delete', user)
    await wrapper.vm.$nextTick()

    const dialog = wrapper.findComponent({ name: 'ConfirmDialog' })
    expect(dialog.props('visible')).toBe(true)
    expect(dialog.props('title')).toBe('Excluir Usuário')
    expect(dialog.props('message')).toContain('Carlos')
  })

  it('deletes user and refreshes list on confirm', async () => {
    mockHasRole = (role: string) => role === 'owner'
    const http = await getHttp()
    const user = makeUser({ id: 'user-99' })
    vi.mocked(http.get).mockResolvedValueOnce({ data: [user] })

    const wrapper = mount(UserListView)
    await flushPromises()

    // Trigger delete
    const dataTable = wrapper.findComponent({ name: 'DataTable' })
    dataTable.vm.$emit('delete', user)
    await wrapper.vm.$nextTick()

    // Mock delete and re-fetch
    vi.mocked(http.delete).mockResolvedValueOnce({} as any)
    vi.mocked(http.get).mockResolvedValueOnce({ data: [] })

    const dialog = wrapper.findComponent({ name: 'ConfirmDialog' })
    dialog.vm.$emit('confirm')
    await flushPromises()

    expect(http.delete).toHaveBeenCalledWith('/v1/users/user-99')
    // Dialog should be closed
    expect(dialog.props('visible')).toBe(false)
  })

  it('closes ConfirmDialog on cancel', async () => {
    mockHasRole = (role: string) => role === 'owner'
    const http = await getHttp()
    const user = makeUser({ id: 'user-99' })
    vi.mocked(http.get).mockResolvedValueOnce({ data: [user] })

    const wrapper = mount(UserListView)
    await flushPromises()

    const dataTable = wrapper.findComponent({ name: 'DataTable' })
    dataTable.vm.$emit('delete', user)
    await wrapper.vm.$nextTick()

    const dialog = wrapper.findComponent({ name: 'ConfirmDialog' })
    expect(dialog.props('visible')).toBe(true)

    dialog.vm.$emit('cancel')
    await wrapper.vm.$nextTick()

    expect(dialog.props('visible')).toBe(false)
  })

  it('shows error message and retry button on fetch failure', async () => {
    const http = await getHttp()
    vi.mocked(http.get).mockRejectedValueOnce({ message: 'Erro de conexão com o servidor' })

    const wrapper = mount(UserListView)
    await flushPromises()

    expect(wrapper.find('.user-list__error').exists()).toBe(true)
    expect(wrapper.find('.user-list__error-message').text()).toBe('Erro de conexão com o servidor')
    expect(wrapper.find('.user-list__retry-btn').exists()).toBe(true)
  })

  it('retries fetch when retry button is clicked', async () => {
    const http = await getHttp()
    vi.mocked(http.get).mockRejectedValueOnce({ message: 'Erro' })

    const wrapper = mount(UserListView)
    await flushPromises()

    expect(wrapper.find('.user-list__error').exists()).toBe(true)

    vi.mocked(http.get).mockResolvedValueOnce({ data: [makeUser()] })
    await wrapper.find('.user-list__retry-btn').trigger('click')
    await flushPromises()

    expect(wrapper.find('.user-list__error').exists()).toBe(false)
    const dataTable = wrapper.findComponent({ name: 'DataTable' })
    expect(dataTable.props('data')).toHaveLength(1)
  })

  it('shows loading state via DataTable while fetching', async () => {
    const http = await getHttp()
    let resolve: (v: any) => void
    vi.mocked(http.get).mockReturnValue(new Promise((r) => { resolve = r }) as any)

    const wrapper = mount(UserListView)
    await wrapper.vm.$nextTick()

    const dataTable = wrapper.findComponent({ name: 'DataTable' })
    expect(dataTable.exists()).toBe(true)
    expect(dataTable.props('loading')).toBe(true)

    resolve!({ data: [] })
  })
})
