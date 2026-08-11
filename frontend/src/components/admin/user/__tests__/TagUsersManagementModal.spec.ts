import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import TagUsersManagementModal from '../TagUsersManagementModal.vue'

const { getTagUsers, addUsersToTag, listUsers, showSuccess } = vi.hoisted(() => ({
  getTagUsers: vi.fn(),
  addUsersToTag: vi.fn(),
  listUsers: vi.fn(),
  showSuccess: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    tags: { getTagUsers, addUsersToTag },
    users: { list: listUsers }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showSuccess, showError: vi.fn() })
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) =>
      params ? `${key}:${JSON.stringify(params)}` : key
  })
}))

const tag = { id: 9, name: 'VIP', created_at: '', updated_at: '' }
const users = (ids: number[]) => ids.map((id) => ({
  id,
  email: `user${id}@example.com`,
  username: `user${id}`,
  status: 'active' as const,
  role: 'user' as const,
  balance: 0,
  concurrency: 1,
  allowed_groups: null,
  balance_notify_enabled: false,
  balance_notify_threshold: null,
  balance_notify_extra_emails: [],
  created_at: '',
  updated_at: '',
  notes: ''
}))

const mountModal = () => mount(TagUsersManagementModal, {
  props: { show: true, tag },
  global: {
    stubs: {
      BaseDialog: {
        props: ['show', 'title'],
        emits: ['close'],
        template: '<div v-if="show"><slot /></div>'
      },
      DataTable: {
        props: ['data', 'selectable', 'selectedKeys'],
        emits: ['update:selected-keys'],
        template: '<div><button v-if="selectable" data-test="select-page" @click="$emit(\'update:selected-keys\', [...selectedKeys, ...data.map(row => row.id)])" /><button v-if="selectable" data-test="select-many" @click="$emit(\'update:selected-keys\', Array.from({ length: 501 }, (_, index) => index + 1))" /><slot name="empty" /></div>'
      },
      Pagination: {
        props: ['total', 'page', 'pageSize'],
        emits: ['update:page', 'update:pageSize'],
        template: '<div><button data-test="next-page" @click="$emit(\'update:page\', page + 1)" /></div>'
      },
      Select: {
        props: ['modelValue', 'options'],
        emits: ['update:model-value'],
        template: '<button data-test="status" @click="$emit(\'update:model-value\', \'disabled\')">status</button>'
      }
    }
  }
})

describe('TagUsersManagementModal', () => {
  beforeEach(() => {
    getTagUsers.mockReset()
    addUsersToTag.mockReset()
    listUsers.mockReset()
    showSuccess.mockReset()
    getTagUsers.mockResolvedValue({ items: users([1]), total: 1, page: 1, page_size: 20, pages: 1 })
    listUsers.mockImplementation((page: number) => Promise.resolve({
      items: users(page === 1 ? [1] : [2]),
      total: 2,
      page,
      page_size: 20,
      pages: 2
    }))
    addUsersToTag.mockResolvedValue({ affected: 2 })
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('loads members and passes search/status/page filters', async () => {
    vi.useFakeTimers()
    const wrapper = mountModal()
    await wrapper.setProps({ show: false })
    await wrapper.setProps({ show: true })
    await flushPromises()

    expect(getTagUsers).toHaveBeenCalledWith(9, 1, 20, { search: undefined, status: undefined })
    await wrapper.get('input[type="search"]').setValue('alice')
    await wrapper.get('input[type="search"]').trigger('input')
    vi.advanceTimersByTime(250)
    await flushPromises()
    expect(getTagUsers).toHaveBeenLastCalledWith(9, 1, 20, { search: 'alice', status: undefined })

    await wrapper.get('[data-test="status"]').trigger('click')
    await flushPromises()
    expect(getTagUsers).toHaveBeenLastCalledWith(9, 1, 20, { search: 'alice', status: 'disabled' })
  })

  it('preserves selections across pages and submits only selected users', async () => {
    vi.useFakeTimers()
    const wrapper = mountModal()
    await flushPromises()
    await wrapper.get('button').trigger('click')
    await flushPromises()
    expect(listUsers).toHaveBeenCalledWith(1, 20, { search: undefined, status: 'active' })

    await wrapper.get('[data-test="select-page"]').trigger('click')
    await wrapper.get('[data-test="next-page"]').trigger('click')
    await flushPromises()
    expect(listUsers).toHaveBeenLastCalledWith(2, 20, { search: undefined, status: 'active' })
    expect(wrapper.text()).toContain('admin.users.selectedUsers:{"count":1}')

    await wrapper.get('[data-test="select-page"]').trigger('click')
    expect(wrapper.text()).toContain('admin.users.selectedUsers:{"count":2}')
    const submit = wrapper.get('button.btn-primary:last-of-type')
    await submit.trigger('click')
    await flushPromises()
    expect(addUsersToTag).toHaveBeenCalledWith(9, [1, 2])
    expect(showSuccess).toHaveBeenCalledWith(expect.stringContaining('2'))
    expect(wrapper.emitted('success')).toHaveLength(1)
    vi.useRealTimers()
  })

  it('does not submit when selection exceeds 500 users', async () => {
    const wrapper = mountModal()
    await flushPromises()
    await wrapper.get('button').trigger('click')
    await flushPromises()
    await wrapper.get('[data-test="select-many"]').trigger('click')
    const submit = wrapper.get('button.btn-primary:last-of-type')
    expect(submit.attributes('disabled')).toBeDefined()
    expect(addUsersToTag).not.toHaveBeenCalled()
  })
})
