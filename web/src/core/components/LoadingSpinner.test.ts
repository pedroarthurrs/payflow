import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import LoadingSpinner from './LoadingSpinner.vue'

describe('LoadingSpinner', () => {
  it('renders with default md size', () => {
    const wrapper = mount(LoadingSpinner)
    expect(wrapper.find('.spinner--md').exists()).toBe(true)
  })

  it('renders with sm size when prop is passed', () => {
    const wrapper = mount(LoadingSpinner, { props: { size: 'sm' } })
    expect(wrapper.find('.spinner--sm').exists()).toBe(true)
  })

  it('renders with lg size when prop is passed', () => {
    const wrapper = mount(LoadingSpinner, { props: { size: 'lg' } })
    expect(wrapper.find('.spinner--lg').exists()).toBe(true)
  })

  it('has role="status" for accessibility', () => {
    const wrapper = mount(LoadingSpinner)
    expect(wrapper.find('[role="status"]').exists()).toBe(true)
  })

  it('has a screen-reader-only label', () => {
    const wrapper = mount(LoadingSpinner)
    expect(wrapper.find('.sr-only').exists()).toBe(true)
    expect(wrapper.find('.sr-only').text()).toBeTruthy()
  })

  it('contains a spinner element', () => {
    const wrapper = mount(LoadingSpinner)
    expect(wrapper.find('.spinner').exists()).toBe(true)
  })
})
