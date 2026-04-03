import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import ToastContainer from './ToastContainer.vue'
import { useToast } from '@/composables/useToast'

describe('ToastContainer', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    const { toasts } = useToast()
    toasts.value = []
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('renders no toasts when list is empty', () => {
    const wrapper = mount(ToastContainer)
    expect(wrapper.findAll('.toast')).toHaveLength(0)
  })

  it('renders a toast for each entry in the toast list', () => {
    const { success, error } = useToast()
    success('Success message')
    error('Error message')

    const wrapper = mount(ToastContainer)
    expect(wrapper.findAll('.toast')).toHaveLength(2)
  })

  it('displays the toast message text', () => {
    const { success } = useToast()
    success('Hello world')

    const wrapper = mount(ToastContainer)
    expect(wrapper.find('.toast__message').text()).toBe('Hello world')
  })

  it('applies the correct CSS class for each toast type', () => {
    const { success, error, warning } = useToast()
    success('ok')
    error('fail')
    warning('warn')

    const wrapper = mount(ToastContainer)
    const toasts = wrapper.findAll('.toast')

    expect(toasts[0].classes()).toContain('toast--success')
    expect(toasts[1].classes()).toContain('toast--error')
    expect(toasts[2].classes()).toContain('toast--warning')
  })

  it('removes a toast when the close button is clicked', async () => {
    const { success, toasts } = useToast()
    success('Dismiss me')

    const wrapper = mount(ToastContainer)
    expect(wrapper.findAll('.toast')).toHaveLength(1)

    await wrapper.find('.toast__close').trigger('click')

    expect(toasts.value).toHaveLength(0)
  })

  it('has a fixed position container at top-right', () => {
    const wrapper = mount(ToastContainer)
    const container = wrapper.find('.toast-container')
    expect(container.exists()).toBe(true)
  })

  it('close button has accessible aria-label', () => {
    const { success } = useToast()
    success('test')

    const wrapper = mount(ToastContainer)
    const btn = wrapper.find('.toast__close')
    expect(btn.attributes('aria-label')).toBeTruthy()
  })
})
