import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import GroupOptionItem from '../GroupOptionItem.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ cachedPublicSettings: null }),
}))

describe('GroupOptionItem text layout', () => {
  it('applies multiline and overflow-safe description styles', () => {
    const description = 'First section\nvery-long-unbroken-description-value-that-must-not-overflow'
    const wrapper = mount(GroupOptionItem, {
      props: {
        name: 'Example group',
        platform: 'openai',
        description,
      },
      global: {
        stubs: {
          GroupBadge: true,
        },
      },
    })

    const descriptionElement = wrapper
      .findAll('span')
      .find((element) => element.text() === description)

    expect(descriptionElement).toBeDefined()
    expect(descriptionElement?.classes()).toContain('whitespace-pre-line')
    expect(descriptionElement?.classes()).toContain('[overflow-wrap:anywhere]')
    expect(descriptionElement?.classes()).toContain('line-clamp-2')
    expect(wrapper.find('[title]').attributes('title')).toBe(description)
  })

  it('keeps limited-time multiplier text on one line', () => {
    const wrapper = mount(GroupOptionItem, {
      props: {
        name: 'Example group',
        platform: 'openai',
        limitedTimeMultiplier: '2x · 每日 · 09:00-10:00',
      },
      global: {
        stubs: {
          GroupBadge: true,
        },
      },
    })

    const multiplierElement = wrapper
      .findAll('span')
      .find((element) => element.text() === '2x · 每日 · 09:00-10:00')

    expect(multiplierElement).toBeDefined()
    expect(multiplierElement?.classes()).toContain('truncate')
    expect(multiplierElement?.classes()).not.toContain('whitespace-normal')
  })

  it.each([
    [true, 0, undefined, 'admin.groups.exclusive'],
    [true, 1, undefined, 'admin.groups.tagExclusive'],
    [true, 1, ['Downstream'], 'Downstreamadmin.groups.exclusive'],
    [true, 2, ['Downstream', 'VIP'], 'Downstream、VIPadmin.groups.exclusive'],
  ])('shows the authorization type for exclusive groups', (isExclusive, tagCount, tagNames, label) => {
    const wrapper = mount(GroupOptionItem, {
      props: {
        name: 'Example group',
        platform: 'openai',
        isExclusive,
        authorizationTagCount: tagCount,
        authorizationTagNames: tagNames,
      },
      global: {
        stubs: {
          GroupBadge: true,
        },
      },
    })

    expect(wrapper.get('[data-testid="group-authorization-type"]').text()).toBe(label)
  })

  it('does not show an authorization type for public groups', () => {
    const wrapper = mount(GroupOptionItem, {
      props: {
        name: 'Example group',
        platform: 'openai',
        isExclusive: false,
        authorizationTagCount: 1,
      },
      global: {
        stubs: {
          GroupBadge: true,
        },
      },
    })

    expect(wrapper.find('[data-testid="group-authorization-type"]').exists()).toBe(false)
  })
})
