import { mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import ConfirmDialog from './ConfirmDialog.vue'

// ConfirmDialog wraps reka-ui's Dialog primitive, which teleports its content
// (overlay + panel) to document.body and only mounts it once `visible` flips
// to true. jsdom lacks a few pointer APIs the primitive relies on for
// outside-click dismissal, so we polyfill them for this file only.
;(Element.prototype as any).hasPointerCapture ??= () => false
;(Element.prototype as any).releasePointerCapture ??= () => {}
;(Element.prototype as any).setPointerCapture ??= () => {}
;(globalThis as any).PointerEvent ??= class PointerEvent extends MouseEvent {}

const baseProps = {
  visible: true,
  title: 'Excluir cliente',
  message: 'Tem certeza que deseja excluir?',
}

let wrapper: VueWrapper<any> | undefined

afterEach(() => {
  wrapper?.unmount()
  document.body.innerHTML = ''
})

function mountDialog(props: Record<string, unknown> = {}, slots: Record<string, string> = {}) {
  wrapper = mount(ConfirmDialog, {
    props: { ...baseProps, ...props },
    slots,
    attachTo: document.body,
  })
  return wrapper
}

describe('ConfirmDialog', () => {
  it('renders the dialog with title and message when visible', async () => {
    mountDialog()
    await nextTick()

    const dialog = document.querySelector('[role="dialog"]')
    expect(dialog).not.toBeNull()
    expect(dialog?.textContent).toContain('Excluir cliente')
    expect(dialog?.textContent).toContain('Tem certeza que deseja excluir?')
  })

  it('does not render the dialog when visible is false', async () => {
    mountDialog({ visible: false })
    await nextTick()

    expect(document.querySelector('[role="dialog"]')).toBeNull()
  })

  it('links the dialog to its title and description for accessibility', async () => {
    mountDialog()
    await nextTick()

    const dialog = document.querySelector('[role="dialog"]') as HTMLElement
    const labelledBy = dialog.getAttribute('aria-labelledby')
    const describedBy = dialog.getAttribute('aria-describedby')
    expect(labelledBy).toBeTruthy()
    expect(describedBy).toBeTruthy()
    expect(document.getElementById(labelledBy!)?.textContent).toBe('Excluir cliente')
    expect(document.getElementById(describedBy!)?.textContent).toBe('Tem certeza que deseja excluir?')
  })

  it('renders the default confirm label', async () => {
    mountDialog()
    await nextTick()

    const buttons = Array.from(document.querySelectorAll('button'))
    const confirmBtn = buttons.find(b => b.textContent?.trim() === 'Confirmar')
    expect(confirmBtn).toBeTruthy()
  })

  it('renders a custom confirm label', async () => {
    mountDialog({ confirmLabel: 'Excluir' })
    await nextTick()

    const buttons = Array.from(document.querySelectorAll('button'))
    expect(buttons.some(b => b.textContent?.trim() === 'Excluir')).toBe(true)
    expect(buttons.some(b => b.textContent?.trim() === 'Confirmar')).toBe(false)
  })

  it('emits confirm when the confirm button is clicked', async () => {
    mountDialog()
    await nextTick()

    const buttons = Array.from(document.querySelectorAll('button'))
    const confirmBtn = buttons.find(b => b.textContent?.trim() === 'Confirmar') as HTMLElement
    confirmBtn.click()
    await nextTick()

    expect(wrapper!.emitted('confirm')).toHaveLength(1)
  })

  it('emits cancel when the cancel button is clicked', async () => {
    mountDialog()
    await nextTick()

    const buttons = Array.from(document.querySelectorAll('button'))
    const cancelBtn = buttons.find(b => b.textContent?.trim() === 'Cancelar') as HTMLElement
    cancelBtn.click()
    await nextTick()

    expect(wrapper!.emitted('cancel')).toHaveLength(1)
  })

  it('emits cancel when the overlay (backdrop) is clicked', async () => {
    mountDialog()
    await nextTick()
    // The dismissable layer registers its outside-pointer listener a tick
    // after mount to avoid reacting to the click that opened it.
    await new Promise(resolve => setTimeout(resolve, 10))

    const overlay = document.querySelector('.fixed.inset-0.z-50') as HTMLElement
    expect(overlay).toBeTruthy()
    overlay.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true, button: 0 }))
    await nextTick()

    expect(wrapper!.emitted('cancel')).toHaveLength(1)
  })

  it('emits cancel on Escape key press', async () => {
    mountDialog()
    await nextTick()

    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    await nextTick()

    expect(wrapper!.emitted('cancel')).toHaveLength(1)
  })

  it('applies the destructive styling for the danger variant by default', async () => {
    mountDialog()
    await nextTick()

    const buttons = Array.from(document.querySelectorAll('button'))
    const confirmBtn = buttons.find(b => b.textContent?.trim() === 'Confirmar') as HTMLElement
    expect(confirmBtn.className).toContain('destructive')
  })

  it('does not apply destructive styling for the warning variant', async () => {
    mountDialog({ variant: 'warning' as const })
    await nextTick()

    const buttons = Array.from(document.querySelectorAll('button'))
    const confirmBtn = buttons.find(b => b.textContent?.trim() === 'Confirmar') as HTMLElement
    expect(confirmBtn.className).not.toContain('destructive')
  })

  it('renders default slot content inside the dialog', async () => {
    mountDialog({}, { default: '<label><input type="checkbox" /> Hard delete</label>' })
    await nextTick()

    const dialog = document.querySelector('[role="dialog"]') as HTMLElement
    expect(dialog.querySelector('input[type="checkbox"]')).not.toBeNull()
    expect(dialog.textContent).toContain('Hard delete')
  })
})
