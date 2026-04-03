import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import DataTable from './DataTable.vue'
import type { Column } from './DataTable.vue'

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

  it('renders rows from data prop', () => {
    const wrapper = mount(DataTable, { props: { columns, data } })
    const rows = wrapper.findAll('.data-row')
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
    const cell = wrapper.find('.data-row td')
    expect(cell.text()).toBe('ACTIVE')
  })

  it('renders null/undefined cell values as empty string', () => {
    const cols: Column[] = [{ key: 'name', label: 'Nome' }]
    const rows = [{ name: null }, { name: undefined }]
    const wrapper = mount(DataTable, { props: { columns: cols, data: rows } })
    const cells = wrapper.findAll('.data-row td:first-child')
    expect(cells[0].text()).toBe('')
    expect(cells[1].text()).toBe('')
  })

  it('shows LoadingSpinner when loading is true', () => {
    const wrapper = mount(DataTable, { props: { columns, data: [], loading: true } })
    expect(wrapper.find('.loading-cell').exists()).toBe(true)
    expect(wrapper.findComponent({ name: 'LoadingSpinner' }).exists()).toBe(true)
    expect(wrapper.find('.empty-cell').exists()).toBe(false)
  })

  it('shows empty message when data is empty and not loading', () => {
    const wrapper = mount(DataTable, { props: { columns, data: [] } })
    expect(wrapper.find('.empty-cell').exists()).toBe(true)
    expect(wrapper.find('.empty-cell').text()).toBe('Nenhum registro encontrado')
    expect(wrapper.find('.loading-cell').exists()).toBe(false)
  })

  it('emits row-click when a row is clicked', async () => {
    const wrapper = mount(DataTable, { props: { columns, data } })
    await wrapper.find('.data-row').trigger('click')
    expect(wrapper.emitted('row-click')).toBeTruthy()
    expect(wrapper.emitted('row-click')![0]).toEqual([data[0]])
  })

  it('emits edit when edit button is clicked', async () => {
    const wrapper = mount(DataTable, { props: { columns, data } })
    await wrapper.find('.btn-edit').trigger('click')
    expect(wrapper.emitted('edit')).toBeTruthy()
    expect(wrapper.emitted('edit')![0]).toEqual([data[0]])
  })

  it('emits delete when delete button is clicked', async () => {
    const wrapper = mount(DataTable, { props: { columns, data } })
    await wrapper.find('.btn-delete').trigger('click')
    expect(wrapper.emitted('delete')).toBeTruthy()
    expect(wrapper.emitted('delete')![0]).toEqual([data[0]])
  })

  it('does not emit row-click when edit button is clicked', async () => {
    const wrapper = mount(DataTable, { props: { columns, data } })
    await wrapper.find('.btn-edit').trigger('click')
    expect(wrapper.emitted('row-click')).toBeFalsy()
  })

  it('does not emit row-click when delete button is clicked', async () => {
    const wrapper = mount(DataTable, { props: { columns, data } })
    await wrapper.find('.btn-delete').trigger('click')
    expect(wrapper.emitted('row-click')).toBeFalsy()
  })

  it('defaults loading to false', () => {
    const wrapper = mount(DataTable, { props: { columns, data } })
    expect(wrapper.findAll('.data-row')).toHaveLength(2)
    expect(wrapper.find('.loading-cell').exists()).toBe(false)
  })
})
