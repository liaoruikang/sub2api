import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

vi.mock('@/api/admin', () => ({
  adminAPI: {
    groups: {
      list: vi.fn().mockResolvedValue({
        items: [
          {
            id: 10,
            name: 'Downstream Group',
            platform: 'openai',
            is_exclusive: true,
            subscription_type: 'standard',
            status: 'active',
            rate_multiplier: 1,
            tag_ids: [7],
            tags: [{ id: 7, name: 'downstream' }]
          }
        ]
      })
    },
    tags: {
      getUserTags: vi.fn().mockResolvedValue([{ id: 7, name: 'downstream' }]),
      list: vi.fn().mockResolvedValue([{ id: 7, name: 'downstream' }])
    },
    users: {
      update: vi.fn()
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showSuccess: vi.fn()
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key === 'admin.groups.exclusive' ? 'exclusive' : key
    })
  }
})

import UserAllowedGroupsModal from '../UserAllowedGroupsModal.vue'

describe('UserAllowedGroupsModal', () => {
  it('formats a tag-derived exclusive group without a visible plus sign', async () => {
    const wrapper = mount(UserAllowedGroupsModal, {
      props: {
        show: false,
        user: {
          id: 42,
          email: 'user@example.com',
          allowed_groups: [],
          tags: [{ id: 7, name: 'downstream' }]
        } as never
      },
      global: {
        stubs: {
          BaseDialog: {
            template: '<div><slot /><slot name="footer" /></div>'
          },
          PlatformIcon: true
        }
      }
    })

    await wrapper.setProps({ show: true })
    await flushPromises()

    const authorizationBadge = wrapper.get('[data-testid="group-authorization-type"]')
    expect(authorizationBadge.text()).toBe('downstreamexclusive')
    expect(wrapper.text()).not.toContain('downstream + exclusive')
  })
})
