import { mount } from '@vue/test-utils'
import { defineComponent, nextTick } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import GroupSelector from '@/components/common/GroupSelector.vue'
import type { Group } from '@/types'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

const makeGroup = (id: number, platform: Group['platform'] = 'openai') => ({
  id,
  name: `Group ${id}`,
  platform,
  subscription_type: 'standard',
  rate_multiplier: 1,
  description: null
} as Group)

describe('GroupSelector multi-select mode', () => {
  it('keeps all candidates when no platform filter is provided', () => {
    const wrapper = mount(GroupSelector, {
      props: {
        modelValue: [],
        groups: [makeGroup(1, 'openai'), makeGroup(2, 'anthropic')],
        searchable: false
      },
      global: {
        stubs: {
          GroupBadge: true,
          GroupOptionItem: true,
          Icon: true
        }
      }
    })

    expect(wrapper.findAll('input[type="checkbox"]')).toHaveLength(2)
  })

  it('filters candidates when a platform filter is provided', () => {
    const wrapper = mount(GroupSelector, {
      props: {
        modelValue: [],
        groups: [makeGroup(1, 'openai'), makeGroup(2, 'anthropic')],
        platform: 'openai',
        searchable: false
      },
      global: {
        stubs: {
          GroupBadge: true,
          GroupOptionItem: true,
          Icon: true
        }
      }
    })

    expect(wrapper.findAll('input[type="checkbox"]')).toHaveLength(1)
  })

  it('appends checked groups in selection order and removes them without mutating the prop', async () => {
    const modelValue = [2]
    const wrapper = mount(GroupSelector, {
      props: {
        modelValue,
        groups: [makeGroup(1), makeGroup(2)],
        ordered: true,
        searchable: false
      },
      global: {
        stubs: {
          GroupBadge: true,
          GroupOptionItem: true,
          Icon: true
        }
      }
    })

    await wrapper.findAll('input[type="checkbox"]')[0].setValue(true)
    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toEqual([2, 1])
    expect(modelValue).toEqual([2])
    expect(wrapper.findAll('label').some((label) => label.text().endsWith('0'))).toBe(false)

    await wrapper.setProps({ modelValue: [2, 1] })
    await wrapper.findAll('input[type="checkbox"]')[1].setValue(false)
    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toEqual([1])
  })

  it('keeps the numbered draggable list while using checkboxes for selection', async () => {
    const modelValue = [2]
    const draggable = defineComponent({
      props: { modelValue: { type: Array, required: true } },
      emits: ['update:modelValue', 'end'],
      template: `
        <div data-test="selected-draggable">
          <slot />
          <button data-test="simulate-drag" @click="$emit('update:modelValue', [1, 2])" />
          <button data-test="simulate-drag-end" @click="$emit('end')" />
        </div>
      `
    })
    const wrapper = mount(GroupSelector, {
      props: {
        modelValue,
        groups: [makeGroup(1), makeGroup(2)],
        ordered: true,
        searchable: false
      },
      global: {
        stubs: {
          GroupBadge: true,
          GroupOptionItem: true,
          Icon: true,
          VueDraggable: draggable
        }
      }
    })

    expect(wrapper.find('[data-test="selected-draggable"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('1')
    expect(wrapper.findAll('input[type="checkbox"]')).toHaveLength(2)
    expect(wrapper.findAll('label').some((label) => label.text().endsWith('0'))).toBe(false)

    await wrapper.get('[data-test="simulate-drag"]').trigger('click')
    await wrapper.get('[data-test="simulate-drag-end"]').trigger('click')
    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toEqual([1, 2])
    expect(modelValue).toEqual([2])
  })

  it('can collapse and reopen a default-open candidate panel', async () => {
    const wrapper = mount(GroupSelector, {
      props: {
        modelValue: [2],
        groups: [makeGroup(1), makeGroup(2)],
        searchable: false,
        collapsible: true,
        defaultOpen: true
      },
      global: {
        stubs: {
          GroupBadge: true,
          GroupOptionItem: true,
          Icon: true
        }
      }
    })

    const trigger = wrapper.get('button[aria-haspopup="listbox"]')
    expect(trigger.attributes('aria-expanded')).toBe('true')
    expect(wrapper.findAll('input[type="checkbox"]')).toHaveLength(2)

    await trigger.trigger('click')
    expect(trigger.attributes('aria-expanded')).toBe('false')
    expect(wrapper.findAll('input[type="checkbox"]')).toHaveLength(0)

    await trigger.trigger('click')
    expect(trigger.attributes('aria-expanded')).toBe('true')
    expect(wrapper.findAll('input[type="checkbox"]')).toHaveLength(2)
  })

  it('passes user-specific rates and authorization types to selected and candidate options', () => {
    const option = defineComponent({
      props: {
        name: { type: String, required: true },
        userRateMultiplier: { type: Number, default: null },
        isExclusive: { type: Boolean, default: false },
        authorizationTagCount: { type: Number, default: 0 },
        authorizationTagNames: { type: Array, default: () => [] }
      },
      template: '<div data-test="option">{{ name }}:{{ userRateMultiplier }}:{{ isExclusive }}:{{ authorizationTagCount }}:{{ authorizationTagNames.join(",") }}</div>'
    })
    const taggedGroup = {
      ...makeGroup(1),
      is_exclusive: true,
      tag_ids: [5],
      tags: [{ id: 5, name: 'Downstream', created_at: '', updated_at: '' }]
    }
    const exclusiveGroupWithoutLoadedTags = {
      ...makeGroup(2),
      is_exclusive: true,
      tag_ids: [6],
      tags: undefined
    }
    const wrapper = mount(GroupSelector, {
      props: {
        modelValue: [2],
        groups: [taggedGroup, exclusiveGroupWithoutLoadedTags],
        ordered: true,
        searchable: false,
        userGroupRates: { 1: 0.8, 2: 0.6 }
      },
      global: {
        stubs: {
          GroupBadge: true,
          GroupOptionItem: option,
          Icon: true,
          VueDraggable: defineComponent({
            props: { modelValue: { type: Array, required: true } },
            template: '<div><slot /></div>'
          })
        }
      }
    })

    expect(wrapper.findAll('[data-test="option"]').map((item) => item.text())).toEqual([
      'Group 2:0.6:true:1:',
      'Group 1:0.8:true:1:Downstream',
      'Group 2:0.6:true:1:'
    ])
  })

  it('shows candidates in a dropdown while keeping the ordered selection visible', async () => {
    const wrapper = mount(GroupSelector, {
      props: {
        modelValue: [2],
        groups: [makeGroup(1), makeGroup(2)],
        ordered: true,
        dropdown: true,
        searchable: false
      },
      global: {
        stubs: {
          GroupBadge: true,
          GroupOptionItem: true,
          Icon: true,
          VueDraggable: defineComponent({
            props: { modelValue: { type: Array, required: true } },
            emits: ['update:modelValue', 'end'],
            template: '<div data-test="selected-draggable"><slot /></div>'
          })
        }
      },
      attachTo: document.body
    })

    expect(wrapper.find('button[aria-haspopup="listbox"]').exists()).toBe(true)
    expect(document.body.querySelector('[role="listbox"]')).toBeNull()
    expect(wrapper.findAll('input[type="checkbox"]')).toHaveLength(0)

    await wrapper.get('button[aria-haspopup="listbox"]').trigger('click')
    await nextTick()

    const dropdown = document.body.querySelector('[role="listbox"]')
    expect(dropdown).not.toBeNull()
    expect(dropdown?.querySelectorAll('input[type="checkbox"]')).toHaveLength(2)
    expect(wrapper.find('[data-test="selected-draggable"]').exists()).toBe(true)

    const checkbox = dropdown?.querySelector('input[type="checkbox"]') as HTMLInputElement
    checkbox.checked = true
    checkbox.dispatchEvent(new Event('change', { bubbles: true }))
    await nextTick()
    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toEqual([2, 1])

    await wrapper.get('button[aria-haspopup="listbox"]').trigger('click')
    expect(document.body.querySelector('[role="listbox"]')).toBeNull()
    wrapper.unmount()
  })
})
