import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { Column } from './DataTable.vue'
import DataTable from './DataTable.vue'

const columns: Column[] = [
  { key: 'name', label: 'Nome' },
  { key: 'email', label: 'Email' },
]

const data = [
  { name: 'Alice', email: 'alice@example.com' },
  { name: 'Bob', email: 'bob@example.com' },
]

describe('DataTable', () => {
  it('renders column headers from columns prop', () => {
    const wrapper = mount(DataTable, { props: { columns, data } })
    const headers = wrapper.findAll('th')
    expect(headers[0].text()).toBe('Nome')
    expect(headers[1].text()).toBe('Email')
    expect(headers[2].text()).toBe('Ações')
  })

  it('does not render an actions header when showActions is false', () => {
    const wrapper = mount(DataTable, { props: { columns, data, showActions: false } })
    const headers = wrapper.findAll('th')
    expect(headers).toHaveLength(2)
    expect(wrapper.text()).not.toContain('Ações')
  })

  it('renders rows from data prop', () => {
    const wrapper = mount(DataTable, { props: { columns, data } })
    const rows = wrapper.findAll('tbody tr')
    expect(rows).toHaveLength(2)
    const cells = rows[0].findAll('td')
    expect(cells[0].text()).toBe('Alice')
    expect(cells[1].text()).toBe('alice@example.com')
  })

  it('applies format function when provided', () => {
    const cols: Column[] = [
      { key: 'status', label: 'Status', format: (v: string) => v.toUpperCase() },
    ]
    const rows = [{ status: 'active' }]
    const wrapper = mount(DataTable, { props: { columns: cols, data: rows } })
    const cell = wrapper.find('tbody tr td')
    expect(cell.text()).toBe('ACTIVE')
  })

  it('passes the full row to the format function', () => {
    const cols: Column[] = [
      {
        key: 'first',
        label: 'Full name',
        format: (_v, row) => `${row.first} ${row.last}`,
      },
    ]
    const rows = [{ first: 'Ada', last: 'Lovelace' }]
    const wrapper = mount(DataTable, { props: { columns: cols, data: rows } })
    expect(wrapper.find('tbody tr td').text()).toBe('Ada Lovelace')
  })

  it('renders null/undefined cell values as empty string', () => {
    const cols: Column[] = [{ key: 'name', label: 'Nome' }]
    const rows = [{ name: null }, { name: undefined }]
    const wrapper = mount(DataTable, { props: { columns: cols, data: rows } })
    const cells = wrapper.findAll('tbody tr td:first-child')
    expect(cells[0].text()).toBe('')
    expect(cells[1].text()).toBe('')
  })

  it('shows LoadingSpinner when loading is true', () => {
    const wrapper = mount(DataTable, { props: { columns, data: [], loading: true } })
    expect(wrapper.findComponent({ name: 'LoadingSpinner' }).exists()).toBe(true)
    expect(wrapper.text()).not.toContain('Nenhum registro encontrado')
  })

  it('shows empty message when data is empty and not loading', () => {
    const wrapper = mount(DataTable, { props: { columns, data: [] } })
    expect(wrapper.text()).toContain('Nenhum registro encontrado')
    expect(wrapper.findComponent({ name: 'LoadingSpinner' }).exists()).toBe(false)
  })

  it('emits row-click when a row is clicked', async () => {
    const wrapper = mount(DataTable, { props: { columns, data } })
    await wrapper.find('tbody tr').trigger('click')
    expect(wrapper.emitted('row-click')).toBeTruthy()
    expect(wrapper.emitted('row-click')![0]).toEqual([data[0]])
  })

  it('emits edit when the edit button is clicked', async () => {
    const wrapper = mount(DataTable, { props: { columns, data } })
    await wrapper.find('[aria-label="Editar"]').trigger('click')
    expect(wrapper.emitted('edit')).toBeTruthy()
    expect(wrapper.emitted('edit')![0]).toEqual([data[0]])
  })

  it('emits delete when the delete button is clicked', async () => {
    const wrapper = mount(DataTable, { props: { columns, data } })
    await wrapper.find('[aria-label="Excluir"]').trigger('click')
    expect(wrapper.emitted('delete')).toBeTruthy()
    expect(wrapper.emitted('delete')![0]).toEqual([data[0]])
  })

  it('does not emit row-click when the edit button is clicked', async () => {
    const wrapper = mount(DataTable, { props: { columns, data } })
    await wrapper.find('[aria-label="Editar"]').trigger('click')
    expect(wrapper.emitted('row-click')).toBeFalsy()
  })

  it('does not emit row-click when the delete button is clicked', async () => {
    const wrapper = mount(DataTable, { props: { columns, data } })
    await wrapper.find('[aria-label="Excluir"]').trigger('click')
    expect(wrapper.emitted('row-click')).toBeFalsy()
  })

  it('does not render action buttons when showActions is false', () => {
    const wrapper = mount(DataTable, { props: { columns, data, showActions: false } })
    expect(wrapper.find('[aria-label="Editar"]').exists()).toBe(false)
    expect(wrapper.find('[aria-label="Excluir"]').exists()).toBe(false)
  })

  it('defaults loading to false', () => {
    const wrapper = mount(DataTable, { props: { columns, data } })
    expect(wrapper.findAll('tbody tr')).toHaveLength(2)
    expect(wrapper.findComponent({ name: 'LoadingSpinner' }).exists()).toBe(false)
  })
})
