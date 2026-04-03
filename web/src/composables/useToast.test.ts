import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { useToast } from './useToast'

describe('useToast', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    // Clear any leftover toasts from previous tests
    const { toasts } = useToast()
    toasts.value = []
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('starts with an empty toast list', () => {
    const { toasts } = useToast()
    expect(toasts.value).toEqual([])
  })

  it('success() adds a toast with type "success"', () => {
    const { success, toasts } = useToast()
    success('Item criado')

    expect(toasts.value).toHaveLength(1)
    expect(toasts.value[0].type).toBe('success')
    expect(toasts.value[0].message).toBe('Item criado')
    expect(toasts.value[0].duration).toBe(3000)
    expect(toasts.value[0].id).toBeTruthy()
  })

  it('error() adds a toast with type "error"', () => {
    const { error, toasts } = useToast()
    error('Falha na operação')

    expect(toasts.value).toHaveLength(1)
    expect(toasts.value[0].type).toBe('error')
    expect(toasts.value[0].message).toBe('Falha na operação')
  })

  it('warning() adds a toast with type "warning"', () => {
    const { warning, toasts } = useToast()
    warning('Atenção')

    expect(toasts.value).toHaveLength(1)
    expect(toasts.value[0].type).toBe('warning')
    expect(toasts.value[0].message).toBe('Atenção')
  })

  it('generates unique IDs for each toast', () => {
    const { success, toasts } = useToast()
    success('First')
    success('Second')

    expect(toasts.value).toHaveLength(2)
    expect(toasts.value[0].id).not.toBe(toasts.value[1].id)
  })

  it('auto-dismisses toast after its duration', () => {
    const { success, toasts } = useToast()
    success('Temporary')

    expect(toasts.value).toHaveLength(1)

    vi.advanceTimersByTime(3000)

    expect(toasts.value).toHaveLength(0)
  })

  it('removeToast() manually removes a toast by id', () => {
    const { success, removeToast, toasts } = useToast()
    success('To be removed')

    const id = toasts.value[0].id
    removeToast(id)

    expect(toasts.value).toHaveLength(0)
  })

  it('removeToast() does not affect other toasts', () => {
    const { success, error, removeToast, toasts } = useToast()
    success('Keep me')
    error('Remove me')

    const errorId = toasts.value[1].id
    removeToast(errorId)

    expect(toasts.value).toHaveLength(1)
    expect(toasts.value[0].message).toBe('Keep me')
  })

  it('shares the same toast list across multiple useToast() calls (singleton)', () => {
    const toast1 = useToast()
    const toast2 = useToast()

    toast1.success('From instance 1')

    expect(toast2.toasts.value).toHaveLength(1)
    expect(toast2.toasts.value[0].message).toBe('From instance 1')
  })

  it('multiple toasts auto-dismiss independently', () => {
    const { success, warning, toasts } = useToast()

    success('First')
    vi.advanceTimersByTime(1000)
    warning('Second')

    expect(toasts.value).toHaveLength(2)

    // First toast should dismiss at 3000ms from start
    vi.advanceTimersByTime(2000)
    expect(toasts.value).toHaveLength(1)
    expect(toasts.value[0].message).toBe('Second')

    // Second toast should dismiss at 4000ms from start (1000 + 3000)
    vi.advanceTimersByTime(1000)
    expect(toasts.value).toHaveLength(0)
  })
})
