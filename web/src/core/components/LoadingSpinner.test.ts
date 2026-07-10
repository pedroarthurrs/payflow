import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import LoadingSpinner from './LoadingSpinner.vue'

describe('LoadingSpinner', () => {
  it('has role="status" for accessibility', () => {
    const wrapper = mount(LoadingSpinner)
    expect(wrapper.find('[role="status"]').exists()).toBe(true)
  })

  it('has a screen-reader-only label with meaningful text', () => {
    const wrapper = mount(LoadingSpinner)
    const label = wrapper.find('.sr-only')
    expect(label.exists()).toBe(true)
    expect(label.text()).toBe('Carregando...')
  })

  it('renders a spinning icon', () => {
    const wrapper = mount(LoadingSpinner)
    const icon = wrapper.find('svg')
    expect(icon.exists()).toBe(true)
    expect(icon.classes()).toContain('animate-spin')
  })

  it('renders the md size class by default', () => {
    const wrapper = mount(LoadingSpinner)
    expect(wrapper.find('svg').classes()).toContain('size-8')
  })

  it('renders the sm size class when size="sm"', () => {
    const wrapper = mount(LoadingSpinner, { props: { size: 'sm' } })
    expect(wrapper.find('svg').classes()).toContain('size-4')
  })

  it('renders the lg size class when size="lg"', () => {
    const wrapper = mount(LoadingSpinner, { props: { size: 'lg' } })
    expect(wrapper.find('svg').classes()).toContain('size-12')
  })
})
