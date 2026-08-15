import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const apiMocks = vi.hoisted(() => ({
  getAllGroups: vi.fn(),
  getPriceMonitor: vi.fn(),
  updatePriceMonitor: vi.fn(),
  listAnnouncements: vi.fn()
}))

const storeMocks = vi.hoisted(() => ({
  showError: vi.fn(),
  showSuccess: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    groups: { getAll: apiMocks.getAllGroups },
    announcements: {
      getPriceMonitor: apiMocks.getPriceMonitor,
      updatePriceMonitor: apiMocks.updatePriceMonitor,
      list: apiMocks.listAnnouncements,
      create: vi.fn(),
      update: vi.fn(),
      delete: vi.fn()
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => storeMocks
}))

vi.mock('@/composables/usePersistedPageSize', () => ({
  getPersistedPageSize: () => 20
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

import AnnouncementsView from '../AnnouncementsView.vue'

describe('AnnouncementsView group monitor', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    apiMocks.getAllGroups.mockResolvedValue([])
    apiMocks.getPriceMonitor.mockResolvedValue({
      enabled: false,
      group_ids: [],
      interval_seconds: 60,
      status: 'active',
      notify_mode: 'popup',
      duration_days: 3,
      targeting: { any_of: [] }
    })
    apiMocks.listAnnouncements.mockResolvedValue({
      items: [],
      total: 0,
      pages: 0,
      page: 1,
      page_size: 20
    })
    apiMocks.updatePriceMonitor.mockImplementation(async (config) => config)
  })

  it('defaults legacy config to price and requires at least one change type', async () => {
    const wrapper = mount(AnnouncementsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>' },
          DataTable: true,
          Pagination: true,
          BaseDialog: {
            props: ['show', 'title'],
            template: '<section v-if="show"><h2>{{ title }}</h2><slot /><slot name="footer" /></section>'
          },
          ConfirmDialog: true,
          Select: true,
          EmptyState: true,
          Icon: true,
          GroupSelector: true,
          AnnouncementTargetingEditor: true,
          AnnouncementReadStatusDialog: true,
          AnnouncementPopup: true
        }
      }
    })
    await flushPromises()

    const openButton = wrapper.findAll('button').find((button) => button.text().includes('分组变更监测'))
    expect(openButton).toBeDefined()
    await openButton!.trigger('click')

    const priceCheckbox = wrapper.get<HTMLInputElement>('input[type="checkbox"][value="price"]')
    const statusCheckbox = wrapper.get<HTMLInputElement>('input[type="checkbox"][value="status"]')
    expect(priceCheckbox.element.checked).toBe(true)
    expect(statusCheckbox.element.checked).toBe(false)

    await priceCheckbox.setValue(false)
    const saveButton = wrapper.findAll('button').find((button) => button.text().includes('保存监测设置'))
    expect(saveButton).toBeDefined()
    await saveButton!.trigger('click')
    expect(storeMocks.showError).toHaveBeenCalledWith('监测内容最少选择一项')
    expect(apiMocks.updatePriceMonitor).not.toHaveBeenCalled()

    await statusCheckbox.setValue(true)
    await saveButton!.trigger('click')
    await flushPromises()
    expect(apiMocks.updatePriceMonitor).toHaveBeenCalledWith(expect.objectContaining({
      change_types: ['status']
    }))
  })
})
