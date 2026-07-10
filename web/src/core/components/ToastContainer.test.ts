import { useToast } from '@/core/composables/useToast'
import { mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import { toast } from 'vue-sonner'
import ToastContainer from './ToastContainer.vue'

// vue-sonner applies toast state updates inside `nextTick(...)` callbacks
// registered *after* the toast is created, so a single `await nextTick()`
// in the test is not always enough to observe the rendered result — flush
// the microtask queue a few times to be safe.
async function flush(): Promise<void> {
  for (let i = 0; i < 3; i++) {
    await nextTick()
  }
}

function waitForAnimationFrame(): Promise<void> {
  return new Promise((resolve) => requestAnimationFrame(() => resolve()))
}

describe('ToastContainer', () => {
  let wrapper: VueWrapper

  beforeEach(() => {
    // Clear any leftover toasts from previous tests (useToast's internal
    // list, which drives the ids used by vue-sonner under the hood).
    const { toasts } = useToast()
    toasts.value = []
  })

  afterEach(() => {
    // vue-sonner's toast state is a module-level singleton shared across
    // every mounted <Toaster>. Unmounting tears down the previous
    // subscription so it doesn't interfere with the next test's instance.
    wrapper?.unmount()
  })

  it('renders no toasts when the list is empty', () => {
    wrapper = mount(ToastContainer)
    expect(wrapper.findAll('[data-sonner-toast]')).toHaveLength(0)
  })

  it('renders a toast for each entry raised through useToast', async () => {
    wrapper = mount(ToastContainer)
    const { success, error } = useToast()
    success('Success message')
    error('Error message')
    await flush()

    expect(wrapper.findAll('[data-sonner-toast]')).toHaveLength(2)
  })

  it('displays the toast message text', async () => {
    wrapper = mount(ToastContainer)
    const { success } = useToast()
    success('Hello world')
    await flush()

    expect(wrapper.find('[data-title]').text()).toBe('Hello world')
  })

  it('applies the correct data-type for each toast type', async () => {
    wrapper = mount(ToastContainer)
    const { success, error, warning } = useToast()
    success('ok')
    error('fail')
    warning('warn')
    await flush()

    const toasts = wrapper.findAll('[data-sonner-toast]')
    expect(toasts).toHaveLength(3)
    expect(toasts.map(t => t.attributes('data-type'))).toEqual(['warning', 'error', 'success'])
  })

  it('marks a toast as removed once it is dismissed', async () => {
    wrapper = mount(ToastContainer)
    const id = toast.success('Dismiss me')
    await flush()

    expect(wrapper.findAll('[data-sonner-toast]')).toHaveLength(1)

    toast.dismiss(id)
    await waitForAnimationFrame()
    await waitForAnimationFrame()
    await flush()

    expect(wrapper.find('[data-sonner-toast]').attributes('data-removed')).toBe('true')
  })

  it('renders the toaster in the top-right position', () => {
    wrapper = mount(ToastContainer)
    const container = wrapper.find('[data-sonner-toaster]')
    expect(container.exists()).toBe(true)
    expect(container.attributes('data-y-position')).toBe('top')
    expect(container.attributes('data-x-position')).toBe('right')
  })

  it('exposes the toast region as a polite live region for screen readers', () => {
    wrapper = mount(ToastContainer)
    expect(wrapper.find('[aria-live="polite"]').exists()).toBe(true)
  })
})
