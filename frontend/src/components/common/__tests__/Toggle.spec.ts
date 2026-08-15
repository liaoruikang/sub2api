import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Toggle from '../Toggle.vue'

describe('Toggle', () => {
  it('emits a changed value when enabled', async () => {
    const wrapper = mount(Toggle, { props: { modelValue: false } })

    await wrapper.get('button').trigger('click')

    expect(wrapper.emitted('update:modelValue')).toEqual([[true]])
  })

  it('does not emit when disabled', async () => {
    const wrapper = mount(Toggle, { props: { modelValue: false, disabled: true } })

    await wrapper.get('button').trigger('click')

    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
    expect(wrapper.get('button').attributes('disabled')).toBeDefined()
  })
})
