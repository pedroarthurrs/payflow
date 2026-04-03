import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ConfirmDialog from './ConfirmDialog.vue'

describe('ConfirmDialog', () => {
  const baseProps = {
    visible: true,
    title: 'Excluir cliente',
    message: 'Tem certeza que deseja excluir?',
  }

  it('renders dialog when visible is true', () => {
    const wrapper = mount(ConfirmDialog, { props: baseProps })
    expect(wrapper.find('.confirm-dialog').exists()).toBe(true)
    expect(wrapper.find('.confirm-dialog__title').text()).toBe('Excluir cliente')
    expect(wrapper.find('.confirm-dialog__message').text()).toBe('Tem certeza que deseja excluir?')
  })

  it('does not render dialog when visible is false', () => {
    const wrapper = mount(ConfirmDialog, {
      props: { ...baseProps, visible: false },
    })
    expect(wrapper.find('.confirm-dialog').exists()).toBe(false)
  })

  it('renders default confirm label as Confirmar', () => {
    const wrapper = mount(ConfirmDialog, { props: baseProps })
    const confirmBtn = wrapper.find('.confirm-dialog__btn--danger')
    expect(confirmBtn.text()).toBe('Confirmar')
  })

  it('renders custom confirm label', () => {
    const wrapper = mount(ConfirmDialog, {
      props: { ...baseProps, confirmLabel: 'Excluir' },
    })
    const confirmBtn = wrapper.find('.confirm-dialog__btn--danger')
    expect(confirmBtn.text()).toBe('Excluir')
  })

  it('emits confirm when confirm button is clicked', async () => {
    const wrapper = mount(ConfirmDialog, { props: baseProps })
    await wrapper.find('.confirm-dialog__btn--danger').trigger('click')
    expect(wrapper.emitted('confirm')).toHaveLength(1)
  })

  it('emits cancel when cancel button is clicked', async () => {
    const wrapper = mount(ConfirmDialog, { props: baseProps })
    await wrapper.find('.confirm-dialog__btn--cancel').trigger('click')
    expect(wrapper.emitted('cancel')).toHaveLength(1)
  })

  it('emits cancel when backdrop is clicked', async () => {
    const wrapper = mount(ConfirmDialog, { props: baseProps })
    await wrapper.find('.confirm-dialog__overlay').trigger('click')
    expect(wrapper.emitted('cancel')).toHaveLength(1)
  })

  it('does not emit cancel when dialog box is clicked', async () => {
    const wrapper = mount(ConfirmDialog, { props: baseProps })
    await wrapper.find('.confirm-dialog').trigger('click')
    expect(wrapper.emitted('cancel')).toBeUndefined()
  })

  it('emits cancel on Escape key', async () => {
    const wrapper = mount(ConfirmDialog, { props: baseProps })
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect(wrapper.emitted('cancel')).toHaveLength(1)
    wrapper.unmount()
  })

  it('applies danger variant by default', () => {
    const wrapper = mount(ConfirmDialog, { props: baseProps })
    expect(wrapper.find('.confirm-dialog__btn--danger').exists()).toBe(true)
  })

  it('applies warning variant when specified', () => {
    const wrapper = mount(ConfirmDialog, {
      props: { ...baseProps, variant: 'warning' as const },
    })
    expect(wrapper.find('.confirm-dialog__btn--warning').exists()).toBe(true)
  })

  it('renders default slot content', () => {
    const wrapper = mount(ConfirmDialog, {
      props: baseProps,
      slots: { default: '<label><input type="checkbox" /> Hard delete</label>' },
    })
    expect(wrapper.find('input[type="checkbox"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('Hard delete')
  })

  it('renders overlay with semi-transparent backdrop', () => {
    const wrapper = mount(ConfirmDialog, { props: baseProps })
    expect(wrapper.find('.confirm-dialog__overlay').exists()).toBe(true)
  })

  it('has role dialog and aria-modal', () => {
    const wrapper = mount(ConfirmDialog, { props: baseProps })
    const dialog = wrapper.find('.confirm-dialog')
    expect(dialog.attributes('role')).toBe('dialog')
    expect(dialog.attributes('aria-modal')).toBe('true')
  })
})
