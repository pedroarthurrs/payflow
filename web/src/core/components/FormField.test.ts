import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import FormField from './FormField.vue'

describe('FormField', () => {
  it('renders a text input with label', () => {
    const wrapper = mount(FormField, {
      props: { label: 'Nome', modelValue: 'João' },
    })
    const label = wrapper.find('label')
    expect(label.text()).toBe('Nome')
    const input = wrapper.find('input')
    expect(input.exists()).toBe(true)
    expect(input.element.type).toBe('text')
    expect(input.element.value).toBe('João')
  })

  it('links label to input via for/id', () => {
    const wrapper = mount(FormField, {
      props: { label: 'Email', modelValue: '' },
    })
    const label = wrapper.find('label')
    const input = wrapper.find('input')
    expect(label.attributes('for')).toBe(input.attributes('id'))
  })

  it('emits update:modelValue on text input', async () => {
    const wrapper = mount(FormField, {
      props: { label: 'Nome', modelValue: '' },
    })
    await wrapper.find('input').setValue('Test')
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual(['Test'])
  })

  it('renders required indicator when required is true', () => {
    const wrapper = mount(FormField, {
      props: { label: 'Nome', modelValue: '', required: true },
    })
    expect(wrapper.find('.form-field__required').text()).toBe('*')
  })

  it('does not render required indicator when required is false', () => {
    const wrapper = mount(FormField, {
      props: { label: 'Nome', modelValue: '' },
    })
    expect(wrapper.find('.form-field__required').exists()).toBe(false)
  })

  it('renders error message when error prop is set', () => {
    const wrapper = mount(FormField, {
      props: { label: 'Email', modelValue: '', error: 'Email inválido' },
    })
    const errorEl = wrapper.find('.form-field__error')
    expect(errorEl.exists()).toBe(true)
    expect(errorEl.text()).toBe('Email inválido')
  })

  it('does not render error message when error prop is empty', () => {
    const wrapper = mount(FormField, {
      props: { label: 'Email', modelValue: '' },
    })
    expect(wrapper.find('.form-field__error').exists()).toBe(false)
  })

  it('renders a select with options', () => {
    const wrapper = mount(FormField, {
      props: {
        label: 'Tipo',
        modelValue: 'CPF',
        type: 'select',
        options: [
          { value: 'CPF', label: 'CPF' },
          { value: 'CNPJ', label: 'CNPJ' },
        ],
      },
    })
    const select = wrapper.find('select')
    expect(select.exists()).toBe(true)
    const options = wrapper.findAll('option')
    expect(options).toHaveLength(2)
    expect(options[0].text()).toBe('CPF')
    expect(options[1].text()).toBe('CNPJ')
  })

  it('renders select with placeholder option', () => {
    const wrapper = mount(FormField, {
      props: {
        label: 'Tipo',
        modelValue: '',
        type: 'select',
        placeholder: 'Selecione...',
        options: [{ value: 'a', label: 'A' }],
      },
    })
    const options = wrapper.findAll('option')
    expect(options[0].text()).toBe('Selecione...')
    expect(options[0].attributes('disabled')).toBeDefined()
  })

  it('emits update:modelValue on select change', async () => {
    const wrapper = mount(FormField, {
      props: {
        label: 'Tipo',
        modelValue: 'CPF',
        type: 'select',
        options: [
          { value: 'CPF', label: 'CPF' },
          { value: 'CNPJ', label: 'CNPJ' },
        ],
      },
    })
    await wrapper.find('select').setValue('CNPJ')
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual(['CNPJ'])
  })

  it('renders a textarea', () => {
    const wrapper = mount(FormField, {
      props: { label: 'Metadata', modelValue: '{}', type: 'textarea' },
    })
    const textarea = wrapper.find('textarea')
    expect(textarea.exists()).toBe(true)
    expect(textarea.element.value).toBe('{}')
  })

  it('emits update:modelValue on textarea input', async () => {
    const wrapper = mount(FormField, {
      props: { label: 'Metadata', modelValue: '', type: 'textarea' },
    })
    await wrapper.find('textarea').setValue('{"key":"val"}')
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual(['{"key":"val"}'])
  })

  it('renders a checkbox with inline label', () => {
    const wrapper = mount(FormField, {
      props: { label: 'Ativo', modelValue: false, type: 'checkbox' },
    })
    const checkbox = wrapper.find('input[type="checkbox"]')
    expect(checkbox.exists()).toBe(true)
    expect((checkbox.element as HTMLInputElement).checked).toBe(false)
    const label = wrapper.find('.form-field__label--inline')
    expect(label.text()).toContain('Ativo')
  })

  it('emits update:modelValue with boolean on checkbox change', async () => {
    const wrapper = mount(FormField, {
      props: { label: 'Ativo', modelValue: false, type: 'checkbox' },
    })
    await wrapper.find('input[type="checkbox"]').setValue(true)
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([true])
  })

  it('renders email input type', () => {
    const wrapper = mount(FormField, {
      props: { label: 'Email', modelValue: '', type: 'email' },
    })
    expect(wrapper.find('input').element.type).toBe('email')
  })

  it('renders password input type', () => {
    const wrapper = mount(FormField, {
      props: { label: 'Senha', modelValue: '', type: 'password' },
    })
    expect(wrapper.find('input').element.type).toBe('password')
  })

  it('applies error class when error is present', () => {
    const wrapper = mount(FormField, {
      props: { label: 'Nome', modelValue: '', error: 'Obrigatório' },
    })
    expect(wrapper.find('.form-field--error').exists()).toBe(true)
  })

  it('renders placeholder on text input', () => {
    const wrapper = mount(FormField, {
      props: { label: 'Nome', modelValue: '', placeholder: 'Digite...' },
    })
    expect(wrapper.find('input').attributes('placeholder')).toBe('Digite...')
  })
})
